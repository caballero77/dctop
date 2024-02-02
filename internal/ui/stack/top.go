package stack

import (
	"dctop/internal/configuration"
	"dctop/internal/docker"
	"dctop/internal/ui/helpers"
	"dctop/internal/ui/messages"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type top struct {
	table helpers.Table

	containerID       string
	processes         map[string][]docker.Process
	processesListSize int
	selected          int
	scrollPosition    int
	focus             bool

	width  int
	height int

	label string
}

func newTop(processesListSize int, theme configuration.Theme) tea.Model {
	getColumnSizes := func(width int) []int {
		return []int{7, 7, width - 39, 8, 10, 5}
	}

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.plain"))
	labeShortcutStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.shortcut"))

	model := top{
		table: helpers.NewTable(getColumnSizes, theme.Sub("table")),

		processesListSize: processesListSize,
		processes:         make(map[string][]docker.Process),
		label:             labeShortcutStyle.Render("t") + labelStyle.Render("op"),
	}

	return helpers.NewBox(model, theme.Sub("border"))
}

func (model top) Focus() bool { return model.focus }

func (model top) Labels() []string { return []string{model.label} }

func (model top) Legends() []string { return []string{} }

func (model top) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return model.UpdateAsBoxed(msg) }

func (model top) Init() tea.Cmd {
	return nil
}

func (model top) UpdateAsBoxed(msg tea.Msg) (helpers.BoxedModel, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.FocusTabChangedMsg:
		model.focus = msg.Tab == messages.Processes

		if model.focus {
			model.selected = 0
		}

		return model, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			if model.focus {
				model.selectUp()
			}
		case tea.KeyDown:
			if model.focus {
				model.selectDown()
			}
		}
	case messages.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height
	case messages.ContainerSelectedMsg:
		model.containerID = msg.Container.InspectData.ID
	case docker.ContainerMsg:
		model.handleContainersUpdates(msg)
	}
	return model, nil
}

func (model *top) handleContainersUpdates(msg docker.ContainerMsg) {
	switch msg := msg.(type) {
	case docker.ContainerUpdateMsg:
		model.processes[msg.Inspect.ID] = msg.Processes
	case docker.ContainerRemoveMsg:
		delete(model.processes, msg.ID)
	}
}

func (model top) View() string {
	processes, ok := model.processes[model.containerID]
	if !ok || len(model.processes) == 0 {
		return lipgloss.Place(model.width-2, model.height, lipgloss.Center, lipgloss.Center, "no data")
	}
	headers := []string{
		"Pid",
		"Ppid",
		"Command",
		"Threads",
		"Mem",
		"Cpu%",
	}

	items := make([][]string, len(processes))
	for i, process := range processes {
		items[i] = []string{
			process.PID,
			process.PPID,
			process.CMD,
			process.Threads,
			process.RSS,
			process.CPU,
		}
	}

	selected := -1
	if model.focus {
		selected = model.selected
	}

	return model.table.Render(headers, items, model.width, selected, model.scrollPosition, model.height)
}

func (model *top) selectUp() {
	if model.selected == 0 {
		model.selected = len(model.processes) - 1
		if len(model.processes) > model.processesListSize && model.processesListSize > 0 {
			model.scrollPosition = model.selected - (model.processesListSize - 1)
		}
	} else {
		model.selected--
		if len(model.processes) > model.processesListSize && model.processesListSize > 0 && model.selected < model.scrollPosition {
			model.scrollPosition = model.selected
		}
	}
}

func (model *top) selectDown() {
	if model.selected == len(model.processes)-1 {
		model.selected = 0
		if len(model.processes) > model.processesListSize && model.processesListSize > 0 {
			model.scrollPosition = 0
		}
	} else {
		model.selected++
		if len(model.processes) > model.processesListSize && model.processesListSize > 0 && model.selected-model.processesListSize >= model.scrollPosition {
			model.scrollPosition = model.selected - (model.processesListSize - 1)
		}
	}
}
