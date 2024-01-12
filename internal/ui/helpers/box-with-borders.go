package helpers

import (
	"dctop/internal/configuration"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type BoxWithBorders struct {
	border     lipgloss.Border
	color      lipgloss.Color
	focusColor lipgloss.Color
}

func NewBox(theme configuration.Theme) BoxWithBorders {
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
	}
}

func (b BoxWithBorders) Render(labels, legends []string, content string, focus bool) string {
	borderColor := b.color
	if focus {
		borderColor = b.focusColor
	}
	borderStyle := lipgloss.NewStyle().Foreground(borderColor).BorderForeground(borderColor)
	border := b.border

	topLeft := borderStyle.Render(border.TopLeft)
	topRight := borderStyle.Render(border.TopRight)
	top := borderStyle.Render(border.Top)

	bottomLeft := borderStyle.Render(border.BottomLeft)
	bottomRight := borderStyle.Render(border.BottomRight)
	bottom := borderStyle.Render(border.Bottom)

	width := lipgloss.Width(content)

	borderWidth := lipgloss.NewStyle().Border(b.border).GetHorizontalBorderSize()

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

	middle := borderStyle.Border(b.border).
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
