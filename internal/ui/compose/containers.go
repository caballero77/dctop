package compose

import (
	"dctop/internal/configuration"
	"dctop/internal/docker"
	"dctop/internal/ui/common"
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

	label  string
	legend string
}

func newContainersList(size int, theme configuration.Theme, service *docker.ComposeService, focus bool) containersList {
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
		focus:              focus,
		service:            service,
		containersMap:      make(map[string]*docker.ContainerInfo),
		cpuUsages:          make(map[string]float64),

		label: labeShortcutStyle.Render("C") + labelStyle.Render("ontainers"),
		legend: legendShortcutStyle.Render("¹") + legendStyle.Render("stop") + " " +
			legendShortcutStyle.Render("²") + legendStyle.Render("start") + " " +
			legendShortcutStyle.Render("³") + legendStyle.Render("pause") + " " +
			legendShortcutStyle.Render("⁴") + legendStyle.Render("unpause") + " " +
			legendShortcutStyle.Render("l") + legendStyle.Render("ogs"),
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
				model.selectUp()
				return model, model.getContainerSelectedCmd()
			}
			return model, nil
		case tea.KeyDown:
			if model.focus {
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

			return model, tea.Batch(model.getContainerSelectedCmd(), waitForActivity(updates))
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

	legend := ""
	if model.focus {
		legend = model.legend
	}

	return model.box.Render(
		[]string{model.label},
		[]string{legend},
		body,
		model.focus,
	)
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
	return func() tea.Msg { return common.ContainerSelectedMsg{Container: *model.containers[model.selected]} }
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
