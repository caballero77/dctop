package compose

import (
	"dctop/internal/configuration"
	"dctop/internal/docker"
	"dctop/internal/ui/common"
	"dctop/internal/utils"
	"dctop/internal/utils/maps"
	"dctop/internal/utils/slices"
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type containersList struct {
	box   *common.BoxWithBorders
	table *common.Table

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

	label               string
	legendStyle         lipgloss.Style
	legendShortcutStyle lipgloss.Style
}

func newContainersList(size int, theme configuration.Theme, service *docker.ComposeService) containersList {
	getColumnSizes := func(width int) []int {
		return []int{15, width - 46, 10, 15, 6}
	}

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.plain"))
	labeShortcutStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.shortcut"))

	legendStyle := lipgloss.NewStyle().Foreground(theme.GetColor("legend.plain"))
	legendShortcutStyle := lipgloss.NewStyle().Foreground(theme.GetColor("legend.shortcut"))

	return containersList{
		box:   common.NewBoxWithLabel(theme.Sub("border")),
		table: common.NewTable(getColumnSizes, theme.Sub("table")),

		containersListSize: size,
		selected:           0,
		scrollPosition:     0,
		containers:         []*docker.ContainerInfo{},
		service:            service,
		containersMap:      make(map[string]*docker.ContainerInfo),
		cpuUsages:          make(map[string]float64),

		label:               labeShortcutStyle.Render("c") + labelStyle.Render("ontainers"),
		legendStyle:         legendStyle,
		legendShortcutStyle: legendShortcutStyle,
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
			case "s":
				return model, func() tea.Msg {
					selectedContainer := model.containers[model.selected]
					switch selectedContainer.InspectData.State.Status {
					case "running":
						err := model.service.ContainerStop(selectedContainer.InspectData.ID)
						if err != nil {
							panic(err)
						}
					case "exited", "dead", "created":
						err := model.service.ContainerStart(selectedContainer.InspectData.ID)
						if err != nil {
							panic(err)
						}
					}
					return nil
				}
			case "p":
				return model, func() tea.Msg {
					selectedContainer := model.containers[model.selected]
					switch selectedContainer.InspectData.State.Status {
					case "running":
						err := model.service.ContainerPause(selectedContainer.InspectData.ID)
						if err != nil {
							panic(err)
						}
					case "paused":
						err := model.service.ContainerUnpause(selectedContainer.InspectData.ID)
						if err != nil {
							panic(err)
						}
					}
					return nil
				}
			case "l":
				if len(model.containers) != 0 {
					selectedContainer := model.containers[model.selected]
					if selectedContainer.InspectData.State.Status != "" {
						return model, tea.Batch(
							func() tea.Msg {
								return common.StartListenningLogsMsg{ContainerID: model.containers[model.selected].InspectData.ID}
							},
							func() tea.Msg { return common.FocusTabChangedMsg{Tab: common.Logs} },
						)
					}
					return model, nil
				}
			}
		case tea.KeyUp:
			if model.focus && len(model.containers) > 0 {
				model.selectUp()
				return model, model.getContainerSelectedCmd()
			}
			return model, nil
		case tea.KeyDown:
			if model.focus && len(model.containers) > 0 {
				model.selectDown()
				return model, model.getContainerSelectedCmd()
			}
			return model, nil
		}
	case common.FocusTabChangedMsg:
		model.focus = msg.Tab == common.Containers

		return model, nil
	case common.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height
	case docker.ContainerMsg:
		return model.handleContainersUpdates(msg)
	}

	return model, nil
}

func (model containersList) handleContainersUpdates(msg docker.ContainerMsg) (containersList, tea.Cmd) {
	switch msg := msg.(type) {
	case docker.ContainerRemoveMsg:
		model.containers = slices.Remove(model.containers, func(container *docker.ContainerInfo) bool { return container.InspectData.ID == msg.ID })
		delete(model.containersMap, msg.ID)
		if model.selected >= len(model.containers) && len(model.containers) > 0 {
			model.selectUp()
		}
		if len(model.containers) == 0 {
			return model, nil
		}
		return model, model.getContainerSelectedCmd()
	case docker.ContainerUpdateMsg:
		container, ok := model.containersMap[msg.Inspect.ID]
		if ok {
			model.cpuUsages[msg.Inspect.ID] = model.calculateCPUUsage(container.StatsSnapshot, msg.Stats)
			container.InspectData = msg.Inspect
			container.Processes = msg.Processes
			container.StatsSnapshot = msg.Stats
			return model, nil
		} else {
			container = &docker.ContainerInfo{
				InspectData:   msg.Inspect,
				StatsSnapshot: msg.Stats,
				Processes:     make([]docker.Process, 0),
			}
			model.containersMap[msg.Inspect.ID] = container
			model.containers = append(model.containers, container)
			containers := model.containers
			sort.Slice(containers, func(i, j int) bool { return containers[i].InspectData.Name < containers[j].InspectData.Name })

			return model, model.getContainerSelectedCmd()
		}
	default:
		return model, nil
	}
}

func (model containersList) View() string {
	if len(model.containers) == 0 || model.width == 0 || model.height == 0 {
		return model.box.Render(
			[]string{model.label},
			[]string{},
			lipgloss.Place(model.width-2, model.height-2, lipgloss.Center, lipgloss.Center, "Can't find any containers associated with selected compose file"),
			model.focus,
		)
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

		var ipAddress string
		if container.InspectData.NetworkSettings != nil && len(container.InspectData.NetworkSettings.Networks) > 0 {
			ipAddress = container.InspectData.NetworkSettings.Networks[networkNames[0]].IPAddress
		} else {
			ipAddress = strings.Repeat("-", 15)
		}

		stack := model.service.Stack()
		return []string{
			utils.DisplayContainerName(container.InspectData.Name, stack),
			container.InspectData.Config.Image,
			container.InspectData.State.Status,
			ipAddress,
			fmt.Sprintf("%.2f", cpu),
		}, nil
	})
	if err != nil {
		panic(err)
	}

	body, err := model.table.Render(headers, items, model.width, model.selected, model.scrollPosition, model.height-2)
	if err != nil {
		panic(err)
	}

	legend := ""
	if model.focus {
		legend = model.getLegend()
	}

	// /fmt.Print(len(model.containers))
	return model.box.Render(
		[]string{model.label},
		[]string{legend},
		body,
		model.focus,
	)
}

func (model containersList) getLegend() string {
	var legend string
	switch model.containers[model.selected].InspectData.State.Status {
	case "running":
		legend = model.legendShortcutStyle.Render("s") + model.legendStyle.Render("top") + " " +
			model.legendShortcutStyle.Render("p") + model.legendStyle.Render("ause")
	case "exited", "dead", "created":
		legend = model.legendShortcutStyle.Render("s") + model.legendStyle.Render("tart")
	case "paused":
		legend = model.legendStyle.Render("un") + model.legendShortcutStyle.Render("p") + model.legendStyle.Render("ause")
	}

	return legend + " " + model.legendShortcutStyle.Render("l") + model.legendStyle.Render("ogs")
}

func (model *containersList) selectUp() {
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

func (model *containersList) selectDown() {
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

func (model containersList) getContainerSelectedCmd() tea.Cmd {
	if len(model.containers) == 0 && model.selected >= 0 {
		return nil
	}
	return func() tea.Msg { return common.ContainerSelectedMsg{Container: *model.containers[model.selected]} }
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
