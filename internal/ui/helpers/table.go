package helpers

import (
	"dctop/internal/configuration"
	"errors"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Table struct {
	headerCellStyle  lipgloss.Style
	bodyCellStyle    lipgloss.Style
	selectedRowStyle lipgloss.Style
	scrollStyle      lipgloss.Style
	getSizes         func(int) []int
}

func NewTable(getSizes func(int) []int, theme configuration.Theme) *Table {
	headerCellStyle := lipgloss.
		NewStyle().
		Foreground(theme.GetColor("header.foreground")).
		Background(theme.GetColor("header.background"))

	bodyCellStyle := lipgloss.
		NewStyle().
		Foreground(theme.GetColor("row.plain.foreground")).
		Background(theme.GetColor("row.plain.background"))

	selectedCellStyle := lipgloss.
		NewStyle().
		Foreground(theme.GetColor("row.selected.foreground")).
		Background(theme.GetColor("row.selected.background"))

	scrollStyle := lipgloss.
		NewStyle().
		Foreground(theme.GetColor("scroll.foreground")).
		Background(theme.GetColor("scroll.background"))

	return &Table{
		headerCellStyle:  headerCellStyle,
		bodyCellStyle:    bodyCellStyle,
		selectedRowStyle: selectedCellStyle,
		scrollStyle:      scrollStyle,
		getSizes:         getSizes,
	}
}

func (table *Table) Render(headerCells []string, rowCells [][]string, width, selected, scrollPosition, height int) (string, error) {
	width -= 3
	height--

	var scrollBar string
	if height > 0 && len(rowCells) > height {
		pos := int(float64(scrollPosition) * float64(height) / float64(len(rowCells)-height))
		var newScrollBar string
		if height == pos {
			newScrollBar = strings.Repeat("\n", height-1) + "█"
		} else {
			newScrollBar = strings.Repeat("\n", pos) + "█" + strings.Repeat("\n", height-pos)
		}
		scrollBar = table.scrollStyle.Render(newScrollBar)
		rowCells = rowCells[scrollPosition : scrollPosition+height]
	} else {
		scrollBar = table.scrollStyle.Render(strings.Repeat(" \n", len(rowCells)))
	}

	size := table.getSizes(width)
	if len(headerCells) != len(size) {
		return "", errors.New("unexpected header length")
	}

	header := table.renderCells(headerCells, width, size, table.headerCellStyle)

	rows := make([]string, len(rowCells))
	for i, row := range rowCells {
		style := table.bodyCellStyle
		if i+scrollPosition == selected {
			style = table.selectedRowStyle
		}
		rows[i] = table.renderCells(row, width, size, style)
	}

	if len(rows) < height {
		emptyRows := make([]string, height-len(rows))
		for i := 0; i < len(emptyRows); i++ {
			emptyRows[i] = strings.Repeat(" ", width)
		}
		rows = append(rows, emptyRows...)
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Bottom,
		lipgloss.JoinVertical(lipgloss.Left, append([]string{header}, rows...)...),
		scrollBar,
	), nil
}

func (table Table) renderCells(data []string, width int, size []int, style lipgloss.Style) string {
	cells := make([]string, len(data))
	for i, cell := range data {
		if len(cell) > size[i]-1 {
			if size[i]-1 >= len(cell) {
				cell = ""
			} else {
				cell = cell[:size[i]-1]
			}
		}
		cells[i] = style.Render(lipgloss.PlaceHorizontal(size[i], lipgloss.Left, cell))
	}

	return lipgloss.PlaceHorizontal(width, lipgloss.Left, lipgloss.JoinHorizontal(lipgloss.Center, cells...))
}
