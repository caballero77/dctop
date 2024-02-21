package stats

import (
	"fmt"
	"math"

	"github.com/caballero77/dctop/internal/configuration"
	"github.com/caballero77/dctop/internal/docker"
	"github.com/caballero77/dctop/internal/ui/helpers"
	"github.com/caballero77/dctop/internal/ui/messages"
	"github.com/caballero77/dctop/internal/ui/stats/drawing/plotting"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type cpu struct {
	cpuPlots map[string]plotting.Plot[float64]

	cpuUsages    map[string]float64
	maxCPUUsages map[string]float64

	plotColor   plotting.ColorGradient
	labelStyle  lipgloss.Style
	legendStyle lipgloss.Style
	scaling     []int

	containerID        string
	prevContainerStats map[string]docker.CPUStats

	width  int
	height int
}

func newCPU(theme configuration.Theme) tea.Model {
	model := cpu{
		cpuPlots:     make(map[string]plotting.Plot[float64]),
		cpuUsages:    make(map[string]float64),
		maxCPUUsages: make(map[string]float64),

		plotColor:   plotting.ColorGradient{From: theme.GetColor("plot.from"), To: theme.GetColor("plot.to")},
		labelStyle:  lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.plain")),
		legendStyle: lipgloss.NewStyle().Foreground(theme.GetColor("legend.plain")),

		prevContainerStats: make(map[string]docker.CPUStats),
		scaling:            []int{15, 25, 35, 45, 55, 65, 75, 100},
	}

	return helpers.NewBox(model, theme.Sub("border"))
}

func (cpu) Focus() bool { return false }

func (model cpu) Labels() []string {
	cpuUsage, ok := model.cpuUsages[model.containerID]
	if !ok {
		return []string{model.labelStyle.Render("cpu")}
	}

	return []string{model.labelStyle.Render(fmt.Sprintf("cpu: %.2f", cpuUsage) + "%")}
}

func (model cpu) Legends() []string {
	maxCPUUsage, ok := model.maxCPUUsages[model.containerID]
	if !ok {
		return []string{}
	}

	scale := model.calculateScalingCoefficient(maxCPUUsage)

	return []string{model.legendStyle.Render(fmt.Sprintf("scale: %d", int(math.Round(scale*100))) + "%")}
}

func (model cpu) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return model.UpdateAsBoxed(msg) }

func (cpu) Init() tea.Cmd {
	return nil
}

func (model cpu) UpdateAsBoxed(msg tea.Msg) (helpers.BoxedModel, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.ContainerSelectedMsg:
		model.containerID = msg.Container.InspectData.ID
	case docker.ContainerMsg:
		model.handleContainersUpdates(msg)
	case messages.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height

		for id, cpuPlot := range model.cpuPlots {
			cpuPlot.SetSize(msg.Width-2, msg.Height-2)
			model.cpuPlots[id] = cpuPlot
		}

	}
	return model, nil
}

func (model *cpu) handleContainersUpdates(msg docker.ContainerMsg) {
	switch msg := msg.(type) {
	case docker.ContainerUpdateMsg:
		switch msg.Inspect.State.Status {
		case "removing", "exited", "dead", "":
			delete(model.cpuPlots, msg.ID)
			delete(model.cpuUsages, msg.Inspect.ID)
		case "restarting", "paused", "running", "created":
			cpuPlot, ok := model.cpuPlots[msg.Inspect.ID]
			if !ok {
				cpuPlot = model.createNewPlot()
			}

			prevStats, ok := model.prevContainerStats[msg.Inspect.ID]
			if ok {
				usage := model.calculateCPUUsage(msg.Stats.CPUStats, prevStats)
				model.cpuUsages[msg.Inspect.ID] = usage
				if model.maxCPUUsages[msg.Inspect.ID] < usage {
					model.maxCPUUsages[msg.Inspect.ID] = usage
				}
				cpuPlot.Push(usage)
			}

			model.prevContainerStats[msg.Inspect.ID] = msg.Stats.CPUStats
			model.cpuPlots[msg.Inspect.ID] = cpuPlot
		}
	case docker.ContainerRemoveMsg:
		delete(model.cpuPlots, msg.ID)
		delete(model.prevContainerStats, msg.ID)
	}
}

func (model cpu) View() string {
	cpuPlot, ok := model.cpuPlots[model.containerID]
	if !ok {
		cpuPlot = model.createNewPlot()
		model.cpuPlots[model.containerID] = cpuPlot
	}

	return cpuPlot.View()
}

func (model cpu) calculateScalingCoefficient(maxValue float64) float64 {
	for i := 0; i < len(model.scaling); i++ {
		if maxValue < float64(model.scaling[i]) {
			return float64(model.scaling[i]) / 100
		}
	}
	return 1
}

func (cpu) calculateCPUUsage(currentStats, prevStats docker.CPUStats) float64 {
	var (
		cpuPercent  = 0.0
		cpuDelta    = float64(currentStats.CPUUsage.TotalUsage) - float64(prevStats.CPUUsage.TotalUsage)
		systemDelta = float64(currentStats.SystemCPUUsage) - float64(prevStats.SystemCPUUsage)
	)

	if systemDelta != 0.0 && cpuDelta != 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(currentStats.OnlineCpus) * 100.0
	}

	return cpuPercent
}

func (model cpu) createNewPlot() plotting.Plot[float64] {
	plot := plotting.New[float64](model.calculateScalingCoefficient, model.plotColor)
	plot.SetSize(model.width-2, model.height-2)
	return plot
}
