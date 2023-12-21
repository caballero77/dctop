package stats

import (
	"dctop/internal/docker"
	"dctop/internal/ui/common"
	memory_utils "dctop/internal/utils/memory"
	"dctop/internal/utils/queues"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type containerIo struct {
	IoRead  *queues.Queue[int]
	IoWrite *queues.Queue[int]
}

type io struct {
	box         *common.BoxWithBorders
	plotStyles  lipgloss.Style
	labelStyle  lipgloss.Style
	legendStyle lipgloss.Style
	scaling     []int

	containerID string
	ioUsages    map[string]containerIo

	width  int
	height int
}

func newIO() io {
	border := lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
	}
	borderStyle := lipgloss.Color("#434C5E")
	focusBorderStyle := lipgloss.Color("#8FBCBB")

	return io{
		box: common.NewBoxWithLabel(border, borderStyle, focusBorderStyle),
		plotStyles: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#81A1C1")),
		labelStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("#D8DEE9")),
		legendStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#434C5E")),
		ioUsages:    make(map[string]containerIo),
		scaling:     []int{15, 25, 35, 45, 55, 65, 75, 100},
	}
}

func (model io) Init() tea.Cmd {
	return nil
}

func (model io) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case common.ContainerSelectedMsg:
		model.containerID = msg.Container.InspectData.ID
	case docker.ContainerUpdateMsg:
		switch msg.Inspect.State.Status {
		case "removing", "exited", "dead":
			delete(model.ioUsages, msg.Inspect.ID)
		case "restarting", "paused", "running", "created":
			network, ok := model.ioUsages[msg.Inspect.ID]
			if !ok {
				network = containerIo{
					IoRead:  queues.New[int](),
					IoWrite: queues.New[int](),
				}
				model.ioUsages[msg.Inspect.ID] = network
			}
			read, write := model.getIoUsage(&msg.Stats.BlkioStats)
			err := pushWithLimit(network.IoRead, read, model.width*2)
			if err != nil {
				panic(err)
			}
			err = pushWithLimit(network.IoWrite, write, model.width*2)
			if err != nil {
				panic(err)
			}
		}
	case common.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height
	}
	return model, nil
}

func (model io) View() string {
	width := model.width - 4
	height := model.height - 2
	ioUsage, ok := model.ioUsages[model.containerID]
	if !ok || (ioUsage.IoRead.Len() == 0 && ioUsage.IoWrite.Len() == 0) {
		return model.box.Render([]string{}, []string{}, lipgloss.PlaceVertical(height, lipgloss.Center, lipgloss.PlaceHorizontal(width, lipgloss.Center, "test")), false)
	}
	incoming := model.RenderNetwork(ioUsage.IoRead, func(current string) string { return model.labelStyle.Render(fmt.Sprintf("IO Read: %s/sec", current)) }, width/2, height)

	outcoming := model.RenderNetwork(ioUsage.IoWrite, func(current string) string {
		return model.labelStyle.Render(fmt.Sprintf("IO Write: %s/sec", current))
	}, width/2+width%2, height)

	return lipgloss.JoinHorizontal(lipgloss.Center, incoming, outcoming)
}

func (model io) RenderNetwork(queue *queues.Queue[int], label func(string) string, width, height int) string {
	if queue.Len() == 0 {
		return model.box.Render([]string{}, []string{}, lipgloss.PlaceVertical(height, lipgloss.Center, lipgloss.PlaceHorizontal(width, lipgloss.Center, "test")), false)
	}

	total, err := queue.Head()
	if err != nil {
		panic(err)
	}

	data, max, maxChange, current := getDataChangeFromQueue(queue.ToArray(), width)
	scale := model.calculateScalingKoeficient(max)
	plot := model.plotStyles.Render(renderPlot(data, scale, width, height))

	currentRx, err := memory_utils.BytesToReadable(float64(total))
	if err != nil {
		panic(err)
	}

	maxRxChange, err := memory_utils.BytesToReadable(maxChange)
	if err != nil {
		panic(err)
	}

	currentChange, err := memory_utils.BytesToReadable(current)
	if err != nil {
		panic(err)
	}

	legend := []string{
		model.legendStyle.Render(fmt.Sprintf("Total: %s", currentRx)),
		model.legendStyle.Render(fmt.Sprintf("Max: %s/sec", maxRxChange)),
	}

	return model.box.Render([]string{label(currentChange)}, legend, plot, false)
}

func (model io) calculateScalingKoeficient(maxValue float64) float64 {
	for i := 0; i < len(model.scaling); i++ {
		if maxValue < float64(model.scaling[i]) {
			return float64(model.scaling[i]) / 100
		}
	}
	return 1
}

func (io) getIoUsage(stats *docker.BlkioStats) (read, write int) {
	for i := 0; i < len(stats.IoServiceBytesRecursive); i++ {
		curr := stats.IoServiceBytesRecursive[i]
		switch curr.Operation {
		case "read":
			read += curr.Value
		case "write":
			write += curr.Value
		}
	}
	return read, write
}
