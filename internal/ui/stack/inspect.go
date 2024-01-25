package stack

import (
	"bytes"
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
	box  helpers.BoxWithBorders
	text tea.Model

	inspects          map[string]types.ContainerJSON
	selectedContainer string
	focus             bool

	label string

	width  int
	height int
}

func newInspect(theme configuration.Theme) inspect {
	label := lipgloss.NewStyle().Foreground(theme.GetColor("title.shortcut")).Render("I") +
		lipgloss.NewStyle().Foreground(theme.GetColor("title.plain")).Render("nspect")
	return inspect{
		box:      helpers.NewBox(theme.Sub("border")),
		text:     helpers.NewTextBox("", lipgloss.NewStyle().Foreground(theme.GetColor("body.text"))),
		inspects: make(map[string]types.ContainerJSON),
		label:    label,
	}
}

func (inspect) Init() tea.Cmd {
	return nil
}

func (model inspect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case messages.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height

		model.text, cmd = model.text.Update(messages.SizeChangeMsq{Width: msg.Width, Height: msg.Height - 2})
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case tea.KeyMsg:
		if model.focus {
			switch msg.Type {
			case tea.KeyUp:
				model.text, cmd = model.text.Update(messages.ScrollMsg{Change: -1})
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			case tea.KeyDown:
				model.text, cmd = model.text.Update(messages.ScrollMsg{Change: 1})
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}
	case docker.ContainerMsg:
		model, cmd = model.handleContainersUpdates(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case messages.ContainerSelectedMsg:
		if model.selectedContainer != msg.Container.InspectData.ID {
			model.selectedContainer = msg.Container.InspectData.ID

			model.text, cmd = model.text.Update(messages.SetTextMgs{Text: model.view(), ResetScroll: false})
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case messages.FocusTabChangedMsg:
		model.focus = msg.Tab == messages.Inspect
	}
	return model, tea.Batch(cmds...)
}

func (model inspect) handleContainersUpdates(msg docker.ContainerMsg) (inspect, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case docker.ContainerUpdateMsg:
		model.inspects[msg.ID] = msg.Inspect
		if model.selectedContainer == msg.ID {
			model.text, cmd = model.text.Update(messages.SetTextMgs{Text: model.view()})
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case docker.ContainerRemoveMsg:
		delete(model.inspects, msg.ID)
		if model.selectedContainer == msg.ID {
			model.text, cmd = model.text.Update(messages.SetTextMgs{Text: model.view()})
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	return model, tea.Batch(cmds...)
}

func (model inspect) View() string {
	return model.box.Render(
		[]string{model.label},
		[]string{},
		model.text.View(),
		model.focus,
	)
}

func (model inspect) view() string {
	data, ok := model.inspects[model.selectedContainer]
	if !ok {
		return ""
	}

	var buffer bytes.Buffer

	divider := strings.Repeat("_", model.width-3)

	buffer.WriteString(lipgloss.PlaceHorizontal(model.width-3, lipgloss.Center, "General Info") + "\n")
	buffer.WriteString(fmt.Sprintf("Name: %s", data.Name) + "\n")
	buffer.WriteString(fmt.Sprintf("Id: %s", data.ID) + "\n")
	buffer.WriteString(fmt.Sprintf("Status: %s", data.State.Status) + "\n")
	buffer.WriteString(fmt.Sprintf("Image: %s %s", data.Config.Image, data.Image) + "\n")
	buffer.WriteString(fmt.Sprintf("Created: %s", data.Created) + "\n")
	buffer.WriteString(fmt.Sprintf("Started: %s", data.State.StartedAt) + "\n")
	buffer.WriteString(fmt.Sprintf("CMD: %s", strings.Join(data.Config.Cmd, " ")) + "\n")
	buffer.WriteString(fmt.Sprintf("Entrypoint: %s", strings.Join(data.Config.Entrypoint, " ")) + "\n")
	buffer.WriteString(divider + "\n")

	buffer.WriteString(lipgloss.PlaceHorizontal(model.width-3, lipgloss.Center, "Ports") + "\n")

	ports := make([]string, 0)
	for port, bindings := range data.HostConfig.PortBindings {
		for _, binding := range bindings {
			ports = append(ports, fmt.Sprintf("%s:%s 🠖 %s", binding.HostIP, binding.HostPort, port.Port()))
		}
	}
	slices.Sort(ports)
	for _, port := range ports {
		buffer.WriteString(port + "\n")
	}

	buffer.WriteString(divider + "\n")
	buffer.WriteString(lipgloss.PlaceHorizontal(model.width-3, lipgloss.Center, "Environment variables") + "\n")

	for _, env := range data.Config.Env {
		buffer.WriteString(env + "\n")
	}

	buffer.WriteString(divider + "\n")

	buffer.WriteString(lipgloss.PlaceHorizontal(model.width-3, lipgloss.Center, "Labels") + "\n")
	labels := make([]string, len(data.Config.Labels))
	i := 0
	for label, value := range data.Config.Labels {
		labels[i] = fmt.Sprintf("%s=%s", label, value)
		i++
	}
	slices.Sort(labels)
	for _, label := range labels {
		buffer.WriteString(label + "\n")
	}

	return buffer.String()
}
