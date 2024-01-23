package stack

import (
	"dctop/internal/configuration"
	"dctop/internal/docker"
	"dctop/internal/ui/helpers"
	"dctop/internal/ui/messages"
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types"
)

type inspect struct {
	box helpers.BoxWithBorders

	inspects          map[string]types.ContainerJSON
	selectedContainer string
	focus             bool

	label string

	weight int
	height int
}

func newInspect(theme configuration.Theme) inspect {
	label := lipgloss.NewStyle().Foreground(theme.GetColor("title.shortcut")).Render("I") +
		lipgloss.NewStyle().Foreground(theme.GetColor("title.plain")).Render("nspect")
	return inspect{
		box:      helpers.NewBox(theme.Sub("border")),
		inspects: make(map[string]types.ContainerJSON),
		label:    label,
	}
}

func (inspect) Init() tea.Cmd {
	return nil
}

func (model inspect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.SizeChangeMsq:
		model.weight = msg.Width
		model.height = msg.Height
	case docker.ContainerMsg:
		model = model.handleContainersUpdates(msg)
	case messages.ContainerSelectedMsg:
		model.selectedContainer = msg.Container.InspectData.ID
	case messages.FocusTabChangedMsg:
		model.focus = msg.Tab == messages.Inspect
	}
	return model, nil
}

func (model inspect) handleContainersUpdates(msg docker.ContainerMsg) inspect {
	switch msg := msg.(type) {
	case docker.ContainerUpdateMsg:
		model.inspects[msg.ID] = msg.Inspect
	case docker.ContainerRemoveMsg:
		delete(model.inspects, msg.ID)
	}
	return model
}

func (model inspect) View() string {
	data, ok := model.inspects[model.selectedContainer]
	if !ok {
		return ""
	}

	lines := make([]string, 0, 20)

	style := lipgloss.NewStyle()

	lines = append(lines,
		style.Render(fmt.Sprintf("Name: %s", data.Name)),
		style.Render(fmt.Sprintf("Id: %s", data.ID)),
		style.Render(fmt.Sprintf("Status: %s", data.State.Status)),
		style.Render(fmt.Sprintf("Image: %s", data.Image)),
		style.Render(fmt.Sprintf("Created: %s", data.Created)),
		style.Render(fmt.Sprintf("Started: %s", data.State.StartedAt)),
		style.Render(fmt.Sprintf("CMD: %s", strings.Join(data.Config.Cmd, " "))),
		style.Render(fmt.Sprintf("Entrypoint: %s", strings.Join(data.Config.Entrypoint, " "))),

		style.Render("Ports:"),
	)

	for label, value := range data.Config.ExposedPorts {
		lines = append(lines, style.Render(fmt.Sprintf("%s > %s", label, value)))
	}

	lines = append(lines, style.Render("Environment variables:"))
	for _, env := range data.Config.Env {
		lines = append(lines, style.Render(env))
	}

	lines = append(lines, style.Render("Labels"))
	labels := make([]string, len(data.Config.Labels))
	i := 0
	for label, value := range data.Config.Labels {
		labels[i] = style.Render(fmt.Sprintf("%s > %s", label, value))
		i++
	}
	slices.Sort(labels)
	lines = append(lines, labels...)

	body := lipgloss.JoinVertical(lipgloss.Left, lines...)

	return model.box.Render(
		[]string{model.label},
		[]string{},
		lipgloss.Place(model.weight-2, model.height-2, lipgloss.Left, lipgloss.Top, body),
		model.focus,
	)
}
