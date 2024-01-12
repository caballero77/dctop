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

type memory struct {
	box         helpers.BoxWithBorders
	plotStyles  lipgloss.Style
	labelStyle  lipgloss.Style
	legendStyle lipgloss.Style

	containerID  string
	memoryUsages map[string]*queues.Queue[uint]
	memoryLimit  uint64
	cache        uint64

	width  int
	height int
}

func newMemory(theme configuration.Theme) memory {
	return memory{
		box:          helpers.NewBox(theme.Sub("border")),
		plotStyles:   lipgloss.NewStyle().Foreground(theme.GetColor("plot")),
		labelStyle:   lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.plain")),
		legendStyle:  lipgloss.NewStyle().Foreground(theme.GetColor("legend.plain")),
		memoryUsages: make(map[string]*queues.Queue[uint]),
	}
}

func (model memory) Init() tea.Cmd {
	return nil
}

func (model memory) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	}
	return model, nil
}

func (model *memory) handleContainersUpdates(msg docker.ContainerMsg) {
	switch msg := msg.(type) {
	case docker.ContainerUpdateMsg:
		switch msg.Inspect.State.Status {
		case "removing", "exited", "dead", "":
			delete(model.memoryUsages, msg.Inspect.ID)
		case "restarting", "paused", "running", "created":
			usage, ok := model.memoryUsages[msg.Inspect.ID]
			if !ok {
				usage = queues.New[uint]()
				model.memoryUsages[msg.Inspect.ID] = usage
			}
			err := usage.PushWithLimit(model.calculateMemoruUsage(msg.Stats), model.width*2)
			if err != nil {
				if err != nil {
					slog.Error("error pushing element into queue with limit")
				}
			}
		}
	case docker.ContainerRemoveMsg:
		delete(model.memoryUsages, msg.ID)
	}
}

func (model memory) View() string {
	memoryUsage, ok := model.memoryUsages[model.containerID]
	width := model.width - 2
	height := model.height - 2
	if !ok || memoryUsage.Len() == 0 {
		return model.box.Render([]string{model.labelStyle.Render("memory")}, []string{}, lipgloss.PlaceVertical(height, lipgloss.Center, lipgloss.PlaceHorizontal(width, lipgloss.Center, "no data")), false)
	}

	memoryData := memoryUsage.ToArray()
	plottingData := make([]float64, len(memoryData))
	var max uint
	for _, value := range memoryData {
		if max < value {
			max = value
		}
	}
	for i, value := range memoryData {
		plottingData[i] = float64(value) / float64(max) * 100
	}

	plot := model.plotStyles.Render(renderPlot(plottingData, 1.6, width, height))

	legend := model.legendStyle.Render(fmt.Sprintf("limit %s", humanize.IBytes(model.memoryLimit)))

	currentUsageInBytes, err := memoryUsage.Head()
	if err != nil {
		slog.Error("error getting head from memory usage queue", err)
		return model.box.Render(
			[]string{model.labelStyle.Render("cpu")},
			[]string{},
			lipgloss.PlaceVertical(height, lipgloss.Center, lipgloss.PlaceHorizontal(width, lipgloss.Center, "error rendering memory usage plot")),
			false,
		)
	}

	label := model.labelStyle.Render(fmt.Sprintf("memory: %s", humanize.IBytes(uint64(currentUsageInBytes))))

	return model.box.Render([]string{label}, []string{legend}, plot, false)
}

func (memory) calculateMemoruUsage(currStats docker.ContainerStats) uint {
	return uint(currStats.MemoryStats.Usage - currStats.MemoryStats.Stats.Cache)
}
