package helpers

import (
	"dctop/internal/configuration"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type BoxedModel interface {
	tea.Model

	Legends() []string
	Labels() []string
	Focus() bool
	UpdateAsBoxed(msg tea.Msg) (BoxedModel, tea.Cmd)
}

type UpdateLabelsMsg struct {
	Labels []string
}

type BoxWithBorders struct {
	innerModel BoxedModel
	border     lipgloss.Border
	color      lipgloss.Color
	focusColor lipgloss.Color
}

func NewBox(model BoxedModel, theme configuration.Theme) BoxWithBorders {
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
	color := theme.GetColor("plain")
	focusColor := theme.GetColor("focus")
	return BoxWithBorders{
		border:     border,
		color:      color,
		focusColor: focusColor,
		innerModel: model,
	}
}

func (model BoxWithBorders) Init() tea.Cmd {
	return model.innerModel.Init()
}

func (model BoxWithBorders) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	model.innerModel, cmd = model.innerModel.UpdateAsBoxed(msg)
	return model, cmd
}

func (model BoxWithBorders) View() string {
	labels := model.innerModel.Labels()
	legends := model.innerModel.Legends()
	focus := model.innerModel.Focus()

	borderColor := model.color
	if focus {
		borderColor = model.focusColor
	}
	borderStyle := lipgloss.NewStyle().Foreground(borderColor).BorderForeground(borderColor)
	border := model.border

	topLeft := borderStyle.Render(border.TopLeft)
	topRight := borderStyle.Render(border.TopRight)
	top := borderStyle.Render(border.Top)

	bottomLeft := borderStyle.Render(border.BottomLeft)
	bottomRight := borderStyle.Render(border.BottomRight)
	bottom := borderStyle.Render(border.Bottom)

	content := model.innerModel.View()

	width := lipgloss.Width(content)

	borderWidth := lipgloss.NewStyle().Border(model.border).GetHorizontalBorderSize()

	var topBorder string
	if len(labels) == 0 {
		gap := strings.Repeat(border.Top, width)
		topBorder = topLeft + borderStyle.Render(gap) + topRight
	} else {
		sep := borderStyle.Render(topRight + border.Top + topLeft)
		label := strings.Join(labels, sep)
		cellsShort := max(0, width+borderWidth-lipgloss.Width(topLeft+topLeft+topRight+topRight+label+top))
		gap := strings.Repeat(border.Top, cellsShort)
		topBorder = topLeft + top + topRight + label + topLeft + borderStyle.Render(gap) + topRight
	}

	var bottomBorder string
	if len(legends) == 0 {
		gap := strings.Repeat(border.Bottom, width)
		bottomBorder = bottomLeft + borderStyle.Render(gap) + bottomRight
	} else {
		sep := borderStyle.Render(border.Top)
		legend := strings.Join(adjustLegendLegth(legends, width), sep)
		cellsShort := max(0, width+borderWidth-lipgloss.Width(bottomLeft+bottomRight+legend+bottom))
		gap := strings.Repeat(border.Top, cellsShort)
		bottomBorder = bottomLeft + bottom + legend + borderStyle.Render(gap) + bottomRight
	}

	middle := borderStyle.Border(model.border).
		BorderTop(false).
		BorderBottom(false).
		Width(width).
		Render(content)

	return topBorder + "\n" + middle + "\n" + bottomBorder
}

func adjustLegendLegth(legends []string, width int) []string {
	length := 0
	for _, legend := range legends {
		length += lipgloss.Width(legend)
	}
	i := len(legends) - 1
	for len(legends) != 0 && i >= 0 {
		if length+2 >= width {
			length -= len(legends[i])
			legends = legends[:i]
			i--
		} else {
			break
		}
	}
	return legends
}
