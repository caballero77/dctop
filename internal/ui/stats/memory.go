package stats

import (
	"dctop/internal/configuration"
	"dctop/internal/docker"
	"dctop/internal/ui/common"
	memory_utils "dctop/internal/utils/memory"
	"dctop/internal/utils/queues"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type memory struct {
	box         *common.BoxWithBorders
	plotStyles  lipgloss.Style
	labelStyle  lipgloss.Style
	legendStyle lipgloss.Style

	containerName      string
	prevContainerStats map[string]docker.ContainerStats
	memoryUsages       map[string]*queues.Queue[float64]
	memoryLimit        int
	cache              int

	width  int
	height int
}

func newMemory(theme configuration.Theme) memory {
	return memory{
		box:                common.NewBoxWithLabel(theme.Sub("border")),
		plotStyles:         lipgloss.NewStyle().Foreground(theme.GetColor("plot")),
		labelStyle:         lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.plain")),
		legendStyle:        lipgloss.NewStyle().Foreground(theme.GetColor("legend.plain")),
		prevContainerStats: make(map[string]docker.ContainerStats),
		memoryUsages:       make(map[string]*queues.Queue[float64]),
	}
}

func (model memory) Init() tea.Cmd {
	return nil
}

func (model memory) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case common.ContainerSelectedMsg:
		model.containerName = msg.Container.InspectData.Name
		model.memoryLimit = msg.Container.StatsSnapshot.MemoryStats.Limit
		model.cache = msg.Container.StatsSnapshot.MemoryStats.Stats.Cache
	case docker.ContainerUpdateMsg:
		switch msg.Inspect.State.Status {
		case "removing", "exited", "dead", "":
			delete(model.memoryUsages, msg.Inspect.Name)
		case "restarting", "paused", "running", "created":
			usage, ok := model.memoryUsages[msg.Inspect.Name]
			if !ok {
				usage = queues.New[float64]()
				model.memoryUsages[msg.Inspect.Name] = usage
			}
			err := pushWithLimit(usage, float64(model.calculateMemoruUsage(msg.Stats)), model.width*2)
			if err != nil {
				panic(err)
			}
			model.prevContainerStats[msg.Inspect.Name] = msg.Stats
		}
	case common.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height
	}
	return model, nil
}

func (model memory) View() string {
	memoryUsage, ok := model.memoryUsages[model.containerName]
	width := model.width - 2
	height := model.height - 2
	if !ok || memoryUsage.Len() == 0 {
		return model.box.Render([]string{}, []string{}, lipgloss.PlaceVertical(height, lipgloss.Center, lipgloss.PlaceHorizontal(width, lipgloss.Center, "test")), false)
	}

	memoryData := memoryUsage.ToArray()
	max := 0.0
	for _, value := range memoryData {
		if max < value {
			max = value
		}
	}
	for i, value := range memoryData {
		memoryData[i] = value / max * 100
	}

	plot := model.plotStyles.Render(renderPlot(memoryData, 1.6, width, height))

	limit, err := memory_utils.BytesToReadable(float64(model.memoryLimit))
	if err != nil {
		panic(err)
	}
	legend := model.legendStyle.Render(fmt.Sprintf("limit %s", limit))

	currentUsageInBytes, err := memoryUsage.Head()
	if err != nil {
		panic(err)
	}
	currentUsage, err := memory_utils.BytesToReadable(currentUsageInBytes)
	if err != nil {
		panic(err)
	}

	label := model.labelStyle.Render(fmt.Sprintf("Memory: %s", currentUsage))

	return model.box.Render([]string{label}, []string{legend}, plot, false)
}

func (memory) calculateMemoruUsage(currStats docker.ContainerStats) int {
	return currStats.MemoryStats.Usage - currStats.MemoryStats.Stats.Cache
}
