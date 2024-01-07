package compose

import (
	"dctop/internal/configuration"
	"dctop/internal/docker"
	"dctop/internal/ui/common"
	"dctop/internal/utils/slices"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type top struct {
	box   *common.BoxWithBorders
	table *common.Table

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

func newTop(processesListSize int, theme configuration.Theme) top {
	getColumnSizes := func(width int) []int {
		return []int{7, 7, width - 39, 8, 10, 5}
	}

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.plain"))
	labeShortcutStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.shortcut"))

	return top{
		box:   common.NewBoxWithLabel(theme.Sub("border")),
		table: common.NewTable(getColumnSizes, theme.Sub("table")),

		selected:          0,
		scrollPosition:    0,
		processesListSize: processesListSize,
		processes:         make(map[string][]docker.Process),
		label:             labeShortcutStyle.Render("t") + labelStyle.Render("op"),
	}
}

func (model top) Init() tea.Cmd {
	return nil
}

func (model top) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case common.FocusTabChangedMsg:
		model.focus = msg.Tab == common.Processes

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
	case common.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height
	case common.ContainerSelectedMsg:
		model.containerID = msg.Container.InspectData.ID
	case docker.ContainerMsg:
		model = model.handleContainersUpdates(msg)
	}
	return model, nil
}

func (model top) handleContainersUpdates(msg docker.ContainerMsg) top {
	switch msg := msg.(type) {
	case docker.ContainerUpdateMsg:
		model.processes[msg.Inspect.ID] = msg.Processes
	case docker.ContainerRemoveMsg:
		delete(model.processes, msg.ID)
	}
	return model
}

func (model top) View() string {
	processes, ok := model.processes[model.containerID]
	if !ok || len(model.processes) == 0 {
		return model.box.Render(
			[]string{model.label},
			[]string{},
			lipgloss.Place(model.width-2, model.height, lipgloss.Center, lipgloss.Center, "no data"),
			model.focus,
		)
	}
	headers := []string{
		"Pid",
		"Ppid",
		"Command",
		"Threads",
		"Mem",
		"Cpu%",
	}

	items, err := slices.Map(processes, func(process docker.Process) ([]string, error) {
		return []string{
			process.PID,
			process.PPID,
			process.CMD,
			process.Threads,
			process.RSS,
			process.CPU,
		}, nil
	})
	if err != nil {
		panic(err)
	}

	selected := -1
	if model.focus {
		selected = model.selected
	}

	body, err := model.table.Render(headers, items, model.width, selected, model.scrollPosition, model.height)
	if err != nil {
		panic(err)
	}
	return model.box.Render(
		[]string{model.label},
		[]string{},
		body,
		model.focus,
	)
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
