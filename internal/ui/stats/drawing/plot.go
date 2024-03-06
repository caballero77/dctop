package drawing

import (
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
	maxValue T

	color ColorGradient

	width  int
	height int
}

func New[T constraints.Float](gradient ColorGradient) Plot[T] {
	return Plot[T]{
		data:  queues.New[T](),
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

	k := 100 / T(model.height*4)

	for i := 0; i < len(data) && i/2 < model.width; i += 2 {
		firstSegment, secondSegment := 100*data[i]/model.maxValue, T(0.0)
		if i+1 < len(data) {
			secondSegment = 100 * data[i+1] / model.maxValue
		}

		for i := 0; i < len(plot); i++ {
			var x, y int

			x, firstSegment = convertToBrailleRuneIndex(firstSegment, k)
			y, secondSegment = convertToBrailleRuneIndex(secondSegment, k)

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

// Adds a new value to the Plot
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

// Is a Go function that converts a value to a Braille Rune index.
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
