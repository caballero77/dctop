package compose

import (
	"dctop/internal/ui/common"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type file struct {
	box         common.BoxWithBorders
	text        tea.Model
	labelStyle  lipgloss.Style
	legendStyle lipgloss.Style

	width  int
	height int

	composeFile []string
	focus       bool
}

func newComposeFile(path string) file {
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

	bytes, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	composeFile := string(bytes)

	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#81A1C1"))

	return file{
		text:        common.NewTextBox(composeFile, style),
		box:         *common.NewBoxWithLabel(border, borderStyle, focusBorderStyle),
		labelStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("#D8DEE9")),
		legendStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#8FBCBB")),
		composeFile: strings.Split(composeFile, "\n"),
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
			}
		}
	}

	return model, tea.Batch(cmds...)
}

func (model file) View() string {
	label := model.labelStyle.Render("Compose file")
	return model.box.Render([]string{label}, []string{}, model.text.View(), model.focus)
}
