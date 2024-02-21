package stats

import (
	"github.com/caballero77/dctop/internal/configuration"
	"github.com/caballero77/dctop/internal/docker"
	"github.com/caballero77/dctop/internal/ui/helpers"
	"github.com/caballero77/dctop/internal/ui/messages"
	"github.com/caballero77/dctop/internal/ui/stats/rate"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/exp/maps"
)

type io struct {
	theme configuration.Theme

	containerID string

	read  map[string]tea.Model
	write map[string]tea.Model

	width  int
	height int
}

func newIO(theme configuration.Theme) tea.Model {
	return io{
		read:  make(map[string]tea.Model),
		write: make(map[string]tea.Model),
		theme: theme,
	}
}

func (model io) Init() tea.Cmd {
	models := append(maps.Values(model.read), maps.Values(model.read)...)
	return helpers.Init(models...)
}

func (model io) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.ContainerSelectedMsg:
		model.containerID = msg.Container.InspectData.ID
	case docker.ContainerMsg:
		model.handleContainersUpdates(msg)
	case messages.SizeChangeMsq:
		model.width = msg.Width
		model.height = msg.Height

		models := make([]helpers.Model, 0, len(model.read)+len(model.write))

		for key, read := range model.read {
			key := key
			models = append(models, helpers.NewModel(read, func(m tea.Model) { model.read[key] = m }))
		}

		for key, write := range model.write {
			key := key
			models = append(models, helpers.NewModel(write, func(m tea.Model) { model.write[key] = m }))
		}

		return model, helpers.PassMsg(messages.SizeChangeMsq{Width: msg.Width / 2, Height: msg.Height},
			models...,
		)
	}

	return model, nil
}

func (model *io) handleContainersUpdates(msg docker.ContainerMsg) {
	switch msg := msg.(type) {
	case docker.ContainerUpdateMsg:
		switch msg.Inspect.State.Status {
		case "removing", "exited", "dead", "":
			delete(model.read, msg.Inspect.ID)
			delete(model.write, msg.Inspect.ID)
		case "restarting", "paused", "running", "created":
			readModel, ok := model.read[msg.Inspect.ID]
			if !ok {
				readModel, _ = rate.New[uint64]("io read", model.theme).
					Update(messages.SizeChangeMsq{Width: model.width / 2, Height: model.height})
			}

			writeModel, ok := model.write[msg.Inspect.ID]
			if !ok {
				writeModel, _ = rate.New[uint64]("io write", model.theme).
					Update(messages.SizeChangeMsq{Width: model.width / 2, Height: model.height})
			}

			read, write := model.getIoUsage(&msg.Stats.BlkioStats)

			model.read[msg.Inspect.ID], _ = readModel.Update(rate.PushMsg[uint64]{Value: read})
			model.write[msg.Inspect.ID], _ = writeModel.Update(rate.PushMsg[uint64]{Value: write})
		}
	case docker.ContainerRemoveMsg:
		delete(model.read, msg.ID)
		delete(model.write, msg.ID)
	}
}

func (model io) View() string {
	readModel, ok := model.read[model.containerID]
	if !ok {
		readModel, _ = rate.New[uint64]("io read", model.theme).
			Update(messages.SizeChangeMsq{Width: model.width / 2, Height: model.height})
	}

	writeModel, ok := model.write[model.containerID]
	if !ok {
		writeModel, _ = rate.New[uint64]("io read", model.theme).
			Update(messages.SizeChangeMsq{Width: model.width / 2, Height: model.height})
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, readModel.View(), writeModel.View())
}

func (io) getIoUsage(stats *docker.BlkioStats) (read, write uint64) {
	for i := 0; i < len(stats.IoServiceBytesRecursive); i++ {
		current := stats.IoServiceBytesRecursive[i]
		switch current.Operation {
		case "read":
			read += uint64(current.Value)
		case "write":
			write += uint64(current.Value)
		}
	}
	return read, write
}
