package helpers

import (
	tea "github.com/charmbracelet/bubbletea"
)

type Tab struct {
	model tea.Model
}

type Tabs struct {
	tabs []Tab
}

func NewTabs(tabs ...Tab) Tabs {
	return Tabs{
		tabs: tabs,
	}
}

func (model Tabs) Init() tea.Cmd {
	models := make([]tea.Model, len(model.tabs))
	for i, tab := range model.tabs {
		models[i] = tab.model
	}

	return Init(models...)
}

func (model Tabs) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	commands := make([]tea.Cmd, 0)
	var cmd tea.Cmd

	for i, tab := range model.tabs {
		model.tabs[i].model, cmd = tab.model.Update(msg)

		if cmd != nil {
			commands = append(commands, cmd)
		}
	}

	return model, tea.Batch(commands...)
}

func (model Tabs) View() string {

	return ""
}
