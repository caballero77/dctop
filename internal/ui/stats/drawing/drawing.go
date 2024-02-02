package drawing

import (
	"math"
	"slices"

	"github.com/charmbracelet/lipgloss"
)

var braille = [][]string{
	{" ", "⢀", "⢠", "⢰", "⢸"},
	{"⡀", "⣀", "⣠", "⣰", "⣸"},
	{"⡄", "⣄", "⣤", "⣴", "⣼"},
	{"⡆", "⣆", "⣦", "⣶", "⣾"},
	{"⡇", "⣇", "⣧", "⣷", "⣿"},
}

func RenderPlot(data []float64, scale float64, width, height int) string {
	lines := make([]string, height)
	k := scale * 100 / float64(height*4)
	for i := 0; i < len(data) && i/2 < width; i += 2 {
		cpuX, cpuY := data[i], 0.0
		if i+1 < len(data) {
			cpuY = data[i+1]
		}

		for i := 0; i < len(lines); i++ {
			var x, y int

			x, cpuX = convertToBrailleRuneIndex(cpuX, k)
			y, cpuY = convertToBrailleRuneIndex(cpuY, k)

			lines[i] += braille[x][y]
		}
	}

	slices.Reverse(lines)

	return lipgloss.PlaceHorizontal(width, lipgloss.Left, lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func convertToBrailleRuneIndex(value, scale float64) (index int, adjustedValue float64) {
	if value >= 4*scale {
		return 4, value - 4*scale
	}

	index = int(math.Floor(value / scale))
	if value >= 0.0001 && index == 0 {
		index = 1
	}

	return min(max(index, 0), 4), 0
}
