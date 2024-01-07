package docker

import (
	"github.com/docker/docker/api/types"
)

type ContainerMessageType string

const (
	Update ContainerMessageType = "update"
	Add    ContainerMessageType = "add"
	Remove ContainerMessageType = "remove"
)

type ContainerMsg interface {
	Type() ContainerMessageType
}

type ContainerUpdateMsg struct {
	ID        string
	Stats     ContainerStats
	Inspect   types.ContainerJSON
	Processes []Process
}

func (msg ContainerUpdateMsg) Type() ContainerMessageType {
	return Update
}

type ContainerCreateMsg struct {
	ID string
}

func (msg ContainerCreateMsg) Type() ContainerMessageType {
	return Add
}

type ContainerRemoveMsg struct {
	ID string
}

func (msg ContainerRemoveMsg) Type() ContainerMessageType {
	return Remove
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
	Version  string             `yaml:"version"`
	Services map[string]Service `yaml:"services"`
	Networks map[string]Network `yaml:"networks"`
}

type Service struct {
	Name   string `yaml:"container_name"`
	Image  string `yaml:"image"`
	Deploy Deploy `yaml:"deploy"`
}

type Deploy struct {
	Mode     string `yaml:"mode"`
	Replicas int    `yaml:"replicas"`
}

type Network struct {
	Driver        string            `yaml:"driver"`
	DriverOptions map[string]string `yaml:"driver_opts"`
	Attachable    bool              `yaml:"attachable"`
	EnableIpv6    bool              `yaml:"enable_ipv6"`
	External      bool              `yaml:"external"`
	Internal      bool              `yaml:"internal"`
	Labels        map[string]string `yaml:"labels"`
	Name          string            `yaml:"name"`
}
