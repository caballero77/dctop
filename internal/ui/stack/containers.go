package stack

import (
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/caballero77/dctop/internal/configuration"
	"github.com/caballero77/dctop/internal/docker"
	"github.com/caballero77/dctop/internal/ui/helpers"
	"github.com/caballero77/dctop/internal/ui/messages"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/exp/maps"
)

type containersList struct {
	table helpers.Table

	selected           int
	scrollPosition     int
	containersListSize int
	containers         []*docker.ContainerInfo
	containersMap      map[string]*docker.ContainerInfo
	cpuUsages          map[string]float64
	focus              bool
	containersService  docker.ContainersService
	updates            chan docker.ContainerMsg

	width  int
	height int

	label               string
	legendStyle         lipgloss.Style
	legendShortcutStyle lipgloss.Style
}

func newContainersList(size int, theme configuration.Theme, containersService docker.ContainersService) (tea.Model, error) {
	getColumnSizes := func(width int) []int {
		return []int{15, width - 46, 10, 15, 6}
	}

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.plain"))
	labeShortcutStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.shortcut"))

	legendStyle := lipgloss.NewStyle().Foreground(theme.GetColor("legend.plain"))
	legendShortcutStyle := lipgloss.NewStyle().Foreground(theme.GetColor("legend.shortcut"))

	updates, err := containersService.GetContainerUpdates()
	if err != nil {
		var model containersList
		return model, fmt.Errorf("error getting containers updates: %w", err)
	}

	model := containersList{
		table: helpers.NewTable(getColumnSizes, theme.Sub("table")),

		containersListSize: size,
		containers:         []*docker.ContainerInfo{},
		containersService:  containersService,
		containersMap:      make(map[string]*docker.ContainerInfo),
		cpuUsages:          make(map[string]float64),

		label:               labeShortcutStyle.Render("c") + labelStyle.Render("ontainers"),
		legendStyle:         legendStyle,
		legendShortcutStyle: legendShortcutStyle,
		updates:             updates,
	}

	return helpers.NewBox(model, theme.Sub("border")), nil
}

func (model containersList) Focus() bool { return model.focus }

func (model containersList) Labels() []string { return []string{model.label} }

func (model containersList) Legends() []string {
	if model.focus {
		return []string{model.getLegend()}
	}

	return []string{}
}

func (model containersList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return model.UpdateAsBoxed(msg)
}

func (model containersList) Init() tea.Cmd {
	return func() tea.Msg {
		return <-model.updates
	}
}

func (model containersList) UpdateAsBoxed(msg tea.Msg) (helpers.BoxedModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyRunes:
			if !model.focus {
				return model, nil
			}

			cmd := model.handleContainerAction(string(msg.Runes))
			if cmd != nil {
				return model, cmd
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
	case messages.FocusTabChangedMsg:
		model.focus = msg.Tab == messages.Containers

		return model, nil
	case messages.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height
	case docker.ContainerMsg:
		cmd := model.handleContainersUpdates(msg)
		return model, cmd
	}

	return model, nil
}

func (model containersList) View() string {
	if len(model.containers) == 0 || model.width == 0 || model.height == 0 {
		return lipgloss.Place(model.width-2, model.height-2, lipgloss.Center, lipgloss.Center, "Can't find any containers associated with selected compose file")
	}
	headers := []string{
		"Name",
		"Image",
		"Status",
		"Ip Address",
		"Cpu%",
	}

	items := make([][]string, len(model.containers))
	for i, container := range model.containers {
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

		stack := model.containersService.Stack()
		items[i] = []string{
			displayContainerName(container.InspectData.Name, stack),
			container.InspectData.Config.Image,
			container.InspectData.State.Status,
			ipAddress,
			fmt.Sprintf("%.2f", cpu),
		}
	}

	return model.table.Render(headers, items, model.width, model.selected, model.scrollPosition, model.height-2)
}

func (model containersList) handleContainerAction(key string) tea.Cmd {
	if len(model.containers) == 0 {
		return nil
	}

	switch key {
	case "s":
		return func() tea.Msg {
			selectedContainer := model.containers[model.selected]
			switch selectedContainer.InspectData.State.Status {
			case "running":
				err := model.containersService.ContainerStop(selectedContainer.InspectData.ID)
				if err != nil {
					slog.Error("error stopping container",
						"id", selectedContainer.InspectData.ID,
						"error", err)
				}
			case "exited", "dead", "created":
				err := model.containersService.ContainerStart(selectedContainer.InspectData.ID)
				if err != nil {
					slog.Error("error starting container",
						"id", selectedContainer.InspectData.ID,
						"error", err)
				}
			}
			return nil
		}
	case "p":
		return func() tea.Msg {
			selectedContainer := model.containers[model.selected]
			switch selectedContainer.InspectData.State.Status {
			case "running":
				err := model.containersService.ContainerPause(selectedContainer.InspectData.ID)
				if err != nil {
					slog.Error("error pausing container",
						"id", selectedContainer.InspectData.ID,
						"error", err)
				}
			case "paused":
				err := model.containersService.ContainerUnpause(selectedContainer.InspectData.ID)
				if err != nil {
					slog.Error("error unpausing container",
						"id", selectedContainer.InspectData.ID,
						"error", err)
				}
			}
			return nil
		}
	case "l":
		if len(model.containers) != 0 {
			selectedContainer := model.containers[model.selected]
			if selectedContainer.InspectData.State.Status != "" {
				return tea.Batch(
					func() tea.Msg {
						return messages.StartListeningLogsMsg{ContainerID: model.containers[model.selected].InspectData.ID}
					},
					func() tea.Msg { return messages.FocusTabChangedMsg{Tab: messages.Logs} },
				)
			}
		}
	case "i":
		if len(model.containers) != 0 {
			selectedContainer := model.containers[model.selected]
			if selectedContainer.InspectData.State.Status != "" {
				return tea.Batch(
					func() tea.Msg { return messages.FocusTabChangedMsg{Tab: messages.Inspect} },
				)
			}
		}
	}
	return nil
}

func (model *containersList) handleContainersUpdates(msg docker.ContainerMsg) tea.Cmd {
	switch msg := msg.(type) {
	case docker.ContainerRemoveMsg:
		model.containers = slices.DeleteFunc(model.containers, func(container *docker.ContainerInfo) bool { return container.InspectData.ID == msg.ID })
		delete(model.containersMap, msg.ID)
		if model.selected >= len(model.containers) && len(model.containers) > 0 {
			model.selectUp()
		}
		if len(model.containers) == 0 {
			return nil
		}
		return model.getContainerSelectedCmd()
	case docker.ContainerUpdateMsg:
		container, ok := model.containersMap[msg.Inspect.ID]
		if ok {
			model.cpuUsages[msg.Inspect.ID] = model.calculateCPUUsage(container.StatsSnapshot, msg.Stats)
			container.InspectData = msg.Inspect
			container.Processes = msg.Processes
			container.StatsSnapshot = msg.Stats
			return nil
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

			return model.getContainerSelectedCmd()
		}
	default:
		return nil
	}
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

func (model containersList) getLegend() string {
	if model.selected >= len(model.containers) {
		return ""
	}
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

	return legend + " " +
		model.legendShortcutStyle.Render("l") + model.legendStyle.Render("ogs") + " " +
		model.legendShortcutStyle.Render("i") + model.legendStyle.Render("nspect")
}

func (model containersList) getContainerSelectedCmd() tea.Cmd {
	if len(model.containers) == 0 && model.selected >= 0 {
		return nil
	}
	return func() tea.Msg { return messages.ContainerSelectedMsg{Container: *model.containers[model.selected]} }
}

func (containersList) calculateCPUUsage(currentStats, prevStats docker.ContainerStats) float64 {
	var (
		cpuPercent  = 0.0
		cpuDelta    = float64(currentStats.CPUStats.CPUUsage.TotalUsage) - float64(prevStats.CPUStats.CPUUsage.TotalUsage)
		systemDelta = float64(currentStats.CPUStats.SystemCPUUsage) - float64(prevStats.CPUStats.SystemCPUUsage)
	)

	if systemDelta != 0.0 && cpuDelta != 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(currentStats.CPUStats.OnlineCpus) * 100.0
	}

	return cpuPercent
}

func displayContainerName(name, stack string) string {
	reg := regexp.MustCompile(fmt.Sprintf("/?(%s-)?(?P<name>[a-zA-Z0-9]+(-[0-9]+)?)", stack))
	index := reg.SubexpIndex("name")
	return reg.FindStringSubmatch(name)[index]
}
