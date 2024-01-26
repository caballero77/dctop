package stack

import (
	"dctop/internal/configuration"
	"dctop/internal/docker"
	"dctop/internal/ui/helpers"
	"dctop/internal/ui/messages"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/viper"
)

type Stack struct {
	width  int
	height int

	config *viper.Viper

	containers tea.Model
	top        tea.Model
	compose    tea.Model
	logs       tea.Model
	inspect    tea.Model

	activeDetailsTab messages.Tab
	activeTab        messages.Tab
}

func New(config *viper.Viper, theme configuration.Theme, containersService docker.ContainersService, composeService docker.ComposeService) (stack Stack, err error) {
	top := newTop(config.GetInt(configuration.ProcessesListHeightName), theme.Sub("processes"))

	compose, err := newCompose(theme.Sub("file"), composeService)
	if err != nil {
		return stack, fmt.Errorf("error creating compose file model: %w", err)
	}

	containers, err := newContainersList(config.GetInt(configuration.ContainersListHeigthName), theme.Sub("containers"), containersService)
	if err != nil {
		return stack, fmt.Errorf("error creating containers list model: %w", err)
	}

	logs := newLogs(containersService, theme.Sub("logs"))
	inspect := newInspect(theme.Sub("inspect"))

	return Stack{
		containers:       containers,
		top:              top,
		logs:             logs,
		inspect:          inspect,
		compose:          compose,
		config:           config,
		activeDetailsTab: messages.Compose,
		activeTab:        messages.Containers,
	}, nil
}

func (model Stack) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg { return messages.FocusTabChangedMsg{Tab: messages.Containers} },
		helpers.Init(
			model.containers,
			model.top,
			model.logs,
			model.compose,
			model.inspect,
		),
	)
}

func (model Stack) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEsc {
			if model.activeDetailsTab == model.activeTab {
				cmds = append(cmds, func() tea.Msg { return messages.FocusTabChangedMsg{Tab: messages.Containers} })
			}

			tabToClose := model.activeDetailsTab
			cmds = append(cmds, func() tea.Msg { return messages.CloseTabMsg{Tab: tabToClose} })
			model.activeDetailsTab = messages.Compose
		}
	case messages.FocusTabChangedMsg:
		model.activeTab = msg.Tab
		if msg.Tab.IsDetailsTab() {
			model.activeDetailsTab = msg.Tab
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
			helpers.NewModel(model.compose, func(m tea.Model) { model.compose = m }).WithMsg(dynamicTabSize),
			helpers.NewModel(model.inspect, func(m tea.Model) { model.inspect = m }).WithMsg(dynamicTabSize),
		))
		return model, cmd
	}

	cmd := helpers.PassMsg(msg,
		helpers.NewModel(model.containers, func(m tea.Model) { model.containers = m }),
		helpers.NewModel(model.top, func(m tea.Model) { model.top = m }),
		helpers.NewModel(model.logs, func(m tea.Model) { model.logs = m }),
		helpers.NewModel(model.compose, func(m tea.Model) { model.compose = m }),
		helpers.NewModel(model.inspect, func(m tea.Model) { model.inspect = m }),
	)
	cmds = append(cmds, cmd)

	return model, tea.Batch(cmds...)
}

func (model Stack) View() string {
	containersTab := model.containers.View()

	processesTab := model.top.View()

	switch model.activeDetailsTab {
	case messages.Compose:
		compose := model.compose.View()
		return lipgloss.JoinVertical(
			lipgloss.Top,
			containersTab,
			processesTab,
			compose,
		)
	case messages.Logs:
		logs := model.logs.View()
		return lipgloss.JoinVertical(
			lipgloss.Top,
			containersTab,
			processesTab,
			logs,
		)
	case messages.Inspect:
		inspect := model.inspect.View()
		return lipgloss.JoinVertical(
			lipgloss.Top,
			containersTab,
			processesTab,
			inspect,
		)
	default:
		return ""
	}
}
