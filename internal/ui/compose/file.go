package compose

import (
	"dctop/internal/configuration"
	"dctop/internal/docker"
	"dctop/internal/ui/common"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type file struct {
	box     common.BoxWithBorders
	text    tea.Model
	service *docker.ComposeService

	width  int
	height int

	composeFile []string
	focus       bool

	label  string
	legend string
}

func newComposeFile(path string, theme configuration.Theme, service *docker.ComposeService) file {
	bytes, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	composeFile := string(bytes)

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.plain"))
	labeShortcutStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.shortcut"))

	legendStyle := lipgloss.NewStyle().Foreground(theme.GetColor("legend.plain"))
	legendShortcutStyle := lipgloss.NewStyle().Foreground(theme.GetColor("legend.shortcut"))

	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#81A1C1"))

	return file{
		text:        common.NewTextBox(composeFile, style),
		box:         *common.NewBoxWithLabel(theme.Sub("border")),
		service:     service,
		composeFile: strings.Split(composeFile, "\n"),
		label:       labelStyle.Render("Compose ") + labeShortcutStyle.Render("f") + labelStyle.Render("ile"),
		legend:      legendShortcutStyle.Render("u") + legendStyle.Render("p") + " " + legendShortcutStyle.Render("d") + legendStyle.Render("own"),
	}
}

func (file) Init() tea.Cmd {
	return nil
}

func (model file) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	var cmd tea.Cmd

	model.text, cmd = model.text.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case common.FocusTabChangedMsg:
		model.focus = msg.Tab == common.Compose
	case common.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height

		model.text, cmd = model.text.Update(common.SizeChangeMsq{Width: msg.Width, Height: msg.Height - 2})
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case tea.KeyMsg:
		if model.focus {
			switch msg.Type {
			case tea.KeyUp:
				model.text, cmd = model.text.Update(common.ScrollMsg{Change: -1})
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			case tea.KeyDown:
				model.text, cmd = model.text.Update(common.ScrollMsg{Change: 1})
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			case tea.KeyRunes:
				if !model.focus {
					return model, tea.Batch(cmds...)
				}
				switch string(msg.Runes) {
				case "u":
					return model, func() tea.Msg {
						err := model.service.ComposeUp()
						if err != nil {
							panic(err)
						}
						return nil
					}
				case "d":
					return model, func() tea.Msg {
						err := model.service.ComposeDown()
						if err != nil {
							panic(err)
						}
						return nil
					}
				}
			}
		}
	}
	return model, tea.Batch(cmds...)
}

func (model file) View() string {
	legends := []string{}
	if model.focus {
		legends = []string{model.legend}
	}
	return model.box.Render([]string{model.label}, legends, model.text.View(), model.focus)
}
