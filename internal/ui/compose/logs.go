package compose

import (
	"dctop/internal/docker"
	"dctop/internal/ui/common"
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

type logs struct {
	box             common.BoxWithBorders
	stdoutText      tea.Model
	stderrText      tea.Model
	selectedLogType LogType
	labelStyle      lipgloss.Style
	legendStyle     lipgloss.Style
	service         docker.ComposeService

	width  int
	height int

	done    chan bool
	updates chan LogsAddedMsg
	open    bool
}

func newLogs(service docker.ComposeService) logs {
	border := lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
	}
	borderStyle := lipgloss.Color("#434C5E")
	focusBorderStyle := lipgloss.Color("#8FBCBB")
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#81A1C1"))
	return logs{
		stdoutText:      common.NewTextBox("", style),
		stderrText:      common.NewTextBox("", style),
		selectedLogType: Stdout,
		box:             *common.NewBoxWithLabel(border, borderStyle, focusBorderStyle),
		labelStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("#D8DEE9")),
		legendStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("#8FBCBB")),
		service:         service,
		updates:         make(chan LogsAddedMsg),
		open:            false,
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
	case common.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height

		model.stdoutText, cmd = model.stdoutText.Update(common.SizeChangeMsq{Width: msg.Width, Height: msg.Height - 2})
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		model.stderrText, cmd = model.stderrText.Update(common.SizeChangeMsq{Width: msg.Width, Height: msg.Height - 2})
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case tea.KeyMsg:
		if model.open {
			switch msg.Type {
			case tea.KeyEsc:
				model.open = false
				close(model.done)

				model.stdoutText, cmd = model.stdoutText.Update(common.ClearTextBoxMsg{})
				if cmd != nil {
					cmds = append(cmds, cmd)
				}

				cmds = append(cmds, func() tea.Msg { return common.FocusTabChangedMsg{Tab: common.Containers} })
			case tea.KeyUp:
				if model.selectedLogType == Stdout {
					model.stdoutText, cmd = model.stdoutText.Update(common.ScrollMsg{Change: -1})
				} else {
					model.stderrText, cmd = model.stderrText.Update(common.ScrollMsg{Change: -1})
				}
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			case tea.KeyDown:
				if model.selectedLogType == Stdout {
					model.stdoutText, cmd = model.stdoutText.Update(common.ScrollMsg{Change: 1})
				} else {
					model.stderrText, cmd = model.stderrText.Update(common.ScrollMsg{Change: 1})
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
			model.stdoutText, cmd = model.stdoutText.Update(common.AppendTextMgs{Text: string(msg.Message), AdjustScroll: true})
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		case Stderr:
			model.stderrText, cmd = model.stderrText.Update(common.AppendTextMgs{Text: string(msg.Message), AdjustScroll: true})
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		cmds = append(cmds, model.waitForLogs())
	case common.StartListenningLogsMsg:
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
						panic(err)
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
	labels := []string{model.labelStyle.Render(fmt.Sprintf("Logs: %s", model.selectedLogType))}
	legends := []string{model.legendStyle.Render("¹stdout ²stderr")}

	var text string

	if model.selectedLogType == Stdout {
		text = model.stdoutText.View()
	} else {
		text = model.stderrText.View()
	}

	return model.box.Render(labels, legends, text, model.open)
}

func (model logs) waitForLogs() tea.Cmd {
	return func() tea.Msg {
		return <-model.updates
	}
}
