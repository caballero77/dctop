package stack

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/caballero77/dctop/internal/configuration"
	"github.com/caballero77/dctop/internal/docker"
	"github.com/caballero77/dctop/internal/ui/helpers"
	"github.com/caballero77/dctop/internal/ui/messages"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type LogType string

const (
	Stdout LogType = "stdout"
	Stderr LogType = "stderr"
)

type LogsAddedMsg struct {
	Message []byte
	LogType LogType
}

type logs struct {
	stdoutText          tea.Model
	stderrText          tea.Model
	selectedLogType     LogType
	labelStyle          lipgloss.Style
	labeShortcutStyle   lipgloss.Style
	legendStyle         lipgloss.Style
	legendShortcutStyle lipgloss.Style
	containersService   docker.ContainersService

	width  int
	height int

	cancel   func()
	updates  chan LogsAddedMsg
	open     bool
	selected bool
}

func newLogs(containersService docker.ContainersService, theme configuration.Theme) tea.Model {
	style := lipgloss.NewStyle().Foreground(theme.GetColor("body.text"))

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.plain"))
	labeShortcutStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.shortcut"))

	legendStyle := lipgloss.NewStyle().Foreground(theme.GetColor("legend.plain"))
	legendShortcutStyle := lipgloss.NewStyle().Foreground(theme.GetColor("legend.shortcut"))

	scrollStyle := lipgloss.NewStyle().
		Foreground(theme.GetColor("scroll.foreground")).
		Background(theme.GetColor("scroll.background"))

	model := logs{
		stdoutText:          helpers.NewTextBox("", style, scrollStyle),
		stderrText:          helpers.NewTextBox("", style, scrollStyle),
		selectedLogType:     Stdout,
		labelStyle:          labelStyle,
		labeShortcutStyle:   labeShortcutStyle,
		legendShortcutStyle: legendShortcutStyle,
		legendStyle:         legendStyle,
		containersService:   containersService,
		updates:             make(chan LogsAddedMsg),
	}

	return helpers.NewBox(model, theme.Sub("border"))
}

// Focus implements helpers.BoxedModel.
func (model logs) Focus() bool { return model.open }

// Labels implements helpers.BoxedModel.
func (model logs) Labels() []string {
	return []string{model.labeShortcutStyle.Render("L") + model.labelStyle.Render(fmt.Sprintf("ogs: %s", model.selectedLogType))}
}

// Legends implements helpers.BoxedModel.
func (model logs) Legends() []string {
	var stdout string
	var stderr string

	if model.selectedLogType == Stdout {
		stdout = model.legendShortcutStyle.Copy().Bold(true).Render("¹") + model.legendStyle.Copy().Bold(true).Render("stdout")
		stderr = model.legendShortcutStyle.Render("²") + model.legendStyle.Render("stderr")
	} else {
		stdout = model.legendShortcutStyle.Render("¹") + model.legendStyle.Render("stdout")
		stderr = model.legendShortcutStyle.Copy().Bold(true).Render("²") + model.legendStyle.Copy().Bold(true).Render("stderr")
	}

	return []string{stdout + " " + stderr}
}

func (model logs) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return model.UpdateAsBoxed(msg) }

func (logs) Init() tea.Cmd {
	return nil
}

func (model logs) UpdateAsBoxed(msg tea.Msg) (helpers.BoxedModel, tea.Cmd) {
	commands := make([]tea.Cmd, 0)
	var cmd tea.Cmd

	model.stdoutText, cmd = model.stdoutText.Update(msg)
	if cmd != nil {
		commands = append(commands, cmd)
	}

	model.stderrText, cmd = model.stderrText.Update(msg)
	if cmd != nil {
		commands = append(commands, cmd)
	}

	switch msg := msg.(type) {
	case messages.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height

		model.stdoutText, cmd = model.stdoutText.Update(messages.SizeChangeMsq{Width: msg.Width, Height: msg.Height - 2})
		if cmd != nil {
			commands = append(commands, cmd)
		}

		model.stderrText, cmd = model.stderrText.Update(messages.SizeChangeMsq{Width: msg.Width, Height: msg.Height - 2})
		if cmd != nil {
			commands = append(commands, cmd)
		}
	case messages.CloseTabMsg:
		if msg.Tab == messages.Logs {
			cmd = model.close()
			if cmd != nil {
				commands = append(commands, cmd)
			}
		}
	case tea.KeyMsg:
		if model.selected {
			switch msg.Type {
			case tea.KeyUp:
				if model.selectedLogType == Stdout {
					model.stdoutText, cmd = model.stdoutText.Update(messages.ScrollMsg{Change: -1})
				} else {
					model.stderrText, cmd = model.stderrText.Update(messages.ScrollMsg{Change: -1})
				}
				if cmd != nil {
					commands = append(commands, cmd)
				}
			case tea.KeyDown:
				if model.selectedLogType == Stdout {
					model.stdoutText, cmd = model.stdoutText.Update(messages.ScrollMsg{Change: 1})
				} else {
					model.stderrText, cmd = model.stderrText.Update(messages.ScrollMsg{Change: 1})
				}
				if cmd != nil {
					commands = append(commands, cmd)
				}
			case tea.KeyRunes:
				switch string(msg.Runes) {
				case "1":
					model.selectedLogType = Stdout
				case "2":
					model.selectedLogType = Stderr
				}
			}
		}

	case LogsAddedMsg:
		switch msg.LogType {
		case Stdout:
			model.stdoutText, cmd = model.stdoutText.Update(messages.AppendTextMgs{Text: string(msg.Message), AdjustScroll: true})
			if cmd != nil {
				commands = append(commands, cmd)
			}
		case Stderr:
			model.stderrText, cmd = model.stderrText.Update(messages.AppendTextMgs{Text: string(msg.Message), AdjustScroll: true})
			if cmd != nil {
				commands = append(commands, cmd)
			}
		}
		commands = append(commands, model.waitForLogs())
	case messages.FocusTabChangedMsg:
		if msg.Tab.IsDetailsTab() && msg.Tab != messages.Logs {
			cmd = model.close()
			if cmd != nil {
				commands = append(commands, cmd)
			}
		} else {
			model.selected = msg.Tab == messages.Logs
		}
	case messages.StartListeningLogsMsg:
		if !model.open {
			model.open = true
			ctx, cancel := context.WithCancel(context.Background())
			stdout, stderr, e := model.containersService.GetContainerLogs(ctx, msg.ContainerID, "100")
			model.cancel = cancel
			go func() {
				for {
					select {
					case log := <-stdout:
						model.updates <- LogsAddedMsg{LogType: Stdout, Message: log}
					case log := <-stderr:
						model.updates <- LogsAddedMsg{LogType: Stderr, Message: log}
					case err := <-e:
						if err != nil && err.Error() != "EOF" {
							slog.Error("error reading logs",
								"id", msg.ContainerID,
								"error", err)
						}
					}
				}
			}()
			commands = append(commands, model.waitForLogs())
		}
	}
	return model, tea.Batch(commands...)
}

func (model logs) View() string {
	if !model.open {
		return ""
	}
	var text string

	if model.selectedLogType == Stdout {
		text = model.stdoutText.View()
	} else {
		text = model.stderrText.View()
	}

	return text
}

func (model *logs) close() tea.Cmd {
	if !model.open {
		return nil
	}
	model.open = false
	model.cancel()

	cmds := make([]tea.Cmd, 0)
	var cmd tea.Cmd

	model.stdoutText, cmd = model.stdoutText.Update(messages.ClearTextBoxMsg{})
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	model.stderrText, cmd = model.stderrText.Update(messages.ClearTextBoxMsg{})
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

func (model logs) waitForLogs() tea.Cmd {
	return func() tea.Msg {
		return <-model.updates
	}
}
