package stats

import (
	"dctop/internal/configuration"
	"dctop/internal/docker"
	"dctop/internal/ui/helpers"
	"dctop/internal/ui/messages"
	"dctop/internal/ui/stats/rate"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/exp/maps"
)

type network struct {
	theme configuration.Theme

	containerID string

	rx map[string]tea.Model
	tx map[string]tea.Model

	width  int
	height int
}

func newNetwork(theme configuration.Theme) network {
	return network{
		rx:    make(map[string]tea.Model),
		tx:    make(map[string]tea.Model),
		theme: theme,
	}
}

func (model network) Init() tea.Cmd {
	models := append(maps.Values(model.rx), maps.Values(model.rx)...)
	return helpers.Init(models...)
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

		models := make([]helpers.Model, 0, len(model.rx)+len(model.tx))

		for key, rx := range model.rx {
			key := key
			models = append(models, helpers.NewModel(rx, func(m tea.Model) { model.rx[key] = m }))
		}

		for key, tx := range model.tx {
			key := key
			models = append(models, helpers.NewModel(tx, func(m tea.Model) { model.tx[key] = m }))
		}

		return model, helpers.PassMsg(messages.SizeChangeMsq{Width: msg.Width / 2, Height: msg.Height},
			models...,
		)
	}

	return model, nil
}

func (model *network) handleContainersUpdates(msg docker.ContainerMsg) {
	switch msg := msg.(type) {
	case docker.ContainerUpdateMsg:
		switch msg.Inspect.State.Status {
		case "removing", "exited", "dead", "":
			delete(model.rx, msg.Inspect.ID)
			delete(model.tx, msg.Inspect.ID)
		case "restarting", "paused", "running", "created":
			readModel, ok := model.rx[msg.Inspect.ID]
			if !ok {
				readModel, _ = rate.New[uint64]("rx", model.theme).
					Update(messages.SizeChangeMsq{Width: model.width / 2, Height: model.height})
			}

			writeModel, ok := model.tx[msg.Inspect.ID]
			if !ok {
				writeModel, _ = rate.New[uint64]("tx", model.theme).
					Update(messages.SizeChangeMsq{Width: model.width / 2, Height: model.height})
			}

			read, write := model.sumNetworkUsage(msg.Stats.Networks)

			model.rx[msg.Inspect.ID], _ = readModel.Update(rate.PushMsg[uint64]{Value: read})
			model.tx[msg.Inspect.ID], _ = writeModel.Update(rate.PushMsg[uint64]{Value: write})
		}
	case docker.ContainerRemoveMsg:
		delete(model.rx, msg.ID)
		delete(model.tx, msg.ID)
	}
}

func (model network) View() string {
	readModel, ok := model.rx[model.containerID]
	if !ok {
		readModel, _ = rate.New[uint64]("rx", model.theme).
			Update(messages.SizeChangeMsq{Width: model.width / 2, Height: model.height})
	}

	writeModel, ok := model.tx[model.containerID]
	if !ok {
		writeModel, _ = rate.New[uint64]("tx", model.theme).
			Update(messages.SizeChangeMsq{Width: model.width / 2, Height: model.height})
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, readModel.View(), writeModel.View())
}

func (network) sumNetworkUsage(networks docker.Networks) (rx, tx uint64) {
	return uint64(networks.Eth0.RxBytes), uint64(networks.Eth0.TxBytes)
}
