package docker

import (
	"github.com/docker/docker/api/types"
)

type ContainerUpdateMsg struct {
	ID        string
	Stats     ContainerStats
	Inspect   types.ContainerJSON
	Processes []Process
}

type ComposeData struct {
	Stack      string
	Containers map[string]*ContainerInfo
}

type ContainerInfo struct {
	InspectData   types.ContainerJSON
	StatsSnapshot ContainerStats
	Processes     []Process
}

type Compose struct {
	Version  string             `yml:"version"`
	Services map[string]Service `yml:"services"`
	Networks map[string]Network `yml:"networks"`
}

type Service struct{}

type Network struct {
	Driver        string            `yml:"driver"`
	DriverOptions map[string]string `yml:"driver_opts"`
	Attachable    bool              `yml:"attachable"`
	EnableIpv6    bool              `yml:"enable_ipv6"`
	External      bool              `yml:"external"`
	Internal      bool              `yml:"internal"`
	Labels        map[string]string `yml:"labels"`
	Name          string            `yml:"name"`
}
