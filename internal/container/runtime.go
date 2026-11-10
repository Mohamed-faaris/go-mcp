// Package container defines the trusted control-plane contract for managed workloads.
// The interface must never be exposed directly to a workload agent.
package container

import (
	"context"
	"time"
)

type ResourceLimits struct {
	MemoryBytes int64         `json:"memoryBytes"`
	CPUCores    float64       `json:"cpuCores"`
	PidsLimit   int64         `json:"pidsLimit"`
	Timeout     time.Duration `json:"timeout"`
}

type Spec struct {
	Name             string            `json:"name"`
	Image            string            `json:"image"`
	Workspace        string            `json:"workspace"`
	OwnerID          string            `json:"ownerId"`
	TaskID           string            `json:"taskId"`
	ProjectID        string            `json:"projectId"`
	PresetID         string            `json:"presetId"`
	Environment      map[string]string `json:"environment,omitempty"`
	Network          string            `json:"network,omitempty"`
	Limits           ResourceLimits    `json:"limits"`
	ReadOnlyRoot     bool              `json:"readOnlyRoot"`
	NoNewPrivileges  bool              `json:"noNewPrivileges"`
	DropCapabilities []string          `json:"dropCapabilities"`
}

type Info struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	State     string            `json:"state"`
	OwnerID   string            `json:"ownerId"`
	TaskID    string            `json:"taskId"`
	ProjectID string            `json:"projectId"`
	PresetID  string            `json:"presetId"`
	CreatedAt time.Time         `json:"createdAt"`
	Labels    map[string]string `json:"labels"`
}

type Runtime interface {
	Create(context.Context, Spec) (Info, error)
	Start(context.Context, string) error
	Stop(context.Context, string, time.Duration) error
	Delete(context.Context, string) error
	Inspect(context.Context, string) (Info, error)
	ListManaged(context.Context) ([]Info, error)
}
