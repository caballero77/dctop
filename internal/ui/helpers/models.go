package helpers

import (
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	model  tea.Model
	setter func(tea.Model)
}

type ModelWithMsg struct {
	Model
	msg tea.Msg
}

func NewModel(model tea.Model, setter func(tea.Model)) Model {
	return Model{
		model:  model,
		setter: setter,
	}
}

func PassMsg(msg tea.Msg, models ...Model) tea.Cmd {
	commands := make([]tea.Cmd, len(models))
	for i, value := range models {
		var newModel tea.Model
		newModel, commands[i] = value.model.Update(msg)
		value.setter(newModel)
	}
	return tea.Batch(commands...)
}

func (base Model) WithMsg(msg tea.Msg) ModelWithMsg {
	return ModelWithMsg{
		Model: base,
		msg:   msg,
	}
}

func PassMsgs(sizes ...ModelWithMsg) tea.Cmd {
	commands := make([]tea.Cmd, len(sizes))
	for i, value := range sizes {
		var newModel tea.Model
		newModel, commands[i] = value.model.Update(value.msg)
		value.setter(newModel)

	}
	return tea.Batch(commands...)
}

func Init(models ...tea.Model) tea.Cmd {
	commands := make([]tea.Cmd, len(models))
	for i, model := range models {
		commands[i] = model.Init()
	}
	return tea.Batch(commands...)
}
