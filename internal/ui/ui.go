package ui

import (
	"dctop/internal/configuration"
	"dctop/internal/docker"
	"dctop/internal/ui/common"
	"dctop/internal/ui/compose"
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
	selectedTab common.Tab
	service     *docker.ComposeService

	width  int
	height int
}

func NewUI(config *viper.Viper, theme configuration.Theme, service *docker.ComposeService) UI {

	return UI{
		theme:  theme,
		config: config,
		stats:  stats.NewStats(theme),

		compose:     compose.New(config, theme, service),
		selectedTab: common.Containers,
		service:     service,
	}
}

func (model UI) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0)
	var cmd tea.Cmd

	cmd = model.stats.Init()
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	cmd = model.compose.Init()
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

func (model UI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	var cmd tea.Cmd

	model.stats, cmd = model.stats.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	model.compose, cmd = model.compose.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return model, tea.Quit
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "c":
				cmds = append(cmds, func() tea.Msg { return common.FocusTabChangedMsg{Tab: common.Containers} })
			case "t":
				cmds = append(cmds, func() tea.Msg { return common.FocusTabChangedMsg{Tab: common.Processes} })
			case "f":
				cmds = append(cmds, func() tea.Msg { return common.FocusTabChangedMsg{Tab: common.Compose} })
			}
		}
	case tea.WindowSizeMsg:
		model.width = msg.Width
		model.height = msg.Height

		var stastMsg, composeMsg common.SizeChangeMsq

		switch {
		case model.height >= 30 && model.width >= 150 || true:
			stastMsg = common.SizeChangeMsq{Width: msg.Width / 2, Height: msg.Height}
			composeMsg = common.SizeChangeMsq{Width: msg.Width / 2, Height: msg.Height}
		case model.width < 150:
			stastMsg = common.SizeChangeMsq{Width: msg.Width, Height: msg.Height / 2}
			composeMsg = common.SizeChangeMsq{Width: msg.Width, Height: msg.Height / 2}
		}

		model.stats, cmd = model.stats.Update(stastMsg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		model.compose, cmd = model.compose.Update(composeMsg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

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
