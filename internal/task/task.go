package task

import (
  "errors"
  "fmt"
  "time"
)

type Status string

const (
  StatusQueued Status = "queued"
  StatusProvisioning Status = "provisioning"
  StatusRunning Status = "running"
  StatusWaitingApproval Status = "waiting_approval"
  StatusValidating Status = "validating"
  StatusCompleted Status = "completed"
  StatusFailed Status = "failed"
  StatusCancelled Status = "cancelled"
  StatusCleanupPending Status = "cleanup_pending"
)

var ErrInvalidTransition = errors.New("task: invalid status transition")

type Task struct {
  ID string `json:"id"`
  OwnerID string `json:"ownerId"`
  ProjectID string `json:"projectId"`
  ProjectPath string `json:"projectPath"`
  Prompt string `json:"prompt"`
  PresetID string `json:"presetId"`
  Status Status `json:"status"`
  ContainerID string `json:"containerId,omitempty"`
  WorkspacePath string `json:"workspacePath,omitempty"`
  RemoteURL string `json:"remoteUrl,omitempty"`
  BranchName string `json:"branchName,omitempty"`
  BaseCommit string `json:"baseCommit,omitempty"`
  ResultCommit string `json:"resultCommit,omitempty"`
  Summary string `json:"summary,omitempty"`
  Error string `json:"error,omitempty"`
  CreatedAt time.Time `json:"createdAt"`
  StartedAt *time.Time `json:"startedAt,omitempty"`
  FinishedAt *time.Time `json:"finishedAt,omitempty"`
  UpdatedAt time.Time `json:"updatedAt"`
}

type CheckResult struct {
  Name string `json:"name"`
  Command string `json:"command,omitempty"`
  Status string `json:"status"`
  ExitCode *int `json:"exitCode,omitempty"`
  Duration string `json:"duration,omitempty"`
  Output string `json:"output,omitempty"`
}

type Report struct {
  TaskID string `json:"taskId"`
  Summary string `json:"summary"`
  ChangedFiles []string `json:"changedFiles"`
  Checks []CheckResult `json:"checks"`
  Commits []string `json:"commits"`
  Warnings []string `json:"warnings"`
  Artifacts []string `json:"artifacts"`
  StartedAt time.Time `json:"startedAt"`
  FinishedAt time.Time `json:"finishedAt"`
}

func New(id, ownerID, projectID, prompt string, now time.Time) (*Task, error) {
  if id == "" || ownerID == "" || projectID == "" || prompt == "" {
    return nil, errors.New("task: id, owner, project, and prompt are required")
  }
  if now.IsZero() { now = time.Now().UTC() }
  now = now.UTC()
  return &Task{ID:id, OwnerID:ownerID, ProjectID:projectID, Prompt:prompt, Status:StatusQueued, CreatedAt:now, UpdatedAt:now}, nil
}

func CanTransition(from, to Status) bool {
  allowed := map[Status]map[Status]bool{
    StatusQueued: {StatusProvisioning:true, StatusCancelled:true, StatusFailed:true},
    StatusProvisioning: {StatusRunning:true, StatusFailed:true, StatusCancelled:true, StatusCleanupPending:true},
    StatusRunning: {StatusWaitingApproval:true, StatusValidating:true, StatusFailed:true, StatusCancelled:true, StatusCleanupPending:true},
    StatusWaitingApproval: {StatusRunning:true, StatusFailed:true, StatusCancelled:true},
    StatusValidating: {StatusCompleted:true, StatusFailed:true, StatusCancelled:true, StatusCleanupPending:true},
    StatusCompleted: {StatusCleanupPending:true},
    StatusFailed: {StatusCleanupPending:true},
    StatusCancelled: {StatusCleanupPending:true},
    StatusCleanupPending: {},
  }
  return allowed[from][to]
}

func (t *Task) Transition(next Status, at time.Time) error {
  if t == nil { return errors.New("task: nil task") }
  if !CanTransition(t.Status, next) { return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, t.Status, next) }
  if at.IsZero() { at = time.Now().UTC() }
  at = at.UTC()
  t.Status, t.UpdatedAt = next, at
  if next == StatusRunning && t.StartedAt == nil { t.StartedAt = &at }
  if next == StatusCompleted || next == StatusFailed || next == StatusCancelled { t.FinishedAt = &at }
  return nil
}
