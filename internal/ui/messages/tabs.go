package messages

type Tab string

const (
	Containers Tab = "containers"
	Processes  Tab = "processes"
	Logs       Tab = "logs"
	Inspect    Tab = "inspect"
	Compose    Tab = "compose"
)

type FocusTabChangedMsg struct {
	Tab Tab
}

type CloseTabMsg struct {
	Tab Tab
}

func (tab Tab) IsDetailsTab() bool {
	return tab == Logs || tab == Inspect || tab == Compose
}
