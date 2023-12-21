package compose

import (
	"dctop/internal/configuration"
	"dctop/internal/docker"
	"dctop/internal/ui/common"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/viper"
)

type Compose struct {
	width  int
	height int

	config *viper.Viper

	containers  tea.Model
	processes   tea.Model
	composeFile tea.Model
	logs        tea.Model
}

func New(config, _ *viper.Viper, service *docker.ComposeService) Compose {
	return Compose{
		containers:  newContainersList(config.GetInt(configuration.ContainersListHeigth), service, true),
		processes:   newProcessesList(config.GetInt(configuration.ProcessesListHeight)),
		logs:        newLogs(*service),
		composeFile: newComposeFile(service.ComposeFilePath()),
		config:      config,
	}
}

func (model Compose) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0)
	var cmd tea.Cmd

	cmd = model.containers.Init()
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	cmd = model.processes.Init()
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	cmd = model.logs.Init()
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	cmd = model.composeFile.Init()
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

func (model Compose) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	var cmd tea.Cmd

	model.containers, cmd = model.containers.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	model.processes, cmd = model.processes.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	model.logs, cmd = model.logs.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	model.composeFile, cmd = model.composeFile.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case common.FocusTabChangedMsg:
		if msg.Tab == common.Compose {
			cmds = append(cmds, func() tea.Msg { return CloseLogsMsg{} })
		}
	case common.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height

		containersHeight := model.config.GetInt(configuration.ContainersListHeigth)
		model.containers, cmd = model.containers.Update(common.SizeChangeMsq{Width: msg.Width, Height: containersHeight})
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		processesHeight := model.config.GetInt(configuration.ProcessesListHeight)
		model.processes, cmd = model.processes.Update(common.SizeChangeMsq{Width: msg.Width, Height: processesHeight})
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		model.logs, cmd = model.logs.Update(common.SizeChangeMsq{Width: msg.Width, Height: msg.Height - containersHeight - processesHeight - 4})
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		model.composeFile, cmd = model.composeFile.Update(common.SizeChangeMsq{Width: msg.Width, Height: msg.Height - containersHeight - processesHeight - 4})
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return model, tea.Batch(cmds...)
}

func (model Compose) View() string {
	containersTab := model.containers.View()

	processesTab := model.processes.View()

	logs := model.logs.View()

	compose := model.composeFile.View()

	var composeColumn string
	if logs == "" {
		composeColumn = lipgloss.JoinVertical(
			lipgloss.Top,
			containersTab,
			processesTab,
			compose,
		)
	} else {
		composeColumn = lipgloss.JoinVertical(
			lipgloss.Top,
			containersTab,
			processesTab,
			logs,
		)
	}

	return composeColumn
}
