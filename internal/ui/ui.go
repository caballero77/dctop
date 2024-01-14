package ui

import (
	"dctop/internal/configuration"
	"dctop/internal/docker"
	"dctop/internal/ui/helpers"
	"dctop/internal/ui/messages"
	"dctop/internal/ui/stack"
	"dctop/internal/ui/stats"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/viper"
)

type UI struct {
	theme       configuration.Theme
	config      *viper.Viper
	stats       tea.Model
	compose     tea.Model
	selectedTab messages.Tab
	updates     chan docker.ContainerMsg

	width  int
	height int
}

func NewUI(config *viper.Viper, theme configuration.Theme, containersService docker.ContainersService, composeService docker.ComposeService) (ui UI, err error) {
	updates, err := containersService.GetContainerUpdates()
	if err != nil {
		return ui, fmt.Errorf("error getting container updates: %w", err)
	}

	compose, err := stack.New(config, theme, containersService, composeService)
	if err != nil {
		return ui, fmt.Errorf("error creating compose ui model: %w", err)
	}

	statistics := stats.NewStats(theme)

	return UI{
		theme:  theme,
		config: config,
		stats:  statistics,

		compose:     compose,
		selectedTab: messages.Containers,
		updates:     updates,
	}, nil
}

func (model UI) Init() tea.Cmd {
	return helpers.Init(
		model.compose,
		model.stats,
	)
}

func (model UI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	switch msg := msg.(type) {
	case docker.ContainerMsg:
		cmds = append(cmds, waitForActivity(model.updates))
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return model, tea.Quit
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "c":
				cmds = append(cmds, func() tea.Msg { return messages.FocusTabChangedMsg{Tab: messages.Containers} })
			case "t":
				cmds = append(cmds, func() tea.Msg { return messages.FocusTabChangedMsg{Tab: messages.Processes} })
			case "f":
				cmds = append(cmds, func() tea.Msg { return messages.FocusTabChangedMsg{Tab: messages.Compose} })
			}
		}
	case tea.WindowSizeMsg:
		model.width = msg.Width
		model.height = msg.Height

		var stastMsg, composeMsg messages.SizeChangeMsq

		switch {
		case model.height >= 30 && model.width >= 150 || true:
			stastMsg = messages.SizeChangeMsq{Width: msg.Width / 2, Height: msg.Height}
			composeMsg = messages.SizeChangeMsq{Width: msg.Width / 2, Height: msg.Height}
		case model.width < 150:
			stastMsg = messages.SizeChangeMsq{Width: msg.Width, Height: msg.Height / 2}
			composeMsg = messages.SizeChangeMsq{Width: msg.Width, Height: msg.Height / 2}
		}

		return model, helpers.PassMsgs(
			helpers.NewModel(model.compose, func(m tea.Model) { model.compose = m }).WithMsg(composeMsg),
			helpers.NewModel(model.stats, func(m tea.Model) { model.stats = m }).WithMsg(stastMsg),
		)
	}
	cmds = append(cmds, helpers.PassMsg(msg,
		helpers.NewModel(model.compose, func(m tea.Model) { model.compose = m }),
		helpers.NewModel(model.stats, func(m tea.Model) { model.stats = m }),
	))

	return model, tea.Batch(cmds...)
}

func (model UI) View() string {
	switch {
	case model.height >= 30 && model.width >= 160 || true:
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			model.compose.View(),
			model.stats.View(),
		)
	case model.width < 150:
		return lipgloss.JoinVertical(
			lipgloss.Top,
			model.compose.View(),
			model.stats.View(),
		)
	default:
		text := lipgloss.JoinVertical(lipgloss.Center, "Terminal size is too small", fmt.Sprintf("Width = %d Height = %d", model.width, model.height))
		return lipgloss.Place(model.width, model.height, lipgloss.Center, lipgloss.Center, text)
	}
}

func waitForActivity(sub chan docker.ContainerMsg) tea.Cmd {
	return func() tea.Msg {
		return <-sub
	}
}
