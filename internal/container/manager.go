package container

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Mohamed-faaris/go-mcp/internal/workspace"
)

var (
	ErrNotOwned = errors.New("container manager: container is not owned by this task")
	ErrImageNotAllowed = errors.New("container manager: image is not allowlisted")
)
var safeIdentity = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

type ManagerOptions struct {
	WorkspaceRoot string
	AllowedImages []string
	Network string
	DefaultLimits ResourceLimits
}

type CreateRequest struct {
	OwnerID string
	TaskID string
	ProjectID string
	PresetID string
	Image string
	Workspace string
	Environment map[string]string
	Limits ResourceLimits
}

type Manager struct {
	runtime Runtime
	registry Registry
	workspaceRoot string
	allowedImages map[string]struct{}
	network string
	defaults ResourceLimits
}

func NewManager(runtime Runtime, registry Registry, opts ManagerOptions) (*Manager, error) {
	if runtime == nil || registry == nil {
		return nil, errors.New("container manager: runtime and registry are required")
	}
	root, err := filepath.Abs(opts.WorkspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("container manager: resolve workspace root: %w", err)
	}
	allowed := make(map[string]struct{}, len(opts.AllowedImages))
	for _, image := range opts.AllowedImages {
		image = strings.TrimSpace(image)
		if image != "" {
			allowed[image] = struct{}{}
		}
	}
	if len(allowed) == 0 {
		return nil, errors.New("container manager: at least one workload image must be allowlisted")
	}
	defaults := opts.DefaultLimits
	if defaults.MemoryBytes == 0 { defaults.MemoryBytes = 2 << 30 }
	if defaults.CPUCores == 0 { defaults.CPUCores = 2 }
	if defaults.PidsLimit == 0 { defaults.PidsLimit = 512 }
	if defaults.Timeout == 0 { defaults.Timeout = 2 * time.Hour }
	return &Manager{runtime: runtime, registry: registry, workspaceRoot: root, allowedImages: allowed, network: opts.Network, defaults: defaults}, nil
}

func (m *Manager) CreateTaskContainer(ctx context.Context, req CreateRequest) (Info, error) {
	for label, value := range map[string]string{"owner ID": req.OwnerID, "task ID": req.TaskID, "project ID": req.ProjectID, "preset ID": req.PresetID} {
		if !safeIdentity.MatchString(value) {
			return Info{}, fmt.Errorf("container manager: invalid %s", label)
		}
	}
	if _, ok := m.allowedImages[req.Image]; !ok {
		return Info{}, ErrImageNotAllowed
	}
	workspacePath, err := workspace.Validate(m.workspaceRoot, req.Workspace)
	if err != nil {
		return Info{}, fmt.Errorf("container manager: invalid workspace: %w", err)
	}
	limits := req.Limits
	if limits.MemoryBytes == 0 { limits.MemoryBytes = m.defaults.MemoryBytes }
	if limits.CPUCores == 0 { limits.CPUCores = m.defaults.CPUCores }
	if limits.PidsLimit == 0 { limits.PidsLimit = m.defaults.PidsLimit }
	if limits.Timeout == 0 { limits.Timeout = m.defaults.Timeout }
	if err := validateLimits(limits); err != nil {
		return Info{}, err
	}
	suffix := make([]byte, 5)
	if _, err := rand.Read(suffix); err != nil {
		return Info{}, fmt.Errorf("container manager: generate container name: %w", err)
	}
	spec := Spec{
		Name: fmt.Sprintf("gomcp-task-%s-%s", req.TaskID, hex.EncodeToString(suffix)),
		Image: req.Image,
		Workspace: workspacePath,
		OwnerID: req.OwnerID,
		TaskID: req.TaskID,
		ProjectID: req.ProjectID,
		PresetID: req.PresetID,
		Environment: cloneMap(req.Environment),
		Network: m.network,
		Limits: limits,
		ReadOnlyRoot: true,
		NoNewPrivileges: true,
		DropCapabilities: []string{"ALL"},
	}
	info, err := m.runtime.Create(ctx, spec)
	if err != nil {
		return Info{}, fmt.Errorf("container manager: create workload: %w", err)
	}
	if info.ID == "" {
		return Info{}, errors.New("container manager: runtime returned an empty container ID")
	}
	if !matchesSpec(info, spec) {
		_ = m.runtime.Delete(context.Background(), info.ID)
		return Info{}, errors.New("container manager: runtime metadata does not match requested owner/task labels")
	}
	if err := m.registry.Put(ctx, info); err != nil {
		cleanupErr := m.runtime.Delete(context.Background(), info.ID)
		if cleanupErr != nil {
			return Info{}, fmt.Errorf("container manager: persist ownership: %v; cleanup container: %w", err, cleanupErr)
		}
		return Info{}, fmt.Errorf("container manager: persist ownership: %w", err)
	}
	return cloneInfo(info), nil
}

func (m *Manager) InspectOwned(ctx context.Context, ownerID, taskID, containerID string) (Info, error) {
	record, err := m.ownedRecord(ctx, ownerID, taskID, containerID)
	if err != nil { return Info{}, err }
	observed, err := m.runtime.Inspect(ctx, containerID)
	if err != nil { return Info{}, err }
	if !sameIdentity(record, observed) {
		return Info{}, errors.New("container manager: runtime identity mismatch; refusing access")
	}
	return observed, nil
}

func (m *Manager) DeleteOwned(ctx context.Context, ownerID, taskID, containerID string) error {
	record, err := m.ownedRecord(ctx, ownerID, taskID, containerID)
	if err != nil { return err }
	observed, err := m.runtime.Inspect(ctx, containerID)
	if err != nil { return err }
	if !sameIdentity(record, observed) {
		return errors.New("container manager: runtime identity mismatch; refusing deletion")
	}
	if strings.EqualFold(observed.State, "running") {
		if err := m.runtime.Stop(ctx, containerID, 10*time.Second); err != nil {
			return fmt.Errorf("container manager: stop owned container: %w", err)
		}
	}
	if err := m.runtime.Delete(ctx, containerID); err != nil {
		return fmt.Errorf("container manager: delete owned container: %w", err)
	}
	if err := m.registry.Delete(ctx, containerID); err != nil {
		return fmt.Errorf("container manager: deleted runtime container but failed to remove registry record: %w", err)
	}
	return nil
}

func (m *Manager) ListOwned(ctx context.Context, ownerID, taskID string) ([]Info, error) {
	if !safeIdentity.MatchString(ownerID) || !safeIdentity.MatchString(taskID) {
		return nil, errors.New("container manager: invalid owner or task ID")
	}
	records, err := m.registry.List(ctx)
	if err != nil { return nil, err }
	out := make([]Info, 0)
	for _, record := range records {
		if record.OwnerID == ownerID && record.TaskID == taskID {
			out = append(out, record)
		}
	}
	return out, nil
}

func (m *Manager) ownedRecord(ctx context.Context, ownerID, taskID, containerID string) (Info, error) {
	if !safeIdentity.MatchString(ownerID) || !safeIdentity.MatchString(taskID) || strings.TrimSpace(containerID) == "" {
		return Info{}, errors.New("container manager: owner, task and container IDs are required")
	}
	record, err := m.registry.Get(ctx, containerID)
	if err != nil { return Info{}, err }
	if record.OwnerID != ownerID || record.TaskID != taskID {
		return Info{}, ErrNotOwned
	}
	return record, nil
}

func validateLimits(l ResourceLimits) error {
	if l.MemoryBytes < 256<<20 || l.MemoryBytes > 16<<30 {
		return errors.New("container manager: memory limit must be between 256 MiB and 16 GiB")
	}
	if math.IsNaN(l.CPUCores) || math.IsInf(l.CPUCores, 0) || l.CPUCores < 0.25 || l.CPUCores > 16 {
		return errors.New("container manager: CPU limit must be between 0.25 and 16 cores")
	}
	if l.PidsLimit < 32 || l.PidsLimit > 4096 {
		return errors.New("container manager: PID limit must be between 32 and 4096")
	}
	if l.Timeout < time.Minute || l.Timeout > 24*time.Hour {
		return errors.New("container manager: timeout must be between one minute and 24 hours")
	}
	return nil
}

func matchesSpec(info Info, spec Spec) bool {
	return info.ID != "" && info.OwnerID == spec.OwnerID && info.TaskID == spec.TaskID &&
		info.ProjectID == spec.ProjectID && info.PresetID == spec.PresetID &&
		info.Labels["gomcp.managed"] == "true" && info.Labels["gomcp.owner_id"] == spec.OwnerID &&
		info.Labels["gomcp.task_id"] == spec.TaskID && info.Labels["gomcp.project_id"] == spec.ProjectID &&
		info.Labels["gomcp.preset_id"] == spec.PresetID
}

func sameIdentity(expected, observed Info) bool {
	return expected.ID == observed.ID && expected.OwnerID == observed.OwnerID &&
		expected.TaskID == observed.TaskID && expected.ProjectID == observed.ProjectID &&
		expected.PresetID == observed.PresetID && observed.Labels["gomcp.managed"] == "true" &&
		observed.Labels["gomcp.owner_id"] == expected.OwnerID &&
		observed.Labels["gomcp.task_id"] == expected.TaskID &&
		observed.Labels["gomcp.project_id"] == expected.ProjectID &&
		observed.Labels["gomcp.preset_id"] == expected.PresetID
}

func cloneMap(in map[string]string) map[string]string {
	if in == nil { return nil }
	out := make(map[string]string, len(in))
	for key, value := range in { out[key] = value }
	return out
}
