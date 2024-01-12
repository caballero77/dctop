package stats

import (
	"dctop/internal/configuration"
	"dctop/internal/docker"
	"dctop/internal/ui/helpers"
	"dctop/internal/ui/messages"
	"dctop/internal/utils/queues"
	"fmt"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

type ContainerNetworks struct {
	IncomingNetwork *queues.Queue[uint64]
	OutgoingNetwork *queues.Queue[uint64]
}

type network struct {
	box         helpers.BoxWithBorders
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
		box:         helpers.NewBox(theme.Sub("border")),
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
	case messages.ContainerSelectedMsg:
		model.containerID = msg.Container.InspectData.ID
	case docker.ContainerMsg:
		model.handleContainersUpdates(msg)
	case messages.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height
	}
	return model, nil
}

func (model *network) handleContainersUpdates(msg docker.ContainerMsg) {
	switch msg := msg.(type) {
	case docker.ContainerUpdateMsg:
		switch msg.Inspect.State.Status {
		case "removing", "exited", "dead", "":
			delete(model.networks, msg.Inspect.ID)
		case "restarting", "paused", "running", "created":
			network, ok := model.networks[msg.Inspect.ID]
			if !ok {
				network = ContainerNetworks{
					IncomingNetwork: queues.New[uint64](),
					OutgoingNetwork: queues.New[uint64](),
				}
				model.networks[msg.Inspect.ID] = network
			}
			rx, tx := model.sumNetworkUsage(msg.Stats.Networks)
			err := network.IncomingNetwork.PushWithLimit(rx, model.width)
			if err != nil {
				if err != nil {
					slog.Error("error pushing element into queue with limit")
				}
			}
			err = network.OutgoingNetwork.PushWithLimit(tx, model.width)
			if err != nil {
				if err != nil {
					slog.Error("error pushing element into queue with limit")
				}
			}
		}
	case docker.ContainerRemoveMsg:
		delete(model.networks, msg.ID)
	}
}

func (model network) View() string {
	network, ok := model.networks[model.containerID]
	width := model.width - 4
	height := model.height - 2
	if !ok {
		network = ContainerNetworks{
			IncomingNetwork: queues.New[uint64](),
			OutgoingNetwork: queues.New[uint64](),
		}
	}

	getLabelRenderer := func(networkType string) func(string) string {
		return func(current string) string {
			if current == "" {
				return model.labelStyle.Render(networkType)
			}
			return model.labelStyle.Render(fmt.Sprintf("%s: %s/sec", networkType, current))
		}
	}

	incoming, err := model.renderNetwork(network.IncomingNetwork, getLabelRenderer("rx"), width/2, height)
	if err != nil {
		slog.Error("error rendering network rx plot",
			"error", err,
			"id", model.containerID)
		return model.renderErrorMessage(width, height)
	}

	outcoming, err := model.renderNetwork(network.OutgoingNetwork, getLabelRenderer("tx"), width/2+width%2, height)
	if err != nil {
		slog.Error("error rendering network tx plot",
			"error", err,
			"id", model.containerID)
		return model.renderErrorMessage(width, height)
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, incoming, outcoming)
}

func (model network) renderNetwork(queue *queues.Queue[uint64], label func(string) string, width, height int) (string, error) {
	if queue.Len() <= 1 {
		return model.box.Render(
			[]string{label("")},
			[]string{},
			lipgloss.PlaceVertical(height, lipgloss.Center, lipgloss.PlaceHorizontal(width, lipgloss.Center, "no data")),
			false,
		), nil
	}

	total, err := queue.Head()
	if err != nil {
		return "", fmt.Errorf("error getting head from memory usage queue: %w", err)
	}

	data, maxRate, currentRate := getRate(queue.ToArray())
	plot := model.plotStyles.Render(renderPlot(data, 1, width, height))

	legends := []string{
		model.legendStyle.Render(fmt.Sprintf("total: %s", humanize.IBytes(total))),
		model.legendStyle.Render(fmt.Sprintf("max: %s/sec", humanize.IBytes(maxRate))),
	}

	return model.box.Render([]string{label(humanize.IBytes(currentRate))}, legends, plot, false), nil
}

func (model network) renderErrorMessage(height, width int) string {
	return model.box.Render(
		[]string{model.labelStyle.Render("cpu")},
		[]string{},
		lipgloss.PlaceVertical(height, lipgloss.Center, lipgloss.PlaceHorizontal(width, lipgloss.Center, "error rendering network usage plot")),
		false,
	)
}

func (network) sumNetworkUsage(networks docker.Networks) (rx, tx uint64) {
	return uint64(networks.Eth0.RxBytes), uint64(networks.Eth0.TxBytes)
}
