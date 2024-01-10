package compose

import (
	"dctop/internal/configuration"
	"dctop/internal/docker"
	"dctop/internal/ui/helpers"
	"dctop/internal/ui/messages"
	"fmt"

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

type CloseLogsMsg struct{}

type logs struct {
	box                 helpers.BoxWithBorders
	stdoutText          tea.Model
	stderrText          tea.Model
	selectedLogType     LogType
	labelStyle          lipgloss.Style
	labeShortcutStyle   lipgloss.Style
	legendStyle         lipgloss.Style
	legendShortcutStyle lipgloss.Style
	service             docker.ComposeService

	width  int
	height int

	done     chan bool
	updates  chan LogsAddedMsg
	open     bool
	selected bool
}

func newLogs(service docker.ComposeService, theme configuration.Theme) logs {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#81A1C1"))

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.plain"))
	labeShortcutStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.shortcut"))

	legendStyle := lipgloss.NewStyle().Foreground(theme.GetColor("legend.plain"))
	legendShortcutStyle := lipgloss.NewStyle().Foreground(theme.GetColor("legend.shortcut"))

	return logs{
		stdoutText:          helpers.NewTextBox("", style),
		stderrText:          helpers.NewTextBox("", style),
		selectedLogType:     Stdout,
		box:                 *helpers.NewBox(theme.Sub("border")),
		labelStyle:          labelStyle,
		labeShortcutStyle:   labeShortcutStyle,
		legendShortcutStyle: legendShortcutStyle,
		legendStyle:         legendStyle,
		service:             service,
		updates:             make(chan LogsAddedMsg),
	}
}

func (logs) Init() tea.Cmd {
	return nil
}

func (model logs) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	var cmd tea.Cmd

	model.stdoutText, cmd = model.stdoutText.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	model.stderrText, cmd = model.stderrText.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case messages.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height

		model.stdoutText, cmd = model.stdoutText.Update(messages.SizeChangeMsq{Width: msg.Width, Height: msg.Height - 2})
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		model.stderrText, cmd = model.stderrText.Update(messages.SizeChangeMsq{Width: msg.Width, Height: msg.Height - 2})
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case CloseLogsMsg:
		cmd = model.close()
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case tea.KeyMsg:
		if model.selected {
			switch msg.Type {
			case tea.KeyEsc:
				cmd = model.close()
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				cmds = append(cmds, func() tea.Msg { return messages.FocusTabChangedMsg{Tab: messages.Containers} })
			case tea.KeyUp:
				if model.selectedLogType == Stdout {
					model.stdoutText, cmd = model.stdoutText.Update(messages.ScrollMsg{Change: -1})
				} else {
					model.stderrText, cmd = model.stderrText.Update(messages.ScrollMsg{Change: -1})
				}
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			case tea.KeyDown:
				if model.selectedLogType == Stdout {
					model.stdoutText, cmd = model.stdoutText.Update(messages.ScrollMsg{Change: 1})
				} else {
					model.stderrText, cmd = model.stderrText.Update(messages.ScrollMsg{Change: 1})
				}
				if cmd != nil {
					cmds = append(cmds, cmd)
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
				cmds = append(cmds, cmd)
			}
		case Stderr:
			model.stderrText, cmd = model.stderrText.Update(messages.AppendTextMgs{Text: string(msg.Message), AdjustScroll: true})
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		cmds = append(cmds, model.waitForLogs())
	case messages.FocusTabChangedMsg:
		model.selected = msg.Tab == messages.Logs
	case messages.StartListenningLogsMsg:
		if !model.open {
			model.open = true
			stdout, stderr, e, done := model.service.GetContainerLogs(msg.ContainerID, "100")
			model.done = done
			go func() {
				for {
					select {
					case log := <-stdout:
						model.updates <- LogsAddedMsg{LogType: Stdout, Message: log}
					case log := <-stderr:
						model.updates <- LogsAddedMsg{LogType: Stderr, Message: log}
					case <-done:
						return
					case err := <-e:
						if err.Error() != "EOF" {
							panic(err)
						}
					}
				}
			}()
			cmds = append(cmds, model.waitForLogs())
		}
	}
	return model, tea.Batch(cmds...)
}

func (model logs) View() string {
	if !model.open {
		return ""
	}
	labels := []string{model.label()}
	legends := []string{model.legend()}

	var text string

	if model.selectedLogType == Stdout {
		text = model.stdoutText.View()
	} else {
		text = model.stderrText.View()
	}

	return model.box.Render(labels, legends, text, model.open)
}

func (model logs) label() string {
	return model.labeShortcutStyle.Render("L") + model.labelStyle.Render(fmt.Sprintf("ogs: %s", model.selectedLogType))
}

func (model logs) legend() string {
	var stdout string
	var stderr string

	if model.selectedLogType == Stdout {
		stdout = model.legendShortcutStyle.Copy().Bold(true).Render("¹") + model.legendStyle.Copy().Bold(true).Render("stdout")
		stderr = model.legendShortcutStyle.Render("²") + model.legendStyle.Render("stderr")
	} else {
		stdout = model.legendShortcutStyle.Render("¹") + model.legendStyle.Render("stdout")
		stderr = model.legendShortcutStyle.Copy().Bold(true).Render("²") + model.legendStyle.Copy().Bold(true).Render("stderr")
	}

	return stdout + " " + stderr
}

func (model *logs) close() tea.Cmd {
	if !model.open {
		return nil
	}
	model.open = false
	close(model.done)

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
