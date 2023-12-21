package common

import (
	"dctop/internal/utils/slices"
	utils_strings "dctop/internal/utils/strings"
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

func NewTable(getSizes func(int) []int) *Table {
	bodyCellStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#D8DEE9"))
	return &Table{
		headerCellStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("#D8DEE9")),
		bodyCellStyle:    bodyCellStyle,
		selectedRowStyle: bodyCellStyle.Copy().Background(lipgloss.Color("#434C5E")),
		scrollStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("#D8DEE9")),
		getSizes:         getSizes,
	}
}

func (table *Table) Render(headerCells []string, rowCells [][]string, width, selected, scrollPosition, height int) (string, error) {
	scrollBar := strings.Repeat(" \n", len(rowCells))
	width -= 3
	height--

	if height > 0 && len(rowCells) > height {
		pos := int(float64(scrollPosition) * float64(height) / float64(len(rowCells)-height))
		newScrollBar, err := utils_strings.ReplaceAtIndex(strings.Repeat("\n", height), table.scrollStyle.Render("█"), pos)
		if err != nil {
			return "", err
		}
		scrollBar = newScrollBar
		rowCells = rowCells[scrollPosition : scrollPosition+height]
	}

	size := table.getSizes(width)
	if len(headerCells) != len(size) {
		return "", errors.New("unexpected header length")
	}

	header, err := table.renderCells(headerCells, size, width, createCellRenderer(table.headerCellStyle))
	if err != nil {
		return "", err
	}

	bodyCellRenderer := createCellRenderer(table.bodyCellStyle)
	selectedRowRenderer := createCellRenderer(table.selectedRowStyle)

	rows, err := slices.MapI(rowCells, func(i int, row []string) (string, error) {
		if i+scrollPosition == selected {
			return table.renderCells(row, size, width, selectedRowRenderer)
		}
		return table.renderCells(row, size, width, bodyCellRenderer)
	})
	if err != nil {
		return "", err
	}

	if len(rows) < height {
		rows = append(rows, slices.Repeat(strings.Repeat(" ", width), height-len(rows))...)
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Bottom,
		lipgloss.JoinVertical(lipgloss.Left, append([]string{header}, rows...)...),
		scrollBar,
	), nil
}

func (table *Table) renderCells(data []string, size []int, width int, render func(string) string) (string, error) {
	columns, err := slices.MapI(data, func(i int, column string) (string, error) {
		if len(column) > size[i]-1 {
			column = column[:size[i]-1]
		}
		return render(lipgloss.PlaceHorizontal(size[i], lipgloss.Left, column)), nil
	})
	if err != nil {
		return "", err
	}

	return lipgloss.PlaceHorizontal(width, lipgloss.Left, lipgloss.JoinHorizontal(lipgloss.Center, columns...)), nil
}

func createCellRenderer(style lipgloss.Style) func(string) string {
	return func(cell string) string {
		return style.Render(cell)
	}
}
