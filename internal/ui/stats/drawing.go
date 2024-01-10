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
	k := 100 / float64(height*4)
	k *= scale
	w := 0
	for i := 0; i < len(data) && w < width; i++ {
		cpuX := data[i]
		cpuY := 0.0
		i++
		if i < len(data) {
			cpuY = data[i]
		}
		for i := 0; i < len(lines); i++ {
			var x, y int
			if cpuX >= 4*k {
				x = 4
				cpuX -= 4 * k
			} else {
				x = int(math.Floor(cpuX / k))
				if cpuX >= 0.0001 && x == 0 && i == 0 {
					x = 1
				}
				cpuX = 0
			}

			if cpuY >= 4*k {
				y = 4
				cpuY -= 4 * k
			} else {
				y = int(math.Floor(cpuY / k))
				if cpuY >= 0.0001 && y == 0 && i == 0 {
					y = 1
				}
				cpuY = 0
			}

			b := braile[x][y]
			lines[i] += b
		}
		w++
	}
	for w < width {
		for i := 0; i < len(lines); i++ {
			lines[i] += braile[0][0]
		}
		w++
	}

	slices.Reverse(lines)

	return lipgloss.JoinVertical(lipgloss.Center, lines...)
}

func getRate(data []uint64) (rates []float64, max, current uint64) {
	changes := make([]uint64, len(data)-1)
	var prev uint64
	for i := 0; i < len(data); i++ {
		value := data[i]
		if i == 0 {
			prev = value
			continue
		}
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
	if max != 0 {
		for i := 0; i < len(changes); i++ {
			rates[i] = (float64(changes[i]) / float64(max)) * 100
		}
	}

	return rates, max, current
}
