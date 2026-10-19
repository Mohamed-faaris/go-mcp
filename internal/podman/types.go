package podman

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// UnixTime unmarshals either an RFC3339 timestamp string or a unix-seconds
// number (Podman returns different formats across versions/endpoints) and
// always exposes unix seconds as the JSON number.
type UnixTime int64

func (t *UnixTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), `"`)
	if s == "" || s == "null" {
		*t = 0
		return nil
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		*t = UnixTime(n)
		return nil
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		*t = UnixTime(f)
		return nil
	}
	if ts, err := time.Parse(time.RFC3339Nano, s); err == nil {
		*t = UnixTime(ts.Unix())
		return nil
	}
	return fmt.Errorf("podman: cannot parse time %q", s)
}

// ContainerSummary represents brief container details from Podman list endpoint.
type ContainerSummary struct {
	ID         string            `json:"Id"`
	Names      []string          `json:"Names"`
	Image      string            `json:"Image"`
	State      string            `json:"State"`
	Status     string            `json:"Status"`
	Created    UnixTime          `json:"Created"`
	Labels     map[string]string `json:"Labels"`
	Mounts     []string          `json:"Mounts"`
	ProjectDir string            `json:"projectDir,omitempty"`
}

// ContainerInspect represents full container details.
type ContainerInspect struct {
	ID      string            `json:"Id"`
	Name    string            `json:"Name"`
	Created time.Time         `json:"Created"`
	State   ContainerState    `json:"State"`
	Config  ContainerConfig   `json:"Config"`
	HostCfg HostConfig        `json:"HostConfig"`
	Mounts  []Mount           `json:"Mounts"`
}

type ContainerState struct {
	Status     string    `json:"Status"`
	Running    bool      `json:"Running"`
	ExitCode   int       `json:"ExitCode"`
	StartedAt  time.Time `json:"StartedAt"`
	FinishedAt time.Time `json:"FinishedAt"`
}

type ContainerConfig struct {
	Image  string            `json:"Image"`
	Cmd    []string          `json:"Cmd"`
	Env    []string          `json:"Env"`
	Labels map[string]string `json:"Labels"`
}

type HostConfig struct {
	Memory   int64 `json:"Memory"`
	NanoCPUs int64 `json:"NanoCpus"`
}

type Mount struct {
	Type        string `json:"Type"`
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
	Mode        string `json:"Mode,omitempty"`
	RW          bool   `json:"RW,omitempty"`
}

type ImageSummary struct {
	ID      string   `json:"Id"`
	RepoTags []string `json:"RepoTags"`
	Size    int64    `json:"Size"`
	Created int64    `json:"Created"`
}

// CreateContainerOpts contains options for creating a new agent sandbox container.
type CreateContainerOpts struct {
	Name        string            `json:"name"`
	Image       string            `json:"image"`
	ProjectDir  string            `json:"projectDir"`  // Host path to bind-mount to /workspace
	MemoryLimit string            `json:"memoryLimit"` // e.g. "2g"
	CPULimit    string            `json:"cpuLimit"`    // e.g. "2"
	Env         []string          `json:"env"`
	MCPTools    []MCPToolSpec     `json:"mcpTools"`
	Labels      map[string]string `json:"labels"`
	Devices     []string          `json:"devices"`     // Host device paths like /dev/video0
}

type MCPToolSpec struct {
	Name    string   `json:"name"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// ExecCreateConfig is passed to create an exec session.
type ExecCreateConfig struct {
	AttachStdin  bool     `json:"AttachStdin"`
	AttachStdout bool     `json:"AttachStdout"`
	AttachStderr bool     `json:"AttachStderr"`
	Tty          bool     `json:"Tty"`
	Cmd          []string `json:"Cmd"`
	Env          []string `json:"Env,omitempty"`
}

type ExecCreateResponse struct {
	ID string `json:"Id"`
}
