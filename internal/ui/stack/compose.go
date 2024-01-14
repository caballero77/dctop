package stack

import (
	"dctop/internal/configuration"
	"dctop/internal/docker"
	"dctop/internal/ui/helpers"
	"dctop/internal/ui/messages"
	"fmt"
	"log/slog"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type compose struct {
	box               helpers.BoxWithBorders
	text              tea.Model
	containersService docker.ComposeService

	width  int
	height int

	composeFile []string
	focus       bool

	label  string
	legend string
}

func newCompose(theme configuration.Theme, containersService docker.ComposeService) (model compose, err error) {
	bytes, err := os.ReadFile(containersService.FilePath())
	if err != nil {
		return model, fmt.Errorf("error reading compose file: %w", err)
	}
	composeFile := string(bytes)

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.plain"))
	labeShortcutStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.shortcut"))

	legendStyle := lipgloss.NewStyle().Foreground(theme.GetColor("legend.plain"))
	legendShortcutStyle := lipgloss.NewStyle().Foreground(theme.GetColor("legend.shortcut"))

	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#81A1C1"))

	return compose{
		text:              helpers.NewTextBox(composeFile, style),
		box:               helpers.NewBox(theme.Sub("border")),
		containersService: containersService,
		composeFile:       strings.Split(composeFile, "\n"),
		label:             labelStyle.Render("Compose ") + labeShortcutStyle.Render("f") + labelStyle.Render("ile"),
		legend:            legendShortcutStyle.Render("u") + legendStyle.Render("p") + " " + legendShortcutStyle.Render("d") + legendStyle.Render("own"),
	}, err
}

func (compose) Init() tea.Cmd {
	return nil
}

func (model compose) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	var cmd tea.Cmd

	model.text, cmd = model.text.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case messages.FocusTabChangedMsg:
		model.focus = msg.Tab == messages.Compose
	case messages.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height

		model.text, cmd = model.text.Update(messages.SizeChangeMsq{Width: msg.Width, Height: msg.Height - 2})
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case tea.KeyMsg:
		if model.focus {
			switch msg.Type {
			case tea.KeyUp:
				model.text, cmd = model.text.Update(messages.ScrollMsg{Change: -1})
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			case tea.KeyDown:
				model.text, cmd = model.text.Update(messages.ScrollMsg{Change: 1})
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
						err := model.containersService.ComposeUp()
						if err != nil {
							slog.Error("error performing compose up", "Error", err)
						}
						return nil
					}
				case "d":
					return model, func() tea.Msg {
						err := model.containersService.ComposeDown()
						if err != nil {
							slog.Error("error performing compose down", "Error", err)
						}
						return nil
					}
				}
			}
		}
	}
	return model, tea.Batch(cmds...)
}

func (model compose) View() string {
	legends := []string{}
	if model.focus {
		legends = []string{model.legend}
	}
	return model.box.Render([]string{model.label}, legends, model.text.View(), model.focus)
}
