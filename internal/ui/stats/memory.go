package stats

import (
	"dctop/internal/configuration"
	"dctop/internal/docker"
	"dctop/internal/ui/helpers"
	"dctop/internal/ui/messages"
	"dctop/internal/ui/stats/drawing/plotting"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

type memory struct {
	plotStyles  lipgloss.Style
	labelStyle  lipgloss.Style
	legendStyle lipgloss.Style

	memoryPlots    map[string]plotting.Plot[float64]
	memoryUsages   map[string]uint
	maxMemoryUsage map[string]uint

	containerID string
	memoryLimit uint64
	cache       uint64

	width  int
	height int
}

func newMemory(theme configuration.Theme) tea.Model {
	model := memory{
		plotStyles:     lipgloss.NewStyle().Foreground(theme.GetColor("plot")),
		labelStyle:     lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.plain")),
		legendStyle:    lipgloss.NewStyle().Foreground(theme.GetColor("legend.plain")),
		memoryPlots:    make(map[string]plotting.Plot[float64]),
		maxMemoryUsage: make(map[string]uint),
		memoryUsages:   make(map[string]uint),
	}

	return helpers.NewBox(model, theme.Sub("border"))
}

func (memory) Focus() bool { return false }

func (model memory) Labels() []string {
	memoryUsage, ok := model.memoryUsages[model.containerID]
	if !ok {
		return []string{model.labelStyle.Render("memory")}
	}

	return []string{model.labelStyle.Render(fmt.Sprintf("memory: %s", humanize.IBytes(uint64(memoryUsage))))}
}

func (model memory) Legends() []string {
	return []string{model.legendStyle.Render(fmt.Sprintf("limit %s", humanize.IBytes(model.memoryLimit)))}
}

func (model memory) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return model.UpdateAsBoxed(msg) }

func (model memory) Init() tea.Cmd {
	return nil
}

func (model memory) UpdateAsBoxed(msg tea.Msg) (helpers.BoxedModel, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.ContainerSelectedMsg:
		model.containerID = msg.Container.InspectData.ID
		model.memoryLimit = uint64(msg.Container.StatsSnapshot.MemoryStats.Limit)
		model.cache = uint64(msg.Container.StatsSnapshot.MemoryStats.Stats.Cache)
	case docker.ContainerMsg:
		model.handleContainersUpdates(msg)
	case messages.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height

		for id, memoryPlot := range model.memoryPlots {
			memoryPlot.SetSize(msg.Width-2, msg.Height-2)
			model.memoryPlots[id] = memoryPlot
		}
	}
	return model, nil
}

func (model *memory) handleContainersUpdates(msg docker.ContainerMsg) {
	switch msg := msg.(type) {
	case docker.ContainerUpdateMsg:
		switch msg.Inspect.State.Status {
		case "removing", "exited", "dead", "":
			delete(model.memoryPlots, msg.ID)
			delete(model.memoryUsages, msg.Inspect.ID)
		case "restarting", "paused", "running", "created":
			memoryPlot, ok := model.memoryPlots[msg.Inspect.ID]
			if !ok {
				memoryPlot = model.createNewPlot()
				model.memoryPlots[msg.Inspect.ID] = memoryPlot
			}
			usage := model.calculateMemoryUsage(msg.Stats)
			model.memoryUsages[msg.Inspect.ID] = usage
			if model.maxMemoryUsage[msg.Inspect.ID] < usage {
				model.maxMemoryUsage[msg.Inspect.ID] = usage
			}
			memoryPlot.Push(100 * float64(usage) / float64(model.maxMemoryUsage[msg.Inspect.ID]))
		}
	case docker.ContainerRemoveMsg:
		delete(model.memoryPlots, msg.ID)
		delete(model.memoryUsages, msg.ID)
	}
}

func (model memory) View() string {
	memoryPlot, ok := model.memoryPlots[model.containerID]
	if !ok {
		memoryPlot = model.createNewPlot()
		model.memoryPlots[model.containerID] = memoryPlot
	}

	return model.plotStyles.Render(memoryPlot.View())
}

func (memory) calculateMemoryUsage(currentStats docker.ContainerStats) uint {
	return uint(currentStats.MemoryStats.Usage - currentStats.MemoryStats.Stats.Cache)
}

func (model memory) createNewPlot() plotting.Plot[float64] {
	memoryPlot := plotting.New[float64]()
	memoryPlot.SetScale(1.6)
	memoryPlot.SetSize(model.width-2, model.height-2)
	return memoryPlot
}
