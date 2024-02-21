package messages

import "github.com/caballero77/dctop/internal/docker"

type SizeChangeMsq struct {
	Width  int
	Height int
}

type ContainerSelectedMsg struct {
	Container docker.ContainerInfo
}

type StartListeningLogsMsg struct {
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
