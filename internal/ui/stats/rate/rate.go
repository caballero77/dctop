package rate

import (
	"fmt"

	"github.com/caballero77/dctop/internal/configuration"
	"github.com/caballero77/dctop/internal/ui/helpers"
	"github.com/caballero77/dctop/internal/ui/messages"
	"github.com/caballero77/dctop/internal/ui/stats/drawing/plotting"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"golang.org/x/exp/constraints"
)

type number interface {
	constraints.Float | constraints.Integer
}

type Model[T number] struct {
	plot plotting.Plot[float64]

	plotStyles  lipgloss.Style
	labelStyle  lipgloss.Style
	legendStyle lipgloss.Style

	name string

	currentRate T
	total       T
	max         T

	width  int
	height int

	ready bool
}

func New[T number](name string, theme configuration.Theme) tea.Model {
	model := Model[T]{
		name:        name,
		plotStyles:  lipgloss.NewStyle().Foreground(theme.GetColor("plot")),
		labelStyle:  lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.plain")),
		legendStyle: lipgloss.NewStyle().Foreground(theme.GetColor("legend.plain")),
		plot:        plotting.New[float64](func(_ float64) float64 { return 1 }),
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

		model.plot.SetSize(msg.Width-2, msg.Height-2)
	case PushMsg[T]:
		model.push(msg.Value)
	}
	return model, nil
}

func (model Model[T]) View() string {
	return model.plotStyles.Render(model.plot.View())
}

func (model *Model[T]) push(value T) {
	model.currentRate = value - model.total
	if model.max < model.currentRate && model.total != 0 {
		model.max = model.currentRate
	}
	model.total = value

	if model.ready {
		model.plot.Push(100 * float64(model.currentRate) / float64(model.max))
	}
	model.ready = true
}
