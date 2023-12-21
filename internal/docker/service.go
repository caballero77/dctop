package docker

import (
	"context"
	"dctop/internal/utils/maps"
	"dctop/internal/utils/slices"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"gopkg.in/yaml.v3"
)

var stackLabel = "com.docker.compose.project"

type ComposeService struct {
	cli             *client.Client
	ctx             context.Context
	containers      []string
	prevStats       map[string]ContainerStats
	stack           string
	composeFilePath string

	containerUpdates chan ContainerUpdateMsg
}

func ProvideComposeService(composeFilePath string) (*ComposeService, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	stack, containers, err := getStack(ctx, cli, composeFilePath)
	if err != nil {
		return nil, err
	}

	containersIds, err := slices.Map(containers, func(c types.Container) (string, error) { return c.ID, nil })
	if err != nil {
		return nil, err
	}

	service := &ComposeService{
		cli:              cli,
		ctx:              ctx,
		stack:            stack,
		containers:       containersIds,
		prevStats:        make(map[string]ContainerStats, len(containersIds)),
		containerUpdates: nil,
		composeFilePath:  composeFilePath,
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

func (service *ComposeService) GetContainerUpdates() (chan ContainerUpdateMsg, error) {
	if service.containerUpdates != nil {
		return service.containerUpdates, nil
	}
	var err error
	service.containerUpdates, err = service.startListenningForStatistic()
	if err != nil {
		return nil, err
	}
	return service.containerUpdates, nil
}

func (service ComposeService) startListenningForStatistic() (chan ContainerUpdateMsg, error) {
	updates := make(chan ContainerUpdateMsg, 100)
	for i := 0; i < len(service.containers); i++ {
		containerID := service.containers[i]

		statisticsResponse, err := service.cli.ContainerStats(service.ctx, containerID, true)
		if err != nil {
			return nil, err
		}
		var newStats ContainerStats
		decoder := json.NewDecoder(statisticsResponse.Body)

		go func() {
			for {
				select {
				case <-service.ctx.Done():
					statisticsResponse.Body.Close()
					return
				default:
					if err := decoder.Decode(&newStats); errors.Is(err, io.EOF) {
						return
					} else if err != nil {
						panic(err)
					}

					inspectResponse, processes, err := service.getContainerInfo(containerID)
					if err != nil {
						panic(err)
					}

					update := ContainerUpdateMsg{
						ID:        containerID,
						Inspect:   inspectResponse,
						Stats:     newStats,
						Processes: processes,
					}

					updates <- update
				}
			}
		}()
	}
	return updates, nil
}

func getStack(ctx context.Context, cli *client.Client, composeFilePath string) (stack string, containers []types.Container, err error) {
	stack = filepath.Base(filepath.Dir(composeFilePath))

	containers, err = cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return "", nil, err
	}

	filteredByStack := slices.Filter(containers, func(container types.Container) bool { return container.Labels[stackLabel] == stack })

	if len(filteredByStack) != 0 {
		return stack, filteredByStack, nil
	}

	bytes, err := os.ReadFile(composeFilePath)
	if err != nil {
		return "", nil, err
	}

	var compose Compose

	err = yaml.Unmarshal(bytes, &compose)
	if err != nil {
		return "", nil, err
	}

	stack = ""

	filter := func(container types.Container) bool {
		_, name := func() (string, string) {
			parts := strings.Split(container.Names[0], "-")
			return parts[0], parts[1]
		}()

		if _, err := slices.Find(maps.Keys(compose.Services), func(key string) bool { return key == name }); err == nil {
			return true
		}

		return false
	}

	filteredByStack = slices.Filter(containers, filter)

	if len(filteredByStack) == 0 {
		return "", nil, errors.New("can't find containers associated with provided compose file")
	}

	return stack, filteredByStack, nil
}
