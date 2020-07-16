package helper

import (
	"github.com/docker/docker/api/types"
	v1 "k8s.io/api/core/v1"
	"tkestack.io/nvml"
)

const (
	K8SPOD ProcType = iota
	DOCKER
	NATIVE
)

type ProcType int

func (t ProcType) string() string {
	switch t {
	case K8SPOD:
		return "K8SPOD"
	case DOCKER:
		return "DOCKER"
	case NATIVE:
		return "NATIVE"
	default:
		return ""
	}
}

type PidPraseOut interface {
	GetPodUid() (string, error)
	GetDockerUid() (string, error)
	SetPodUid(uid string) error
	SetDockerUid(uid string) error
}

type ProcHelper interface {
	PraseProc() (PidPraseOut, error)
}

// ProcPraseFunc to prase the /proc/{pid}/cpuset
type ProcPraseFunc func(procInfo string) (PidPraseOut, error)

type ProcPraser interface {
	Prase() error
}

func (p ProcPraseFunc) Prase(procInfo string) (PidPraseOut, error) {
	return p(procInfo)
}

type PodGPUInfo struct {
	Pod         v1.Pod
	ProcessInfo *nvml.ProcessInfo
}

type ContainerHelper interface {
	GetContainers(processInfo []*nvml.ProcessInfo) ([]*types.Container, error)
	GetK8sPods(processInfo []*nvml.ProcessInfo) ([]*PodGPUInfo, error)
}
