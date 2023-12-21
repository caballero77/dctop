package stats

import (
	"dctop/internal/utils/queues"
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

func getDataChangeFromQueue(data []int, width int) (changes []float64, maxValue, maxChange, current float64) {
	changes = make([]float64, len(data)-1)
	prev := 0.0
	maxChange = .0
	for i := 0; i < len(data) && i < width*2; i++ {
		if i == 0 {
			prev = float64(data[i])
			continue
		}
		curr := float64(data[i])
		changes[i-1] = prev - curr
		prev = curr
		if changes[i-1] > maxChange {
			maxChange = changes[i-1]
		}
	}
	current = .0
	if len(changes) > 0 {
		current = changes[0]
	}

	max := .0
	if maxChange != 0 {
		for i := 0; i < len(changes); i++ {
			changes[i] = (changes[i] / maxChange) * 100
			if changes[i] > max {
				max = changes[i]
			}
		}
	}

	return changes, max, maxChange, current
}

func pushWithLimit[T any](queue *queues.Queue[T], value T, limit int) error {
	queue.Push(value)
	for limit != 0 && queue.Len() > limit {
		_, err := queue.Pop()
		if err != nil {
			return err
		}
	}
	return nil
}
