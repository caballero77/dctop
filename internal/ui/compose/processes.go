package compose

import (
	"dctop/internal/docker"
	"dctop/internal/ui/common"
	"dctop/internal/utils/slices"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type processesList struct {
	box        *common.BoxWithBorders
	labelStyle lipgloss.Style
	table      *common.Table

	containerID       string
	processes         []docker.Process
	processesListSize int
	selected          int
	scrollPosition    int
	focus             bool

	width  int
	height int
}

func newProcessesList(processesListSize int) processesList {
	border := lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
	}
	borderStyle := lipgloss.Color("#434C5E")
	focusBorderStyle := lipgloss.Color("#8FBCBB")

	getColumnSizes := func(width int) []int {
		return []int{7, 7, width - 39, 8, 10, 5}
	}

	return processesList{
		box:        common.NewBoxWithLabel(border, borderStyle, focusBorderStyle),
		labelStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#D8DEE9")),
		table:      common.NewTable(getColumnSizes),

		selected:          0,
		scrollPosition:    0,
		processesListSize: processesListSize,
		processes:         []docker.Process{},
	}
}

func (model processesList) Init() tea.Cmd {
	return nil
}

func (model processesList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				model.SelectUp()
			}
		case tea.KeyDown:
			if model.focus {
				model.SelectDown()
			}
		}
	case common.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height
	case common.ContainerSelectedMsg:
		model.processes = msg.Container.Processes
		model.containerID = msg.Container.InspectData.ID
	case docker.ContainerUpdateMsg:
		if model.containerID == msg.Inspect.ID {
			model.processes = msg.Processes
		}
	}
	return model, nil
}

func (model processesList) View() string {
	if len(model.processes) == 0 {
		label := model.labelStyle.Render("Processes")
		return model.box.Render(
			[]string{label},
			[]string{},
			lipgloss.Place(model.width, model.height, lipgloss.Center, lipgloss.Center, "loading..."),
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

	items, err := slices.Map(model.processes, func(process docker.Process) ([]string, error) {
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

	label := model.labelStyle.Render("Processes")

	return model.box.Render(
		[]string{label},
		[]string{},
		body,
		model.focus,
	)
}

func (model *processesList) SetProcesses(processes []docker.Process) {
	model.processes = processes
	model.selected = 0
	model.scrollPosition = 0
}

func (model *processesList) SelectUp() {
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

func (model *processesList) SelectDown() {
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

func (model *processesList) SetFocus(focus bool) {
	model.focus = focus
}
