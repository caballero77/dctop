package helpers

import (
	"dctop/internal/ui/messages"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TextBox struct {
	width  int
	height int

	style       lipgloss.Style
	scrollStyle lipgloss.Style

	text           string
	lines          []string
	scrollPosition int
}

func NewTextBox(text string, style lipgloss.Style) TextBox {
	return TextBox{
		text:        text,
		style:       style,
		scrollStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#D8DEE9")),
	}
}

func (TextBox) Init() tea.Cmd {
	return nil
}

func (model TextBox) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height
		model.lines = model.getLines(model.text)
	case messages.ScrollMsg:
		if msg.Change > 0 {
			model = model.ScrollDown(msg.Change)
		} else if msg.Change < 0 {
			model = model.ScrollUp(-msg.Change)
		}
	case messages.AppendTextMgs:
		model.text += msg.Text
		newLines := model.getLines(msg.Text)
		model.lines = append(model.lines, newLines...)
		if msg.AdjustScroll {
			model = model.ScrollDown(len(newLines))
		}
	case messages.ClearTextBoxMsg:
		model.lines = []string{}
		model.text = ""
		model.scrollPosition = 0
	}
	return model, nil
}

func (model TextBox) View() string {
	if model.lines == nil || len(model.lines) == 0 {
		return lipgloss.Place(model.width-2, model.height, lipgloss.Center, lipgloss.Center, "empty")
	}
	height := model.height
	lines := model.lines

	scrollBar := model.scrollStyle.Render(renderScrollBar(len(lines), height, model.scrollPosition))
	if len(lines) > height {
		lines = lines[model.scrollPosition : model.scrollPosition+height]
	}

	text := model.style.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))

	return lipgloss.PlaceVertical(height,
		lipgloss.Top,
		lipgloss.JoinHorizontal(lipgloss.Left, text, scrollBar),
	)
}

func (model TextBox) Append(value string) (resultModel TextBox, n int) {
	if value != "" {
		model.text += value
		newLines := model.getLines(value)
		model.lines = append(model.lines, newLines...)
		return model, len(newLines)
	}
	return model, 0
}

func (model TextBox) ScrollUp(n int) TextBox {
	if n <= 0 {
		return model
	}
	model.scrollPosition = max(model.scrollPosition-n, 0)
	return model
}

func (model TextBox) ScrollDown(n int) TextBox {
	if n <= 0 {
		return model
	}
	model.scrollPosition = min(model.scrollPosition+n, len(model.lines)-model.height)
	return model
}

func (model TextBox) Clear() TextBox {
	model.lines = []string{}
	model.text = ""
	model.scrollPosition = 0
	return model
}

func (model *TextBox) getLines(text string) []string {
	if text == "" || model.height == 0 || model.width == 0 {
		return []string{}
	}
	width := model.width - 3
	lines := make([]string, 0)
	j := 0
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			continue
		}
		for len(line) > width {
			lines = append(lines, line[:width])
			j++
			line = line[width:]
		}
		if line != "" {
			lines = append(lines, line+strings.Repeat(" ", width-len(line)))
			j++
		}
	}
	return lines
}
