package stats

import (
	"github.com/caballero77/dctop/internal/configuration"
	"github.com/caballero77/dctop/internal/ui/helpers"
	"github.com/caballero77/dctop/internal/ui/messages"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Stats struct {
	network          tea.Model
	ioStats          tea.Model
	cpu              tea.Model
	memoryStatsModel tea.Model

	width  int
	height int
}

func NewStats(theme configuration.Theme) Stats {
	network := newNetwork(theme.Sub("network"))
	io := newIO(theme.Sub("io"))
	cpu := newCPU(theme.Sub("cpu"))
	memory := newMemory(theme.Sub("memory"))
	return Stats{
		network:          network,
		ioStats:          io,
		cpu:              cpu,
		memoryStatsModel: memory,
	}
}

func (model Stats) Init() tea.Cmd {
	return helpers.Init(
		model.network,
		model.ioStats,
		model.memoryStatsModel,
		model.cpu,
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
			helpers.NewModel(model.cpu, func(m tea.Model) { model.cpu = m }).WithMsg(cpuSize),
			helpers.NewModel(model.memoryStatsModel, func(m tea.Model) { model.memoryStatsModel = m }).WithMsg(memorySize),
			helpers.NewModel(model.network, func(m tea.Model) { model.network = m }).WithMsg(networkSize),
			helpers.NewModel(model.ioStats, func(m tea.Model) { model.ioStats = m }).WithMsg(ioSize),
		))
		return model, cmd
	}

	commands := make([]tea.Cmd, 0)
	commands = append(commands, helpers.PassMsg(msg,
		helpers.NewModel(model.network, func(m tea.Model) { model.network = m }),
		helpers.NewModel(model.ioStats, func(m tea.Model) { model.ioStats = m }),
		helpers.NewModel(model.memoryStatsModel, func(m tea.Model) { model.memoryStatsModel = m }),
		helpers.NewModel(model.cpu, func(m tea.Model) { model.cpu = m }),
	))

	return model, tea.Batch(commands...)
}

func (model Stats) View() string {
	networkTab := model.network.View()

	ioTab := model.ioStats.View()

	memoryTab := model.memoryStatsModel.View()

	cpuTab := model.cpu.View()

	return lipgloss.JoinVertical(
		lipgloss.Top,
		cpuTab,
		memoryTab,
		networkTab,
		ioTab,
	)
}
