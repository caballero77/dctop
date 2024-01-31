package rate

import (
	"dctop/internal/configuration"
	"dctop/internal/ui/helpers"
	"dctop/internal/ui/messages"
	"dctop/internal/ui/stats/drawing"
	"dctop/internal/utils/queues"
	"fmt"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"golang.org/x/exp/constraints"
)

type number interface {
	constraints.Float | constraints.Integer
}

type Model[T number] struct {
	data *queues.Queue[T]

	plotStyles  lipgloss.Style
	labelStyle  lipgloss.Style
	legendStyle lipgloss.Style

	name string

	currentRate T
	total       T
	max         T

	width  int
	height int
}

func New[T number](name string, theme configuration.Theme) tea.Model {
	model := Model[T]{
		data:        queues.New[T](),
		name:        name,
		plotStyles:  lipgloss.NewStyle().Foreground(theme.GetColor("plot")),
		labelStyle:  lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.plain")),
		legendStyle: lipgloss.NewStyle().Foreground(theme.GetColor("legend.plain")),
	}

	return helpers.NewBox(model, theme.Sub("border"))
}

func (Model[T]) Focus() bool { return false }

func (model Model[T]) Labels() []string {
	return []string{
		model.labelStyle.Render(fmt.Sprintf("%s: %s/sec", model.name, humanize.IBytes(uint64(model.currentRate)))),
	}
}

func (model Model[T]) Legends() []string {
	return []string{
		model.legendStyle.Render(fmt.Sprintf("total: %s", humanize.IBytes(uint64(model.total)))),
		model.legendStyle.Render(fmt.Sprintf("max: %s/sec", humanize.IBytes(uint64(model.max)))),
	}
}

func (model Model[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return model.UpdateAsBoxed(msg) }

func (Model[T]) Init() tea.Cmd { return nil }

func (model Model[T]) UpdateAsBoxed(msg tea.Msg) (helpers.BoxedModel, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.SizeChangeMsq:
		model.width = msg.Width - 2
		model.height = msg.Height - 2
	case PushMsg[T]:
		model.push(msg.Value)
	}
	return model, nil
}

func (model Model[T]) View() string {
	if model.data.Len() <= 1 {
		return lipgloss.Place(model.width, model.height, lipgloss.Center, lipgloss.Center, "no data")
	}

	data := getRate(model.data.ToArray())

	plot := drawing.RenderPlot(data, 1, model.width, model.height)

	return model.plotStyles.Render(plot)
}

func (model *Model[T]) push(value T) {
	model.currentRate = value - model.total
	if model.max < model.currentRate && model.total != 0 {
		model.max = model.currentRate
	}
	model.total = value

	err := model.data.PushWithLimit(value, model.width*2)
	if err != nil {
		slog.Error("Error pushing value in queue with limit",
			"component", model.name,
			"limit", model.width*2,
			"error", err)
	}
}

func getRate[T number](data []T) []float64 {
	var max T
	changes := make([]T, len(data)-1)
	var prev T
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

	rates := make([]float64, len(changes))
	if max == 0 {
		return rates
	}

	for i := 0; i < len(changes); i++ {
		rates[i] = (float64(changes[i]) / float64(max)) * 100
	}

	return rates
}
