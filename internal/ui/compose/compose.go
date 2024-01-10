package compose

import (
	"dctop/internal/configuration"
	"dctop/internal/docker"
	"dctop/internal/ui/helpers"
	"dctop/internal/ui/messages"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/viper"
)

type Compose struct {
	width  int
	height int

	config *viper.Viper

	containers  tea.Model
	top         tea.Model
	composeFile tea.Model
	logs        tea.Model
}

func New(config *viper.Viper, theme configuration.Theme, service *docker.ComposeService) Compose {
	return Compose{
		containers:  newContainersList(config.GetInt(configuration.ContainersListHeigthName), theme.Sub("containers"), service),
		top:         newTop(config.GetInt(configuration.ProcessesListHeightName), theme.Sub("processes")),
		logs:        newLogs(*service, theme.Sub("logs")),
		composeFile: newComposeFile(service.ComposeFilePath(), theme.Sub("file"), service),
		config:      config,
	}
}

func (model Compose) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg { return messages.FocusTabChangedMsg{Tab: messages.Containers} },
		helpers.Init(
			model.containers,
			model.top,
			model.logs,
			model.composeFile,
		),
	)
}

func (model Compose) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	switch msg := msg.(type) {
	case messages.FocusTabChangedMsg:
		if msg.Tab == messages.Compose {
			cmds = append(cmds, func() tea.Msg { return CloseLogsMsg{} })
		}
	case messages.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height

		var (
			containersHeight = model.config.GetInt(configuration.ContainersListHeigthName) + 3
			processesHeight  = model.config.GetInt(configuration.ProcessesListHeightName) + 3

			containersSize = messages.SizeChangeMsq{Width: msg.Width, Height: containersHeight}
			topSize        = messages.SizeChangeMsq{Width: msg.Width, Height: processesHeight}
			dynamicTabSize = messages.SizeChangeMsq{Width: msg.Width, Height: msg.Height - containersHeight - processesHeight - 2}
		)

		cmd := tea.Batch(helpers.PassMsgs(
			helpers.NewModel(model.containers, func(m tea.Model) { model.containers = m }).WithMsg(containersSize),
			helpers.NewModel(model.top, func(m tea.Model) { model.top = m }).WithMsg(topSize),
			helpers.NewModel(model.logs, func(m tea.Model) { model.logs = m }).WithMsg(dynamicTabSize),
			helpers.NewModel(model.composeFile, func(m tea.Model) { model.composeFile = m }).WithMsg(dynamicTabSize),
		))
		return model, cmd
	}

	cmd := helpers.PassMsg(msg,
		helpers.NewModel(model.containers, func(m tea.Model) { model.containers = m }),
		helpers.NewModel(model.top, func(m tea.Model) { model.top = m }),
		helpers.NewModel(model.logs, func(m tea.Model) { model.logs = m }),
		helpers.NewModel(model.composeFile, func(m tea.Model) { model.composeFile = m }),
	)
	cmds = append(cmds, cmd)

	return model, tea.Batch(cmds...)
}

func (model Compose) View() string {
	containersTab := model.containers.View()

	processesTab := model.top.View()

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
