package stats

import (
	"math"
	"slices"

	"github.com/charmbracelet/lipgloss"
)

var braile = [][]string{
	{" ", "⢀", "⢠", "⢰", "⢸"},
	{"⡀", "⣀", "⣠", "⣰", "⣸"},
	{"⡄", "⣄", "⣤", "⣴", "⣼"},
	{"⡆", "⣆", "⣦", "⣶", "⣾"},
	{"⡇", "⣇", "⣧", "⣷", "⣿"},
}

func renderPlot(data []float64, scale float64, width, height int) string {
	lines := make([]string, height)
	k := scale * 100 / float64(height*4)
	for i := 0; i < len(data) && i/2 < width; i += 2 {
		cpuX, cpuY := data[i], 0.0
		if i+1 < len(data) {
			cpuY = data[i+1]
		}

		for i := 0; i < len(lines); i++ {
			var x, y int

			x, cpuX = convertToBraileRuneIndex(cpuX, k)
			y, cpuY = convertToBraileRuneIndex(cpuY, k)

			lines[i] += braile[x][y]
		}
	}

	slices.Reverse(lines)

	return lipgloss.PlaceHorizontal(width, lipgloss.Left, lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func convertToBraileRuneIndex(value, scale float64) (index int, asjustedValue float64) {
	if value >= 4*scale {
		return 4, value - 4*scale
	}

	index = int(math.Floor(value / scale))
	if value >= 0.0001 && index == 0 {
		index = 1
	}
	return index, 0
}

func getRate(data []uint64) (rates []float64, max, current uint64) {
	changes := make([]uint64, len(data)-1)
	var prev uint64
	for i := 0; i < len(data); i++ {
		if i == 0 {
			prev = data[i]
			continue
		}
		value := data[i]
		curr := value
		changes[i-1] = prev - curr
		prev = curr
		if changes[i-1] > max {
			max = changes[i-1]
		}
	}
	if len(changes) > 0 {
		current = changes[0]
	}

	rates = make([]float64, len(changes))
	if max == 0 {
		return rates, max, current
	}

	for i := 0; i < len(changes); i++ {
		rates[i] = (float64(changes[i]) / float64(max)) * 100
	}

	return rates, max, current
}
