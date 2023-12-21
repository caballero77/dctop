package stats

import (
	"dctop/internal/docker"
	"dctop/internal/ui/common"
	"dctop/internal/utils/queues"
	"fmt"
	"math"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type cpu struct {
	box         *common.BoxWithBorders
	plotStyles  lipgloss.Style
	labelStyle  lipgloss.Style
	legendStyle lipgloss.Style
	scaling     []int

	containerID        string
	cpuUsages          map[string]*queues.Queue[float64]
	prevContainerStats map[string]docker.ContainerStats

	width  int
	height int
}

func newCPU() cpu {
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
	focusBorderStyle := lipgloss.Color("#434C5E")

	return cpu{
		box: common.NewBoxWithLabel(border, borderStyle, focusBorderStyle),
		plotStyles: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#81A1C1")),
		legendStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#434C5E")),
		labelStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("#D8DEE9")),

		cpuUsages:          make(map[string]*queues.Queue[float64]),
		prevContainerStats: make(map[string]docker.ContainerStats),
		scaling:            []int{15, 25, 35, 45, 55, 65, 75, 100},
	}
}

func (cpu) Init() tea.Cmd {
	return nil
}

func (model cpu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case common.ContainerSelectedMsg:
		model.containerID = msg.Container.InspectData.ID
	case docker.ContainerUpdateMsg:
		switch msg.Inspect.State.Status {
		case "removing", "exited", "dead":
			delete(model.prevContainerStats, msg.Inspect.ID)
			delete(model.cpuUsages, msg.Inspect.ID)
		case "restarting", "paused", "running", "created":
			prevStats, ok := model.prevContainerStats[msg.Inspect.ID]
			if ok {
				usage, ok := model.cpuUsages[msg.Inspect.ID]
				if !ok {
					usage = queues.New[float64]()
					model.cpuUsages[msg.Inspect.ID] = usage
				}
				err := pushWithLimit(usage, model.calculateCPUUsage(msg.Stats, prevStats), model.width*2)
				if err != nil {
					panic(err)
				}
			}
			model.prevContainerStats[msg.Inspect.ID] = msg.Stats
		}
	case common.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height
	}
	return model, nil
}

func (model cpu) View() string {
	cpuUsage, ok := model.cpuUsages[model.containerID]
	width := model.width - 2
	height := model.height - 2
	if !ok || cpuUsage == nil || cpuUsage.Len() == 0 {
		return model.box.Render([]string{}, []string{}, lipgloss.PlaceVertical(height, lipgloss.Center, lipgloss.PlaceHorizontal(width, lipgloss.Center, "test")), false)
	}

	cpuData := cpuUsage.ToArray()
	max := 0.0
	for _, value := range cpuData {
		if max < value {
			max = value
		}
	}
	scale := model.calculateScalingKoeficient(max)

	plot := model.plotStyles.Render(renderPlot(cpuData, scale, width, height))

	legend := model.legendStyle.Render(fmt.Sprintf("scale: %d", int(math.Round(scale*100))) + "%")

	currentUsage, err := cpuUsage.Head()
	if err != nil {
		panic(err)
	}
	label := model.labelStyle.Render(fmt.Sprintf("Cpu: %.2f", currentUsage) + "%")

	return model.box.Render([]string{label}, []string{legend}, plot, false)
}

func (model cpu) calculateScalingKoeficient(maxValue float64) float64 {
	for i := 0; i < len(model.scaling); i++ {
		if maxValue < float64(model.scaling[i]) {
			return float64(model.scaling[i]) / 100
		}
	}
	return 1
}

func (cpu) calculateCPUUsage(currStats, prevStats docker.ContainerStats) float64 {
	var (
		cpuPercent  = 0.0
		cpuDelta    = float64(currStats.CPUStats.CPUUsage.TotalUsage) - float64(prevStats.CPUStats.CPUUsage.TotalUsage)
		systemDelta = float64(currStats.CPUStats.SystemCPUUsage) - float64(prevStats.CPUStats.SystemCPUUsage)
	)

	if systemDelta != 0.0 && cpuDelta != 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(currStats.CPUStats.OnlineCpus) * 100.0
	}

	return cpuPercent
}
