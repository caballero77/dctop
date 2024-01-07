package docker

import (
	"context"
	"dctop/internal/utils/maps"
	"dctop/internal/utils/slices"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"gopkg.in/yaml.v3"
)

var stackLabel = "com.docker.compose.project"

type ComposeService struct {
	cli             *client.Client
	ctx             context.Context
	containers      map[string]struct{}
	stack           string
	composeFilePath string
	compose         Compose

	containerUpdates    chan ContainerMsg
	unsubscribeChannels map[string]chan struct{}
	stopSyncronization  chan struct{}
}

func ProvideComposeService(composeFilePath string) (*ComposeService, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	stack, compose, containers, err := getStack(ctx, cli, composeFilePath)
	if err != nil {
		return nil, err
	}

	containerIds := make(map[string]struct{}, len(containers))
	for _, container := range containers {
		containerIds[container.ID] = struct{}{}
	}

	service := &ComposeService{
		cli:                 cli,
		ctx:                 ctx,
		stack:               stack,
		containers:          containerIds,
		containerUpdates:    nil,
		composeFilePath:     composeFilePath,
		compose:             compose,
		unsubscribeChannels: make(map[string]chan struct{}),
	}

	return service, nil
}

func (service ComposeService) Stack() string {
	return service.stack
}

func (service ComposeService) ComposeFilePath() string {
	return service.composeFilePath
}

func (service ComposeService) Close() error {
	return service.cli.Close()
}

func (service ComposeService) ContainerPause(id string) error {
	return service.cli.ContainerPause(service.ctx, id)
}

func (service ComposeService) ContainerUnpause(id string) error {
	return service.cli.ContainerUnpause(service.ctx, id)
}

func (service ComposeService) ContainerStop(id string) error {
	return service.cli.ContainerStop(service.ctx, id, container.StopOptions{})
}

func (service ComposeService) ContainerStart(id string) error {
	return service.cli.ContainerStart(service.ctx, id, types.ContainerStartOptions{})
}

func (service ComposeService) ComposeDown() error {
	cmd := exec.Command("docker-compose", "-f", service.composeFilePath, "down") // #nosec G204
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (service ComposeService) ComposeUp() error {
	cmd := exec.Command("docker-compose", "-f", service.composeFilePath, "up", "-d") // #nosec G204
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (service ComposeService) GetContainerLogs(id, tail string) (stdout, stderr chan []byte, e chan error, done chan bool) {
	done = make(chan bool)
	e = make(chan error)
	stdout, stderr = make(chan []byte), make(chan []byte)

	go func() {
		reader, err := service.cli.ContainerLogs(service.ctx, id, types.ContainerLogsOptions{
			ShowStderr: true,
			ShowStdout: true,
			Timestamps: false,
			Tail:       tail,
			Follow:     true,
		})
		if err != nil {
			e <- err
			return
		}
		hdr := make([]byte, 8)
		for {
			select {
			case <-done:
				return
			default:
				_, err := reader.Read(hdr)
				if err != nil {
					e <- err
					return
				}
				count := binary.BigEndian.Uint32(hdr[4:])
				dat := make([]byte, count)
				_, err = reader.Read(dat)
				if err != nil {
					e <- err
					return
				}

				switch hdr[0] {
				case 1:
					stdout <- dat
				default:
					stderr <- dat
				}
			}
		}
	}()

	return stdout, stderr, e, done
}

func parseProcesses(top container.ContainerTopOKBody) []Process {
	titles := make(map[string]int, len(top.Titles))
	for i := 0; i < len(top.Titles); i++ {
		titles[top.Titles[i]] = i
	}

	result := make([]Process, len(top.Processes))
	for i, process := range top.Processes {
		result[i] = Process{
			PID:     process[titles["PID"]],
			PPID:    process[titles["PPID"]],
			Threads: process[titles["THCNT"]],
			RSS:     process[titles["RSS"]],
			CPU:     process[titles["%CPU"]],
			CMD:     process[titles["CMD"]],
		}
	}
	return result
}

func (service ComposeService) getContainerInfo(containerID string) (types.ContainerJSON, []Process, error) {
	inspectResponse, err := service.cli.ContainerInspect(service.ctx, containerID)
	if err != nil {
		return inspectResponse, nil, err
	}
	processes := make([]Process, 0)
	if inspectResponse.State.Status != "exited" {
		top, err := service.cli.ContainerTop(service.ctx, containerID, []string{"-eo", "pid,ppid,thcount,rss,%cpu,cmd"})
		if err != nil {
			return inspectResponse, nil, err
		}

		processes = parseProcesses(top)
	}

	return inspectResponse, processes, nil
}

func (service *ComposeService) GetContainerUpdates() (chan ContainerMsg, error) {
	if service.containerUpdates != nil {
		return service.containerUpdates, nil
	}

	service.containerUpdates = make(chan ContainerMsg, 10000)

	err := service.startListenningForStatistic()
	if err != nil {
		return nil, err
	}

	service.stopSyncronization = make(chan struct{})

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		for {
			select {
			case <-ticker.C:
				err := service.syncContainers()
				if err != nil {
					continue
				}
			case <-service.stopSyncronization:
				return
			}
		}
	}()

	return service.containerUpdates, nil
}

func (service ComposeService) startListenningForStatistic() error {
	for id := range service.containers {
		err := service.startLiseningForUpdates(id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (service *ComposeService) startLiseningForUpdates(containerID string) error {
	done := make(chan struct{})

	service.unsubscribeChannels[containerID] = done

	statisticsResponse, err := service.cli.ContainerStats(service.ctx, containerID, true)
	if err != nil {
		return err
	}
	var newStats ContainerStats
	decoder := json.NewDecoder(statisticsResponse.Body)

	go func() {
		service.containerUpdates <- ContainerCreateMsg{ID: containerID}
		for {
			fmt.Sprintln(containerID)
			select {
			case <-done:
				statisticsResponse.Body.Close()
				return
			case <-service.ctx.Done():
				statisticsResponse.Body.Close()
				return
			default:
				if err := decoder.Decode(&newStats); errors.Is(err, io.EOF) {
					return
				} else if err != nil {
					panic(err)
				}

				if newStats.Read.IsZero() {
					service.removeContainer(containerID)
					return
				}

				inspectResponse, processes, err := service.getContainerInfo(containerID)
				if err != nil {
					fmt.Println(newStats)
					continue
				}

				service.containerUpdates <- ContainerUpdateMsg{
					ID:        containerID,
					Inspect:   inspectResponse,
					Stats:     newStats,
					Processes: processes,
				}
			}
		}
	}()

	return nil
}

func (service ComposeService) syncContainers() error {
	containers, err := service.cli.ContainerList(service.ctx,
		types.ContainerListOptions{
			All: true,
			Filters: filters.NewArgs(
				filters.KeyValuePair{Key: "label", Value: fmt.Sprintf("%s=%s", stackLabel, service.stack)},
			),
		})
	if err != nil {
		return err
	}

	existingContainers := make(map[string]struct{})

	for _, container := range containers {
		existingContainers[container.ID] = struct{}{}
		if _, ok := service.containers[container.ID]; !ok {
			service.containers[container.ID] = struct{}{}
			err := service.startLiseningForUpdates(container.ID)
			if err != nil {
				return err
			}
		}
	}

	for id := range service.containers {
		if _, ok := existingContainers[id]; !ok {
			service.removeContainer(id)
		}
	}

	return nil
}

func (service *ComposeService) removeContainer(id string) {
	close(service.unsubscribeChannels[id])
	service.containerUpdates <- ContainerRemoveMsg{ID: id}
	delete(service.containers, id)
}

func getStack(ctx context.Context, cli *client.Client, composeFilePath string) (stack string, compose Compose, containers []types.Container, err error) {
	stack = filepath.Base(filepath.Dir(composeFilePath))

	containers, err = cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return "", compose, nil, err
	}

	bytes, err := os.ReadFile(composeFilePath)
	if err != nil {
		return "", compose, nil, err
	}

	err = yaml.Unmarshal(bytes, &compose)
	if err != nil {
		return "", compose, nil, err
	}

	filteredByStack := slices.Filter(containers, func(container types.Container) bool { return container.Labels[stackLabel] == stack })

	if len(filteredByStack) != 0 {
		return stack, compose, filteredByStack, nil
	}

	filter := func(container types.Container) bool {
		if containerStack, ok := container.Labels["com.docker.compose.project"]; ok {
			return containerStack == stack
		}

		parts := strings.Split(container.Names[0], "-")

		if len(parts) > 1 {
			if parts[0] == stack {
				return true
			}

			name := strings.Join(parts[1:], "-")
			if _, err := slices.Find(maps.Keys(compose.Services), func(key string) bool { return key == name }); err == nil {
				return true
			}
		}

		if _, err := slices.Find(maps.Values(compose.Services), func(service Service) bool { return service.Name == container.Names[0] }); err == nil {
			return true
		}

		return false
	}

	filteredByStack = slices.Filter(containers, filter)

	return stack, compose, filteredByStack, nil
}
