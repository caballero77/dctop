package stats

import (
	"dctop/internal/ui/common"

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

func NewStats() Stats {
	return Stats{
		networkModel:     newNetwork(),
		ioStatsModel:     newIO(),
		cpuStatsModel:    newCPU(),
		memoryStatsModel: newMemory(),
	}
}

func (model Stats) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0)
	var cmd tea.Cmd

	cmd = model.networkModel.Init()
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	cmd = model.ioStatsModel.Init()
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	cmd = model.cpuStatsModel.Init()
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	cmd = model.memoryStatsModel.Init()
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

func (model Stats) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	var cmd tea.Cmd

	model.cpuStatsModel, cmd = model.cpuStatsModel.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	model.memoryStatsModel, cmd = model.memoryStatsModel.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	model.networkModel, cmd = model.networkModel.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	model.ioStatsModel, cmd = model.ioStatsModel.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	if msg, ok := msg.(common.SizeChangeMsq); ok {
		model.width = msg.Width
		model.height = msg.Height

		model.cpuStatsModel, cmd = model.cpuStatsModel.Update(common.SizeChangeMsq{Width: msg.Width, Height: msg.Height - 3*(msg.Height/5)})
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		model.memoryStatsModel, cmd = model.memoryStatsModel.Update(common.SizeChangeMsq{Width: msg.Width, Height: msg.Height / 5})
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		model.networkModel, cmd = model.networkModel.Update(common.SizeChangeMsq{Width: msg.Width, Height: msg.Height / 5})
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		model.ioStatsModel, cmd = model.ioStatsModel.Update(common.SizeChangeMsq{Width: msg.Width, Height: msg.Height / 5})
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

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
