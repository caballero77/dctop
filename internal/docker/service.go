package docker

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"

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
}

func ProvideComposeService(composeFilePath string) (*ComposeService, error) {
	slog.Info("Creating docker client")
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		slog.Error("Error creating docker client",
			"Error", err)
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
	slog.Debug("Compose service has stoped")

	for _, unsubscribe := range service.unsubscribeChannels {
		close(unsubscribe)
	}
	return service.cli.Close()
}

func (service ComposeService) ContainerPause(id string) error {
	slog.Debug("Pausing container",
		"Id", id)

	return service.cli.ContainerPause(service.ctx, id)
}

func (service ComposeService) ContainerUnpause(id string) error {
	slog.Debug("Unpausing container",
		"Id", id)

	return service.cli.ContainerUnpause(service.ctx, id)
}

func (service ComposeService) ContainerStop(id string) error {
	slog.Debug("Stoping container",
		"Id", id)

	return service.cli.ContainerStop(service.ctx, id, container.StopOptions{})
}

func (service ComposeService) ContainerStart(id string) error {
	slog.Debug("Starting container",
		"Id", id)

	return service.cli.ContainerStart(service.ctx, id, types.ContainerStartOptions{})
}

func (service ComposeService) ComposeDown() error {
	slog.Debug("Executing down comand on compose file")

	cmd := exec.Command("docker-compose", "-f", service.composeFilePath, "down") // #nosec G204
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (service ComposeService) ComposeUp() error {
	slog.Debug("Executing up comand on compose file")

	cmd := exec.Command("docker-compose", "-f", service.composeFilePath, "up", "-d") // #nosec G204
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (service ComposeService) GetContainerLogs(id, tail string) (stdout, stderr chan []byte, e chan error, done chan bool) {
	slog.Info("Start listening container logs",
		"Id", id)

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
			slog.Error("Error while requesting logs",
				"Id", id,
				"Error", err)

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
					slog.Error("Error while reading log header",
						"Id", id,
						"Error", err)

					e <- err
					return
				}
				count := binary.BigEndian.Uint32(hdr[4:])
				dat := make([]byte, count)
				_, err = reader.Read(dat)
				if err != nil {
					slog.Error("Error while reading log message",
						"Id", id,
						"Error", err)
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

func (service ComposeService) getContainerInfo(id string) (types.ContainerJSON, []Process, error) {
	slog.Debug("Requesting container Debug",
		"Id", id)
	inspectResponse, err := service.cli.ContainerInspect(service.ctx, id)
	if err != nil {
		slog.Error("Container inspect error",
			"Id", id,
			"Error", err)
		return inspectResponse, nil, err
	}
	processes := make([]Process, 0)
	if inspectResponse.State.Status != "exited" {
		top, err := service.cli.ContainerTop(service.ctx, id, []string{"-eo", "pid,ppid,thcount,rss,%cpu,cmd"})
		if err != nil {
			slog.Error("Error requesting container top processes",
				"Id", id,
				"Error", err)
			return inspectResponse, nil, err
		}

		processes = parseProcesses(top)
	}

	return inspectResponse, processes, nil
}

func (service *ComposeService) GetContainerUpdates() (chan ContainerMsg, error) {
	if service.containerUpdates != nil {
		slog.Debug("Getting container updates channel")
		return service.containerUpdates, nil
	}
	slog.Info("Subscribing on containers updates")

	service.containerUpdates = make(chan ContainerMsg)

	for id := range service.containers {
		err := service.startLiseningForUpdates(id)
		if err != nil {
			slog.Error("Error subscribing on  updates",
				"Id", id,
				"Error", err)
			return nil, err
		}
	}

	slog.Info("Start synchronization process of containers")
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		for range ticker.C {
			err := service.syncContainers()
			if err != nil {
				slog.Error("Error in process of containers synchronization",
					"Error", err)
				continue
			}
		}
	}()

	slog.Debug("Getting container updates channel")
	return service.containerUpdates, nil
}

func (service *ComposeService) startLiseningForUpdates(id string) error {
	slog.Info("Subscribing on container updates",
		"Id", id)

	done := make(chan struct{})

	service.unsubscribeChannels[id] = done
	service.containers[id] = struct{}{}

	slog.Debug("Start listenning container statsisticts",
		"Id", id)

	statisticsResponse, err := service.cli.ContainerStats(service.ctx, id, true)
	if err != nil {
		slog.Error("Error requesting container statistics",
			"Id", id,
			"Error", err)

		return err
	}
	var newStats ContainerStats
	decoder := json.NewDecoder(statisticsResponse.Body)

	go func() {
		service.containerUpdates <- ContainerCreateMsg{ID: id}
		for {
			select {
			case <-done:
				slog.Info("Stop listeting container statistics",
					"Id", id)

				statisticsResponse.Body.Close()
				return
			case <-service.ctx.Done():
				slog.Info("Stop listeting container statistics due to end of provided stream",
					"Id", id)

				statisticsResponse.Body.Close()
				return
			default:
				if err := decoder.Decode(&newStats); errors.Is(err, io.EOF) {
					return
				} else if err != nil {
					slog.Error("Error decoding container statistic",
						"Id", id,
						"Error", err)

					panic(err)
				}

				inspectResponse, processes, err := service.getContainerInfo(id)
				if err != nil {
					slog.Error("Error inspecting container",
						"Id", id,
						"Error", err)

					service.removeContainer(id)
					continue
				}

				service.containerUpdates <- ContainerUpdateMsg{
					ID:        id,
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
	slog.Info("Start reading compose file")
	stack = filepath.Base(filepath.Dir(composeFilePath))

	bytes, err := os.ReadFile(composeFilePath)
	if err != nil {
		slog.Error("Error reading compose file",
			"Error", err)
		return "", compose, nil, err
	}

	err = yaml.Unmarshal(bytes, &compose)
	if err != nil {
		slog.Error("Error unmarchaling compose file data",
			"Error", err)
		return "", compose, nil, err
	}

	containers, err = cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		slog.Error("Error requesting containers list",
			"Error", err)
		return "", compose, nil, err
	}

	filteredByStack := slices.DeleteFunc(containers, func(container types.Container) bool { return container.Labels[stackLabel] != stack })

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
			if slices.ContainsFunc(maps.Keys(compose.Services), func(key string) bool { return key != name }) {
				return true
			}
		}

		return slices.ContainsFunc(maps.Values(compose.Services), func(service Service) bool { return service.Name != container.Names[0] })
	}

	filteredByStack = slices.DeleteFunc(containers, filter)

	return stack, compose, filteredByStack, nil
}
