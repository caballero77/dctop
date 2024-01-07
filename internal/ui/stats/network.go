package stats

import (
	"dctop/internal/configuration"
	"dctop/internal/docker"
	"dctop/internal/ui/common"
	memory_utils "dctop/internal/utils/memory"
	"dctop/internal/utils/queues"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ContainerNetworks struct {
	IncomingNetwork *queues.Queue[int]
	OutgoingNetwork *queues.Queue[int]
}

type network struct {
	box         *common.BoxWithBorders
	plotStyles  lipgloss.Style
	labelStyle  lipgloss.Style
	legendStyle lipgloss.Style
	scaling     []int

	containerID string
	networks    map[string]ContainerNetworks

	width  int
	height int
}

func newNetwork(theme configuration.Theme) network {
	return network{
		box:         common.NewBoxWithLabel(theme.Sub("border")),
		plotStyles:  lipgloss.NewStyle().Foreground(theme.GetColor("plot")),
		labelStyle:  lipgloss.NewStyle().Bold(true).Foreground(theme.GetColor("title.plain")),
		legendStyle: lipgloss.NewStyle().Foreground(theme.GetColor("legend.plain")),
		networks:    make(map[string]ContainerNetworks),
		scaling:     []int{15, 25, 35, 45, 55, 65, 75, 100},
	}
}

func (model network) Init() tea.Cmd {
	return nil
}

func (model network) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case common.ContainerSelectedMsg:
		model.containerID = msg.Container.InspectData.ID
	case docker.ContainerMsg:
		model = model.handleContainersUpdates(msg)
	case common.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height
	}
	return model, nil
}

func (model network) handleContainersUpdates(msg docker.ContainerMsg) network {
	switch msg := msg.(type) {
	case docker.ContainerUpdateMsg:
		switch msg.Inspect.State.Status {
		case "removing", "exited", "dead", "":
			delete(model.networks, msg.Inspect.ID)
		case "restarting", "paused", "running", "created":
			network, ok := model.networks[msg.Inspect.ID]
			if !ok {
				network = ContainerNetworks{
					IncomingNetwork: queues.New[int](),
					OutgoingNetwork: queues.New[int](),
				}
				model.networks[msg.Inspect.ID] = network
			}
			rx, tx := model.sumNetworkUsage(msg.Stats.Networks)
			err := pushWithLimit(network.IncomingNetwork, rx, model.width*2)
			if err != nil {
				panic(err)
			}
			err = pushWithLimit(network.OutgoingNetwork, tx, model.width*2)
			if err != nil {
				panic(err)
			}
		}
	case docker.ContainerRemoveMsg:
		delete(model.networks, msg.ID)
	}
	return model
}

func (model network) View() string {
	network, ok := model.networks[model.containerID]
	width := model.width - 4
	height := model.height - 2
	if !ok || (network.IncomingNetwork.Len() == 0 && network.OutgoingNetwork.Len() == 0) {
		return model.box.Render([]string{}, []string{}, lipgloss.PlaceVertical(height, lipgloss.Center, lipgloss.PlaceHorizontal(width, lipgloss.Center, "no data")), false)
	}
	incoming := model.RenderNetwork(network.IncomingNetwork, func(current string) string { return model.labelStyle.Render(fmt.Sprintf("RX: %s/sec", current)) }, width/2, height)

	outcoming := model.RenderNetwork(network.OutgoingNetwork, func(current string) string { return model.labelStyle.Render(fmt.Sprintf("TX: %s/sec", current)) }, width/2+width%2, height)

	return lipgloss.JoinHorizontal(lipgloss.Center, incoming, outcoming)
}

func (model network) RenderNetwork(queue *queues.Queue[int], label func(string) string, width, height int) string {
	if queue.Len() == 0 {
		return model.box.Render([]string{}, []string{}, lipgloss.PlaceVertical(height, lipgloss.Center, lipgloss.PlaceHorizontal(width, lipgloss.Center, "no data")), false)
	}

	total, err := queue.Head()
	if err != nil {
		panic(err)
	}

	data, max, maxChange, current := getDataChangeFromQueue(queue.ToArray(), width)
	scale := model.calculateScalingKoeficient(max)
	plot := model.plotStyles.Render(renderPlot(data, scale, width, height))

	currentRx, err := memory_utils.BytesToReadable(float64(total))
	if err != nil {
		panic(err)
	}

	maxRxChange, err := memory_utils.BytesToReadable(maxChange)
	if err != nil {
		panic(err)
	}

	currentChange, err := memory_utils.BytesToReadable(current)
	if err != nil {
		panic(err)
	}

	legends := []string{
		model.legendStyle.Render(fmt.Sprintf("Total: %s", currentRx)),
		model.legendStyle.Render(fmt.Sprintf("Max: %s/sec", maxRxChange)),
	}

	length := 0
	for _, legend := range legends {
		length += lipgloss.Width(legend)
	}
	i := len(legends) - 1
	for len(legends) != 0 && i >= 0 {
		if length+2 >= width {
			length -= len(legends[i])
			legends = legends[:i]
			i--
		} else {
			break
		}
	}

	return model.box.Render([]string{label(currentChange)}, legends, plot, false)
}

func (model network) calculateScalingKoeficient(maxValue float64) float64 {
	for i := 0; i < len(model.scaling); i++ {
		if maxValue < float64(model.scaling[i]) {
			return float64(model.scaling[i]) / 100
		}
	}
	return 1
}

func (network) sumNetworkUsage(networks docker.Networks) (rx, tx int) {
	return networks.Eth0.RxBytes, networks.Eth0.TxBytes
}
