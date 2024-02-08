package plotting

import (
	"dctop/internal/ui/messages"
	"dctop/internal/utils/queues"
	"fmt"
	"log/slog"
	"math"
	"slices"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/exp/constraints"
)

var braille = [5][5]string{
	{" ", "⢀", "⢠", "⢰", "⢸"},
	{"⡀", "⣀", "⣠", "⣰", "⣸"},
	{"⡄", "⣄", "⣤", "⣴", "⣼"},
	{"⡆", "⣆", "⣦", "⣶", "⣾"},
	{"⡇", "⣇", "⣧", "⣷", "⣿"},
}

type Plot[T constraints.Float] struct {
	data     *queues.Queue[T]
	scale    func(T) T
	maxValue T

	width  int
	height int
}

func New[T constraints.Float](scale func(T) T) Plot[T] {
	return Plot[T]{
		data:  queues.New[T](),
		scale: scale,
	}
}

func (Plot[T]) Init() tea.Cmd { return nil }

func (model Plot[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.SizeChangeMsq:
		model.SetSize(msg.Width, msg.Height)
	case PushMsg[T]:
		model.Push(msg.Value)
	}

	return model, nil
}

func (model Plot[T]) View() string {
	if model.data.Len() < 1 || model.width == 0 || model.height == 0 {
		return lipgloss.Place(model.width, model.height, lipgloss.Center, lipgloss.Center, "no data")
	}

	data := model.data.ToArray()
	plot := make([]string, model.height)

	scale := model.scale(model.maxValue)
	k := scale * 100 / T(model.height*4)

	for i := 0; i < len(data) && i/2 < model.width; i += 2 {
		cpuX, cpuY := 100*data[i]/model.maxValue, T(0.0)
		if i+1 < len(data) {
			cpuY = 100 * data[i+1] / model.maxValue
		}

		for i := 0; i < len(plot); i++ {
			var x, y int

			x, cpuX = convertToBrailleRuneIndex(cpuX, k)
			y, cpuY = convertToBrailleRuneIndex(cpuY, k)

			plot[i] += braille[x][y]
		}
	}

	slices.Reverse(plot)

	return lipgloss.PlaceHorizontal(model.width, lipgloss.Left, lipgloss.JoinVertical(lipgloss.Left, plot...))
}

func (model *Plot[T]) SetSize(width, height int) {
	model.width = width
	model.height = height
}

func (model *Plot[T]) Push(value T) {
	err := model.data.PushWithLimit(value, model.width*2)
	if err != nil {
		slog.Error("Error adding new value to plot",
			"limit", model.width*2,
			"error", err)
	}

	var maxValue T
	for _, item := range model.data.ToArray() {
		maxValue = max(maxValue, item)
	}
	model.maxValue = maxValue
}

func convertToBrailleRuneIndex[T constraints.Float](value, scale T) (index int, adjustedValue T) {
	if value >= 4*scale {
		return 4, value - 4*scale
	}

	index = int(math.Floor(float64(value / scale)))
	if value >= 0.0001 && index == 0 {
		index = 1
	}

	return min(max(index, 0), 4), 0
}

func drawPlotWithGradient(lines []string, start, end lipgloss.Color) []string {
	gradient := generateColorGradient(start, end, len(lines))

	for i, line := range lines {
		lines[i] = lipgloss.NewStyle().Foreground(gradient[i]).Render(line)
	}

	return lines
}

func generateColorGradient(start, end lipgloss.Color, numSteps int) []lipgloss.Color {
	startRGB := hexToRGB(start)
	endRGB := hexToRGB(end)

	stepSize := [3]float64{
		float64(endRGB[0]-startRGB[0]) / float64(numSteps-1),
		float64(endRGB[1]-startRGB[1]) / float64(numSteps-1),
		float64(endRGB[2]-startRGB[2]) / float64(numSteps-1),
	}

	var gradient []lipgloss.Color
	for step := 0; step < numSteps; step++ {
		rgb := [3]uint32{
			startRGB[0] + uint32(math.Round(float64(step)*stepSize[0])),
			startRGB[1] + uint32(math.Round(float64(step)*stepSize[1])),
			startRGB[2] + uint32(math.Round(float64(step)*stepSize[2])),
		}

		colorHex := fmt.Sprintf("#%02x%02x%02x", rgb[0], rgb[1], rgb[2])
		gradient = append(gradient, lipgloss.Color(colorHex))
	}

	return gradient
}

func hexToRGB(hexColor lipgloss.Color) [3]uint32 {
	r, g, b, _ := hexColor.RGBA()
	return [3]uint32{r, g, b}
}
