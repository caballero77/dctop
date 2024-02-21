package plotting

import (
	"fmt"
	"image/color"
	"log/slog"
	"slices"

	"github.com/caballero77/dctop/internal/ui/messages"
	"github.com/caballero77/dctop/internal/utils/queues"

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

type ColorGradient struct {
	From lipgloss.Color
	To   lipgloss.Color
}

type Plot[T constraints.Float] struct {
	data     *queues.Queue[T]
	scale    func(T) T
	maxValue T

	color ColorGradient

	width  int
	height int
}

func New[T constraints.Float](scale func(T) T, gradient ColorGradient) Plot[T] {
	return Plot[T]{
		data:  queues.New[T](),
		scale: scale,
		color: gradient,
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

	gradient := generateColorGradient(model.color.From, model.color.To, len(plot))

	for i, line := range plot {
		plot[i] = lipgloss.NewStyle().Foreground(gradient[i]).Render(line)
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

	index = int(value / scale)
	if value >= 0.0001 && index == 0 {
		index = 1
	}

	if index < 0 {
		index = 0
	} else if index > 4 {
		index = 4
	}

	return index, 0
}

func generateColorGradient(start, end lipgloss.Color, numSteps int) []lipgloss.Color {
	color1 := lipglossToRGBA(start)
	color2 := lipglossToRGBA(end)

	gradient := make([]lipgloss.Color, numSteps)
	rStep := float64(color2.R-color1.R) / float64(numSteps-1)
	gStep := float64(color2.G-color1.G) / float64(numSteps-1)
	bStep := float64(color2.B-color1.B) / float64(numSteps-1)
	aStep := float64(color2.A-color1.A) / float64(numSteps-1)

	for i := 0; i < numSteps; i++ {
		r := uint8(float64(color1.R) + float64(i)*rStep)
		g := uint8(float64(color1.G) + float64(i)*gStep)
		b := uint8(float64(color1.B) + float64(i)*bStep)
		a := uint8(float64(color1.A) + float64(i)*aStep)
		gradient[i] = lipgloss.Color(fmt.Sprintf("#%02x%02x%02x%02x", r, g, b, a))
	}

	return gradient
}

func lipglossToRGBA(lipglossColor lipgloss.Color) color.RGBA {
	r, g, b, a := lipglossColor.RGBA()

	return color.RGBA{
		R: uint8(r),
		G: uint8(g),
		B: uint8(b),
		A: uint8(a),
	}
}
