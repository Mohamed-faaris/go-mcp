# Container Runtime Contract

## Separation

The trusted control plane owns the container runtime client and Podman socket. Workload containers never receive the socket, runtime API credentials, host root, host namespaces, or sibling workspaces. All agent-facing lifecycle calls go through the task-scoped `container.Manager`; the low-level Podman client is control-plane-only.

## Implemented Go interfaces

The runtime adapter implements `internal/container.Runtime`:

- `Create(ctx, Spec) -> Info`
- `Start(ctx, runtimeID)`
- `Stop(ctx, runtimeID, timeout)`
- `Delete(ctx, runtimeID)`
- `Inspect(ctx, runtimeID)`
- `ListManaged(ctx)`

The manager exposes task-scoped operations:

- `CreateTaskContainer(ctx, CreateRequest)`
- `InspectOwned(ctx, ownerID, taskID, runtimeID)`
- `StartOwned(ctx, ownerID, taskID, runtimeID)`
- `StopOwned(ctx, ownerID, taskID, runtimeID, timeout)`
- `DeleteOwned(ctx, ownerID, taskID, runtimeID)`
- `ListOwned(ctx, ownerID, taskID)`

The runtime interface is not passed into agent/model context. Native tools should use the manager methods or a narrower workspace/process facade.

## Immutable manager metadata

The manager reserves the complete `gomcp.*` label namespace. Caller-supplied labels under this namespace are discarded. Identity labels:

- `gomcp.managed=true`
- `gomcp.owner_id=<principal-id>`
- `gomcp.task_id=<task-id>`
- `gomcp.project_id=<project-id>`
- `gomcp.preset_id=<preset-id>`
- `gomcp.created_at=<RFC3339>`

The registry stores identity independently of Podman labels. Authorization resolves the immutable runtime ID through the registry, verifies owner and task, inspects the live container, and cross-checks all identity labels. A container name prefix is never proof of ownership.

The initial `FileRegistry` uses an atomic JSON replacement and private file permissions. It is a single-control-plane-writer development implementation. Multi-replica deployments must replace it with a transactional shared database.

## Workload spec and enforced defaults

The manager accepts only an allowlisted image and a workspace strictly below a configured dedicated workspace root. Workspace validation resolves symlinks and rejects path escapes, the workspace root itself, sibling paths, and non-directories.

Task workloads are created with:
- one read-write bind mount at `/workspace`
- read-only container root filesystem with writable tmpfs support
- `no_new_privileges=true`
- all Linux capabilities dropped
- memory, CPU and PID limits
- the configured named Podman network
- typed owner/task/project/preset identity labels
- no device mounts or arbitrary caller-supplied host mounts

The runtime adapter intentionally does not forward the legacy `Devices` field or MCP-provided host mount data. The configured network must be a dedicated workload network chosen by the control plane; an empty network setting falls back to Podman's default networking and is not an egress-isolation policy.

The manager validates a task wall-clock timeout, but the scheduler/task context is responsible for enforcing it. Disk quotas are not yet enforced by the Podman adapter. Deployments should use rootless Podman and configure CPU, memory, PID and timeout policies before accepting untrusted code.

## Ownership and approvals

- An agent can inspect/start/stop/delete only the container bound to its task identity.
- An agent may request an additional managed container only through an approval-gated control-plane operation and configured image/quota policy.
- Deleting another task's managed container is not exposed by `DeleteOwned`. A separate approval workflow must bind a one-time approval to the exact runtime ID, action, requester, and argument digest; the manager rechecks the registry and policy immediately before execution.
- Unmanaged containers and protected control-plane containers are hard-denied, not approvable.
- The control plane handles cleanup and workspace removal after the workload has stopped. Registry records are the authorization source; Podman labels alone are insufficient.

## Server integration requirement

The task worker must construct one `container.Manager` during server startup using:
1. the dedicated workspace root (created by `workspace.NewManager`);
2. a durable `container.FileRegistry` or later database implementation;
3. `podman.NewRuntime(existingPodmanClient)`;
4. an image allowlist derived from the shared `internal/presets` catalog;
5. an explicitly configured workload network and resource defaults.

The task worker must use `CreateTaskContainer` and `StartOwned` for provisioning and must use `DeleteOwned` for cleanup. Before calling low-level exec/log methods on a task container, it must verify ownership through `InspectOwned` or use a manager-mediated exec/log method. Legacy admin-created containers remain a separate control-plane feature and are not task-owned workloads.
