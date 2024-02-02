package docker

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

const stackLabel = "com.docker.compose.project"

type ContainersService struct {
	cli        *client.Client
	ctx        context.Context
	containers map[string]struct{}
	stack      string

	containerUpdates          chan ContainerMsg
	unsubscribeChannels       map[string]func()
	stopSynchronizationTicker func()
}

func NewContainersService(ctx context.Context, stack string) (ContainersService, error) {
	slog.Info("Creating docker client")
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		slog.Error("error creating docker client",
			"error", err)
		var containersService ContainersService
		return containersService, err
	}

	service := ContainersService{
		cli:                 cli,
		ctx:                 ctx,
		stack:               stack,
		containers:          make(map[string]struct{}),
		containerUpdates:    nil,
		unsubscribeChannels: make(map[string]func()),
	}

	return service, nil
}

func (service ContainersService) Stack() string {
	return service.stack
}

func (service ContainersService) Close() error {
	slog.Debug("Containers service has stopped")

	if service.stopSynchronizationTicker != nil {
		service.stopSynchronizationTicker()
	}

	for _, unsubscribe := range service.unsubscribeChannels {
		unsubscribe()
	}
	err := service.cli.Close()
	if err != nil {
		return fmt.Errorf("error while closing containers service: %w", err)
	}
	return nil
}

func (service ContainersService) ContainerPause(id string) error {
	slog.Debug("Pausing container",
		"Id", id)

	err := service.cli.ContainerPause(service.ctx, id)
	if err != nil {
		return fmt.Errorf("error while pausing container: %w", err)
	}
	return nil
}

func (service ContainersService) ContainerUnpause(id string) error {
	slog.Debug("Unpausing container",
		"Id", id)

	err := service.cli.ContainerUnpause(service.ctx, id)
	if err != nil {
		return fmt.Errorf("error while unpausing container: %w", err)
	}
	return nil
}

func (service ContainersService) ContainerStop(id string) error {
	slog.Debug("Stopping container",
		"Id", id)

	err := service.cli.ContainerStop(service.ctx, id, container.StopOptions{})
	if err != nil {
		return fmt.Errorf("error while stopping container: %w", err)
	}
	return nil
}

func (service ContainersService) ContainerStart(id string) error {
	slog.Debug("Starting container",
		"Id", id)

	err := service.cli.ContainerStart(service.ctx, id, types.ContainerStartOptions{})
	if err != nil {
		return fmt.Errorf("error while starting container: %w", err)
	}
	return nil
}

func (service ContainersService) GetContainerLogs(ctx context.Context, id, tail string) (stdout, stderr chan []byte, e chan error) {
	slog.Info("Start listening container logs",
		"Id", id)

	e = make(chan error)
	stdout, stderr = make(chan []byte), make(chan []byte)

	go func() {
		reader, err := service.cli.ContainerLogs(ctx, id, types.ContainerLogsOptions{
			ShowStderr: true,
			ShowStdout: true,
			Timestamps: false,
			Tail:       tail,
			Follow:     true,
		})
		if err != nil {
			e <- fmt.Errorf("error while requesting container logs: %w", err)
			close(e)
			return
		}
		hdr := make([]byte, 8)
		for {
			select {
			case <-ctx.Done():
				close(e)
				close(stderr)
				close(stdout)
				return
			default:
				_, err := reader.Read(hdr)
				if err != nil {
					e <- fmt.Errorf("error while reading log header: %w", err)
					close(e)
					close(stderr)
					close(stdout)
					return
				}
				count := binary.BigEndian.Uint32(hdr[4:])
				dat := make([]byte, count)
				_, err = reader.Read(dat)
				if err != nil {
					e <- fmt.Errorf("error while reading log message: %w", err)
					close(e)
					close(stderr)
					close(stdout)
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

	return stdout, stderr, e
}

func (service *ContainersService) GetContainerUpdates() (chan ContainerMsg, error) {
	if service.containerUpdates != nil {
		slog.Debug("Getting container updates channel")
		return service.containerUpdates, nil
	}
	slog.Info("Subscribing on containers updates")

	service.containerUpdates = make(chan ContainerMsg)

	slog.Info("Start synchronization process of containers")

	sync := func() {
		err := service.syncContainers()
		if err != nil {
			slog.Error("error in process of containers synchronization",
				"error", err)
		}
	}

	ticker := time.NewTicker(5 * time.Second)
	service.stopSynchronizationTicker = ticker.Stop

	go func() {
		sync()
		for range ticker.C {
			sync()
		}
	}()

	slog.Debug("Getting container updates channel")
	return service.containerUpdates, nil
}

func (service ContainersService) mapTopProcess(top container.ContainerTopOKBody) []Process {
	titles := make(map[string]int, len(top.Titles))
	for i := 0; i < len(top.Titles); i++ {
		titles[top.Titles[i]] = i
	}

	processes := make([]Process, len(top.Processes))
	for i, process := range top.Processes {
		processes[i] = Process{
			PID:     process[titles["PID"]],
			PPID:    process[titles["PPID"]],
			Threads: process[titles["THCNT"]],
			RSS:     process[titles["RSS"]],
			CPU:     process[titles["%CPU"]],
			CMD:     process[titles["CMD"]],
		}
	}

	return processes
}

func (service *ContainersService) startListeningForUpdates(id string) error {
	slog.Info("Subscribing on container updates",
		"Id", id)

	ctx, cancel := context.WithCancel(service.ctx)

	service.unsubscribeChannels[id] = cancel
	service.containers[id] = struct{}{}

	slog.Debug("Start listening container statistics",
		"Id", id)

	statisticsResponse, err := service.cli.ContainerStats(ctx, id, true)
	if err != nil {
		return fmt.Errorf("error requesting container statistics: %w", err)
	}
	var newStats ContainerStats
	decoder := json.NewDecoder(statisticsResponse.Body)

	go func() {
		service.containerUpdates <- ContainerCreateMsg{ID: id}
		for {
			select {
			case <-ctx.Done():
				slog.Info("Stop listening container statistics due to end of provided stream",
					"Id", id)

				statisticsResponse.Body.Close()
				return
			default:
				if err := decoder.Decode(&newStats); errors.Is(err, io.EOF) {
					return
				} else if err != nil {
					slog.Error("error decoding container statistic",
						"id", id,
						"error", err)

					continue
				}

				inspectResponse, err := service.cli.ContainerInspect(ctx, id)
				if err != nil {
					slog.Error("error inspecting container",
						"id", id,
						"error", err)

					service.removeContainer(id)
					continue
				}

				var processes []Process

				if inspectResponse.State.Status != "exited" {
					top, err := service.cli.ContainerTop(ctx, id, []string{"-eo", "pid,ppid,thcount,rss,%cpu,cmd"})
					if err != nil {
						slog.Error("error while requesting container top processes",
							"id", id,
							"error", err)
					}

					processes = service.mapTopProcess(top)
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

func (service ContainersService) syncContainers() error {
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
			err := service.startListeningForUpdates(container.ID)
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

func (service *ContainersService) removeContainer(id string) {
	if unsubscribe, ok := service.unsubscribeChannels[id]; ok {
		unsubscribe()
	}
	service.containerUpdates <- ContainerRemoveMsg{ID: id}
	delete(service.containers, id)
}
