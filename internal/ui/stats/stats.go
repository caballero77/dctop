package stats

import (
	"dctop/internal/configuration"
	"dctop/internal/ui/helpers"
	"dctop/internal/ui/messages"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Stats struct {
	networkModel     tea.Model
	ioStatsModel     tea.Model
	cpuStatsModel    tea.Model
	memoryStatsModel tea.Model

	width  int
	height int
}

func NewStats(theme configuration.Theme) Stats {
	return Stats{
		networkModel:     newNetwork(theme.Sub("network")),
		ioStatsModel:     newIO(theme.Sub("io")),
		cpuStatsModel:    newCPU(theme.Sub("cpu")),
		memoryStatsModel: newMemory(theme.Sub("memory")),
	}
}

func (model Stats) Init() tea.Cmd {
	return helpers.Init(
		model.networkModel,
		model.ioStatsModel,
		model.memoryStatsModel,
		model.cpuStatsModel,
	)
}

func (model Stats) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(messages.SizeChangeMsq); ok {
		model.width = msg.Width
		model.height = msg.Height

		var (
			cpuSize     = messages.SizeChangeMsq{Width: msg.Width, Height: msg.Height - 3*(msg.Height/5)}
			memorySize  = messages.SizeChangeMsq{Width: msg.Width, Height: msg.Height / 5}
			networkSize = messages.SizeChangeMsq{Width: msg.Width, Height: msg.Height / 5}
			ioSize      = messages.SizeChangeMsq{Width: msg.Width, Height: msg.Height / 5}
		)

		cmd := tea.Batch(helpers.PassMsgs(
			helpers.NewModel(model.cpuStatsModel, func(m tea.Model) { model.cpuStatsModel = m }).WithMsg(cpuSize),
			helpers.NewModel(model.memoryStatsModel, func(m tea.Model) { model.memoryStatsModel = m }).WithMsg(memorySize),
			helpers.NewModel(model.networkModel, func(m tea.Model) { model.networkModel = m }).WithMsg(networkSize),
			helpers.NewModel(model.ioStatsModel, func(m tea.Model) { model.ioStatsModel = m }).WithMsg(ioSize),
		))
		return model, cmd
	}

	cmds := make([]tea.Cmd, 0)
	cmds = append(cmds, helpers.PassMsg(msg,
		helpers.NewModel(model.networkModel, func(m tea.Model) { model.networkModel = m }),
		helpers.NewModel(model.ioStatsModel, func(m tea.Model) { model.ioStatsModel = m }),
		helpers.NewModel(model.memoryStatsModel, func(m tea.Model) { model.memoryStatsModel = m }),
		helpers.NewModel(model.cpuStatsModel, func(m tea.Model) { model.cpuStatsModel = m }),
	))

	return model, tea.Batch(cmds...)
}

func (model Stats) View() string {
	networkTab := model.networkModel.View()

	ioTab := model.ioStatsModel.View()

	memoryTab := model.memoryStatsModel.View()

	cpuTab := model.cpuStatsModel.View()

	return lipgloss.JoinVertical(
		lipgloss.Top,
		cpuTab,
		memoryTab,
		networkTab,
		ioTab,
	)
}
