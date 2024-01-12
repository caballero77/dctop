package stats

import (
	"dctop/internal/configuration"
	"dctop/internal/docker"
	"dctop/internal/ui/helpers"
	"dctop/internal/ui/messages"
	"dctop/internal/utils/queues"
	"fmt"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

type containerIo struct {
	IoRead  *queues.Queue[uint64]
	IoWrite *queues.Queue[uint64]
}

type io struct {
	box         helpers.BoxWithBorders
	plotStyles  lipgloss.Style
	labelStyle  lipgloss.Style
	legendStyle lipgloss.Style
	scaling     []int

	containerID string
	ioUsages    map[string]containerIo

	width  int
	height int
}

func newIO(theme configuration.Theme) io {
	return io{
		box:         helpers.NewBox(theme.Sub("border")),
		plotStyles:  lipgloss.NewStyle().Foreground(theme.GetColor("plot")),
		labelStyle:  lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.plain")),
		legendStyle: lipgloss.NewStyle().Foreground(theme.GetColor("legend.plain")),
		ioUsages:    make(map[string]containerIo),
		scaling:     []int{15, 25, 35, 45, 55, 65, 75, 100},
	}
}

func (model io) Init() tea.Cmd {
	return nil
}

func (model io) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.ContainerSelectedMsg:
		model.containerID = msg.Container.InspectData.ID
	case docker.ContainerMsg:
		model.handleContainersUpdates(msg)
	case messages.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height
	}
	return model, nil
}

func (model *io) handleContainersUpdates(msg docker.ContainerMsg) {
	switch msg := msg.(type) {
	case docker.ContainerUpdateMsg:
		switch msg.Inspect.State.Status {
		case "removing", "exited", "dead", "":
			delete(model.ioUsages, msg.Inspect.ID)
		case "restarting", "paused", "running", "created":
			network, ok := model.ioUsages[msg.Inspect.ID]
			if !ok {
				network = containerIo{
					IoRead:  queues.New[uint64](),
					IoWrite: queues.New[uint64](),
				}
				model.ioUsages[msg.Inspect.ID] = network
			}
			read, write := model.getIoUsage(&msg.Stats.BlkioStats)
			err := network.IoRead.PushWithLimit(read, model.width*2)
			if err != nil {
				slog.Error("error pushing element into queue with limit")
			}
			err = network.IoWrite.PushWithLimit(write, model.width*2)
			if err != nil {
				slog.Error("error pushing element into queue with limit")
			}
		}
	case docker.ContainerRemoveMsg:
		delete(model.ioUsages, msg.ID)
	}
}

func (model io) View() string {
	width := model.width - 4
	height := model.height - 2
	ioUsage, ok := model.ioUsages[model.containerID]
	if !ok {
		ioUsage = containerIo{
			IoRead:  queues.New[uint64](),
			IoWrite: queues.New[uint64](),
		}
	}

	getLabelRenderer := func(networkType string) func(string) string {
		return func(current string) string {
			if current == "" {
				return model.labelStyle.Render(networkType)
			}
			return model.labelStyle.Render(fmt.Sprintf("%s: %s/sec", networkType, current))
		}
	}

	read, err := model.RenderIo(ioUsage.IoRead, getLabelRenderer("io read"), width/2, height)
	if err != nil {
		slog.Error("error rendering io read plot",
			"error", err,
			"id", model.containerID)
		return model.renderErrorMessage(width, height)
	}

	write, err := model.RenderIo(ioUsage.IoWrite, getLabelRenderer("io write"), width/2+width%2, height)
	if err != nil {
		slog.Error("error rendering io write plot",
			"error", err,
			"id", model.containerID)
		return model.renderErrorMessage(width, height)
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, read, write)
}

func (model io) RenderIo(queue *queues.Queue[uint64], label func(string) string, width, height int) (string, error) {
	if queue.Len() <= 1 {
		return model.box.Render(
			[]string{label("")},
			[]string{},
			lipgloss.PlaceVertical(height, lipgloss.Center, lipgloss.PlaceHorizontal(width, lipgloss.Center, "no data")),
			false,
		), nil
	}

	total, err := queue.Head()
	if err != nil {
		return "", fmt.Errorf("error getting head from io usage queue: %w", err)
	}

	data, maxRate, currentRate := getRate(queue.ToArray())
	plot := model.plotStyles.Render(renderPlot(data, 1, width, height))

	legends := []string{
		model.legendStyle.Render(fmt.Sprintf("total: %s", humanize.IBytes(total))),
		model.legendStyle.Render(fmt.Sprintf("max: %s/sec", humanize.IBytes(maxRate))),
	}

	return model.box.Render([]string{label(humanize.IBytes(currentRate))}, legends, plot, false), nil
}

func (model io) renderErrorMessage(height, width int) string {
	return model.box.Render(
		[]string{model.labelStyle.Render("cpu")},
		[]string{},
		lipgloss.PlaceVertical(height, lipgloss.Center, lipgloss.PlaceHorizontal(width, lipgloss.Center, "error rendering io usage plot")),
		false,
	)
}

func (io) getIoUsage(stats *docker.BlkioStats) (read, write uint64) {
	for i := 0; i < len(stats.IoServiceBytesRecursive); i++ {
		curr := stats.IoServiceBytesRecursive[i]
		switch curr.Operation {
		case "read":
			read += uint64(curr.Value)
		case "write":
			write += uint64(curr.Value)
		}
	}
	return read, write
}
