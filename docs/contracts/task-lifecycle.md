# Task Lifecycle Contract

## Task resource

```json
{
  "id": "task_uuid",
  "projectId": "project_uuid",
  "ownerId": "principal_id",
  "prompt": "Implement the requested change",
  "status": "queued",
  "presetId": "node-pnpm",
  "containerId": null,
  "branch": "agent/task_uuid",
  "baseCommit": null,
  "resultCommit": null,
  "createdAt": "RFC3339",
  "startedAt": null,
  "finishedAt": null,
  "failure": null,
  "reportId": null
}
```

Statuses: `queued`, `provisioning`, `running`, `waiting_approval`, `validating`, `completed`, `failed`, `cancelled`, `cleanup_pending`.

Allowed transitions:
- queued -> provisioning | cancelled | failed
- provisioning -> running | failed | cancelled | cleanup_pending
- running -> waiting_approval | validating | failed | cancelled | cleanup_pending
- waiting_approval -> running | failed | cancelled
- validating -> completed | failed | cancelled | cleanup_pending
- completed | failed | cancelled -> cleanup_pending

All state transitions are performed by the control plane using compare-and-set/version checks. A worker cannot mark its own task completed without reporting observed validation results. Cancellation is idempotent and must stop the agent and its child process groups before cleanup.

## Create task request

```json
{
  "projectId": "project_uuid",
  "prompt": "Implement the requested change",
  "presetId": "node-pnpm",
  "baseRef": "main",
  "options": {
    "pushBranch": false,
    "keepContainer": false,
    "networkAccess": "restricted"
  }
}
```

The server resolves repository and project metadata from the project record. Never accept a caller-supplied host workspace path as a bind mount.

## Report contract

Report check statuses are `passed`, `failed`, `skipped`, and `not_run`. Include the actual command, exit code, duration, bounded output excerpt, and commit SHA against which it ran. A task with failed required checks cannot be reported as fully successful.
