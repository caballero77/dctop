package common

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
