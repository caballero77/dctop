package messages

import "dctop/internal/docker"

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

type SetTextMgs struct {
	Text        string
	ResetScroll bool
}

type ClearTextBoxMsg struct{}
