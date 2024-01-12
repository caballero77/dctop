package messages

import "dctop/internal/docker"

type Tab string

const (
	Containers Tab = "containers"
	Processes  Tab = "processes"
	Logs       Tab = "logs"
	Compose    Tab = "compose"
)

type FocusTabChangedMsg struct {
	Tab Tab
}

type SizeChangeMsq struct {
	Width  int
	Height int
}

type ContainerSelectedMsg struct {
	Container docker.ContainerInfo
}

type StartListenningLogsMsg struct {
	ContainerID string
}

type ScrollMsg struct {
	Change int
}

type AppendTextMgs struct {
	Text         string
	AdjustScroll bool
}

type ClearTextBoxMsg struct{}