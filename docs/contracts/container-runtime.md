# Container Runtime Contract

## Separation

The trusted control plane owns the container runtime client and Podman socket. Workload containers never receive the socket, runtime API credentials, host root, or sibling workspaces.

## Runtime interface

The Go interface should expose:
- Create(spec) -> immutable runtime ID and observed metadata
- Start(id)
- Stop(id, timeout)
- Delete(id)
- Inspect(id)
- ListManaged(filters)
- Exec(id, command, deadline, outputLimit)
- Attach/stream logs and process output

The public agent tool layer must call a task-scoped service, never the runtime client directly.

## Immutable manager metadata

Labels:
- `gomcp.managed=true`
- `gomcp.owner_id=<principal-id>`
- `gomcp.task_id=<task-id>`
- `gomcp.project_id=<project-id>`
- `gomcp.preset_id=<preset-id>`
- `gomcp.created_at=<RFC3339>`

The registry stores these values independently of Podman labels. Authorization uses registry records plus runtime ID, then cross-checks labels. Labels alone are not authorization.

## Workload spec

Must include allowlisted image, isolated workspace mount, task-scoped environment, CPU/memory/PID/disk/time quotas, network policy, working directory, and security options. No privileged mode, host namespaces, devices, or Podman socket by default.

## Ownership rules

- An agent can inspect/exec/stop/delete only the container bound to its task identity.
- An agent may request an additional managed container. This is an approval-gated manager operation and creates a new owner/task registry record.
- Deleting another task's container requires a one-time approval bound to that exact target ID and reason. The manager rechecks policy and ownership at execution time.
- Unmanaged containers and protected control-plane containers are hard-denied, not approvable by agent request.
- Cleanup is idempotent and manager-controlled.
