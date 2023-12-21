package compose

import (
	"dctop/internal/docker"
	"dctop/internal/ui/common"
	"dctop/internal/utils/maps"
	"dctop/internal/utils/slices"
	"errors"
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type containersList struct {
	box         *common.BoxWithBorders
	table       *common.Table
	labelStyle  lipgloss.Style
	legendStyle lipgloss.Style

	selected           int
	scrollPosition     int
	containersListSize int
	containers         []*docker.ContainerInfo
	containersMap      map[string]*docker.ContainerInfo
	cpuUsages          map[string]float64
	focus              bool
	service            *docker.ComposeService

	width  int
	height int
}

func newContainersList(size int, service *docker.ComposeService, focus bool) containersList {
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
		return []int{15, width - 46, 10, 15, 6}
	}

	return containersList{
		box:         common.NewBoxWithLabel(border, borderStyle, focusBorderStyle),
		labelStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("#D8DEE9")),
		table:       common.NewTable(getColumnSizes),
		legendStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#8FBCBB")),

		containersListSize: size,
		selected:           0,
		scrollPosition:     0,
		containers:         []*docker.ContainerInfo{},
		focus:              focus,
		service:            service,
		containersMap:      make(map[string]*docker.ContainerInfo),
		cpuUsages:          make(map[string]float64),
	}
}

func (model containersList) Init() tea.Cmd {
	updates, err := model.service.GetContainerUpdates()
	if err != nil {
		panic(err)
	}
	return func() tea.Msg {
		return <-updates
	}
}

func (model containersList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyRunes:
			if !model.focus {
				return model, nil
			}
			switch string(msg.Runes) {
			case "1":
				return model, func() tea.Msg {
					selectedContainer := model.containers[model.selected]
					err := model.service.ContainerStop(selectedContainer.InspectData.ID)
					if err != nil {
						panic(err)
					}
					return nil
				}
			case "2":
				return model, func() tea.Msg {
					selectedContainer := model.containers[model.selected]
					err := model.service.ContainerStart(selectedContainer.InspectData.ID)
					if err != nil {
						panic(err)
					}
					return nil
				}
			case "3":
				return model, func() tea.Msg {
					selectedContainer := model.containers[model.selected]
					err := model.service.ContainerPause(selectedContainer.InspectData.ID)
					if err != nil {
						panic(err)
					}
					return nil
				}
			case "4":
				return model, func() tea.Msg {
					selectedContainer := model.containers[model.selected]
					err := model.service.ContainerUnpause(selectedContainer.InspectData.ID)
					if err != nil {
						panic(err)
					}
					return nil
				}
			case "l":
				if len(model.containers) != 0 {
					return model, tea.Batch(
						func() tea.Msg {
							return common.StartListenningLogsMsg{ContainerID: model.containers[model.selected].InspectData.ID}
						},
						func() tea.Msg { return common.FocusTabChangedMsg{Tab: common.Logs} },
					)
				}
			}
		case tea.KeyUp:
			if model.focus {
				model.SelectUp()
				return model, model.GetContainerSelectedCmd()
			}
			return model, nil
		case tea.KeyDown:
			if model.focus {
				model.SelectDown()
				return model, model.GetContainerSelectedCmd()
			}
			return model, nil
		}
	case common.FocusTabChangedMsg:
		model.focus = msg.Tab == common.Containers

		return model, nil
	case common.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height
	case docker.ContainerUpdateMsg:
		container, ok := model.containersMap[msg.ID]
		updates, err := model.service.GetContainerUpdates()
		if err != nil {
			panic(err)
		}
		if ok {
			model.cpuUsages[msg.ID] = model.calculateCPUUsage(container.StatsSnapshot, msg.Stats)
			container.InspectData = msg.Inspect
			container.Processes = msg.Processes
			container.StatsSnapshot = msg.Stats
			return model, waitForActivity(updates)

		} else {
			container = &docker.ContainerInfo{
				InspectData:   msg.Inspect,
				StatsSnapshot: msg.Stats,
				Processes:     make([]docker.Process, 0),
			}
			model.containersMap[msg.ID] = container
			model.containers = append(model.containers, container)
			containers := model.containers
			sort.Slice(containers, func(i, j int) bool { return containers[i].InspectData.Name < containers[j].InspectData.Name })

			return model, tea.Batch(model.GetContainerSelectedCmd(), waitForActivity(updates))
		}
	}

	return model, nil
}

func (model containersList) View() string {
	if len(model.containers) == 0 {
		return ""
	}
	headers := []string{
		"Name",
		"Image",
		"Status",
		"Ip Address",
		"Cpu%",
	}

	items, err := slices.Map(model.containers, func(container *docker.ContainerInfo) ([]string, error) {
		cpu, ok := model.cpuUsages[container.InspectData.ID]
		if !ok {
			cpu = .0
		}

		networkNames := maps.Keys(container.InspectData.NetworkSettings.Networks)

		stack := model.service.Stack()
		return []string{
			beautifyContainerName(container.InspectData.Name, stack),
			container.InspectData.Config.Image,
			container.InspectData.State.Status,
			container.InspectData.NetworkSettings.Networks[networkNames[0]].IPAddress,
			fmt.Sprintf("%.2f", cpu),
		}, nil
	})
	if err != nil {
		panic(err)
	}

	body, err := model.table.Render(headers, items, model.width, model.selected, model.scrollPosition, model.height)
	if err != nil {
		panic(err)
	}

	label := model.labelStyle.Render("Containers")

	legend := ""
	if model.focus {
		legend = model.legendStyle.Render("¹stop ²start ³pause ⁴unpause")
	}

	return model.box.Render(
		[]string{label},
		[]string{legend},
		body,
		model.focus,
	)
}

func (model *containersList) AddContainer(container *docker.ContainerInfo) {
	model.containers = append(model.containers, container)
	containers := model.containers
	sort.Slice(containers, func(i, j int) bool { return containers[i].InspectData.Name < containers[j].InspectData.Name })
}

func (model *containersList) Empty() bool {
	return len(model.containers) == 0
}

func (model *containersList) SetFocus(focus bool) {
	model.focus = focus
}

func (model *containersList) SelectUp() {
	if model.selected == 0 {
		model.selected = len(model.containers) - 1
		if len(model.containers) > model.containersListSize && model.containersListSize > 0 {
			model.scrollPosition = model.selected - (model.containersListSize - 1)
		}
	} else {
		model.selected--
		if len(model.containers) > model.containersListSize && model.containersListSize > 0 && model.selected < model.scrollPosition {
			model.scrollPosition = model.selected
		}
	}
}

func (model *containersList) SelectDown() {
	if model.selected == len(model.containers)-1 {
		model.selected = 0
		if len(model.containers) > model.containersListSize && model.containersListSize > 0 {
			model.scrollPosition = 0
		}
	} else {
		model.selected++
		if len(model.containers) > model.containersListSize && model.containersListSize > 0 && model.selected-model.containersListSize >= model.scrollPosition {
			model.scrollPosition = model.selected - (model.containersListSize - 1)
		}
	}
}

func (model containersList) GetContainerSelectedCmd() tea.Cmd {
	return func() tea.Msg { return common.ContainerSelectedMsg{Container: *model.containers[model.selected]} }
}

func (model *containersList) GetSelectedContainer() (*docker.ContainerInfo, error) {
	if len(model.containers) == 0 {
		return nil, errors.New("containers list is empty")
	}
	return model.containers[model.selected], nil
}

func beautifyContainerName(name, stack string) string {
	if strings.HasPrefix(name, "/") {
		name = strings.TrimLeft(name, "/")
	}

	stackPrefix := fmt.Sprintf("%s-", stack)
	if strings.HasPrefix(name, stackPrefix) {
		name = strings.TrimLeft(name, stackPrefix)
	}
	return name
}

func waitForActivity(sub chan docker.ContainerUpdateMsg) tea.Cmd {
	return func() tea.Msg {
		return <-sub
	}
}

func (containersList) calculateCPUUsage(currStats, prevStats docker.ContainerStats) float64 {
	var (
		cpuPercent  = 0.0
		cpuDelta    = float64(currStats.CPUStats.CPUUsage.TotalUsage) - float64(prevStats.CPUStats.CPUUsage.TotalUsage)
		systemDelta = float64(currStats.CPUStats.SystemCPUUsage) - float64(prevStats.CPUStats.SystemCPUUsage)
	)

	if systemDelta != 0.0 && cpuDelta != 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(currStats.CPUStats.OnlineCpus) * 100.0
	}

	return cpuPercent
}
