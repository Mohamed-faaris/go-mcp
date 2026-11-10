# Container-Native Agent Platform

## Product direction

go-mcp is a container-native autonomous software engineering platform. It receives a task, provisions an isolated workspace, runs an agent against a checked-out project, validates the result, records Git changes, and returns a structured report.

The platform has two execution domains:

- **Control plane:** the trusted go-mcp-server process. It authenticates callers, owns the Podman connection, schedules tasks, records container ownership, streams logs, and collects reports.
- **Workload plane:** short-lived or persistent project containers. They contain the project checkout and development toolchain. The agent has broad access inside this boundary.

The agent must never receive the Podman socket, host filesystem mounts beyond its assigned workspace, host credentials, or control-plane secrets.

## Target architecture

```text
ChatGPT App / API Client / Web UI
             |
          HTTPS
             |
      Cloudflare Tunnel
             |
      go-mcp-server
      (Control Plane)
       |     |      |
  Auth/API  Queue  Registry
       |     |      |
       +-- Task Orchestrator
                 |
           Agent Runtime
                 |
          Container Manager
                 |
              Podman
                 |
       Isolated Task Container
       - /workspace/project
       - Agent runtime + tools
       - Git
       - Build/test toolchains
       - Task-scoped credentials
                 |
       Result collector / report
```

Cloudflare Tunnel terminates external ingress at the control plane only. The Podman socket and workload containers are never directly exposed to the internet. Container ports are private by default. When a task explicitly needs a preview URL, the control plane creates a scoped route/proxy to that workload and removes it when the task ends.

## Core entities

### Task

A task is the unit of work and owns its execution resources.

Suggested fields:

- `id`, `owner_id`, `project_id`
- `prompt`, `status`, `created_at`, `started_at`, `finished_at`
- `container_id`, `workspace_path`
- `base_commit`, `result_commit`, `branch_name`
- `exit_reason`, `report_id`

Statuses: `queued`, `provisioning`, `running`, `validating`, `completed`, `failed`, `cancelled`, `cleanup_pending`.

### Managed container

Every managed container receives immutable labels:

- `gomcp.managed=true`
- `gomcp.owner_id=<principal>`
- `gomcp.task_id=<task-id>`
- `gomcp.project_id=<project-id>`
- `gomcp.created_at=<RFC3339 timestamp>`

The manager records the Podman container ID, creator, task, creation time, and lifecycle state in its registry. A container is deletable by the agent only when the registry confirms that it was created by that agent/task. Prefix matching is not ownership verification.

### Project

A project stores a repository URL, default branch, optional environment template, and credential reference. Each task receives an isolated clone or worktree. Do not mount a developer's live working directory read-write into an autonomous task container.

### Report

A report contains task summary, changed files, commit(s), branch, tests and commands executed, exit codes, logs/artifacts, warnings, and cleanup outcome. Reports must distinguish passed, failed, skipped, and not-run checks.

## Agent responsibilities

The agent is a software engineering worker, not a general desktop operator. It should:

1. Inspect the repository and identify language, package manager, build system, and test commands.
2. Create or use a task branch and establish the base commit.
3. Plan work and make changes in the task workspace.
4. Run formatting, linting, type checks, builds, and tests that apply.
5. Inspect the final diff and check for accidental secrets or generated files.
6. Commit successful work with a meaningful message when Git authoring is enabled.
7. Return a machine-readable result and human-readable report.
8. Leave the container running only when requested; otherwise clean up after artifacts and logs are collected.

The agent may use any tools installed in its workload image, but tool execution is scoped to the container. The initial tool set should be structured wrappers around shell, filesystem, process, search, Git, and test operations. MCP can remain as an adapter for optional tools, not as the foundation of workspace access.

## Container lifecycle and ownership

The control plane exposes create, inspect, start, stop, restart, exec, logs, and delete operations. Agent-facing lifecycle tools must be task-scoped:

- Create: allowed within configured image, CPU, memory, disk, and concurrency limits. The manager assigns owner/task labels.
- List/inspect: return only containers visible to the caller or task.
- Exec/files/processes: operate only on the current task container.
- Delete: require registry ownership by the current agent/task and reject control-plane or foreign containers.
- Cleanup: manager can clean up any managed container according to retention policy; workload agents cannot bypass ownership checks.

Never trust a caller-supplied container name as proof of ownership. Resolve IDs through the registry and verify labels against registry records. Use random IDs, not user input, for container names.

## Tool surface

Expose a stable task-scoped tool API inspired by Remote Desktop Commander, but implemented against the container runtime:

### Container and task tools
- `create_task_container`
- `get_task_container`
- `list_my_task_containers`
- `start_my_task_container`
- `stop_my_task_container`
- `delete_my_task_container`
- `get_task_status`
- `cancel_task`

### Filesystem tools
- `read_file`
- `read_multiple_files`
- `write_file`
- `create_directory`
- `list_directory`
- `move_file`
- `get_file_info`
- `edit_block`
- `search_files`

All paths are resolved beneath the assigned workspace. Reject traversal, symlink escapes, and access outside the workspace.

### Terminal and process tools
- `run_command`
- `start_process`
- `read_process_output`
- `interact_with_process`
- `terminate_process_session`
- `list_process_sessions`
- `list_processes`
- `kill_process`

Commands run in the task container with a deadline, output limit, environment allowlist, and process-group cleanup. Long-running commands return a session ID and are polled separately.

### Git and validation tools
- `git_status`
- `git_diff`
- `git_log`
- `git_create_branch`
- `git_commit`
- `detect_project_commands`
- `run_tests`
- `run_build`
- `collect_task_report`

Git credentials should be short-lived and scoped to the target repository. Never expose credentials in command output, logs, environment dumps, or model context.

## HTTP API

Version all public endpoints under `/api/v1`. Initial resource shape:

- `POST /api/v1/tasks` create and queue a task
- `GET /api/v1/tasks/{taskID}` task status and summary
- `POST /api/v1/tasks/{taskID}/cancel` cancel a task
- `GET /api/v1/tasks/{taskID}/events` task event stream
- `GET /api/v1/tasks/{taskID}/report` final report
- `GET /api/v1/containers` list caller-owned managed containers
- `POST /api/v1/containers/{id}/stop`
- `DELETE /api/v1/containers/{id}`
- `GET /api/v1/containers/{id}/logs`

Use authenticated principals and per-resource authorization. Do not use shared default passwords or browser-stored base64 credentials. Configure secure sessions or bearer tokens, CSRF protection for cookie auth, rate limits, request size limits, and audit events.

## Security model

"Full access" means unrestricted development operations inside the assigned workload container, subject to resource quotas and task timeouts. It does not mean privileged host access.

- Run rootless Podman where possible; otherwise use a dedicated unprivileged workload user.
- Do not use `--privileged`, host PID/network namespaces, host device mounts, or the Podman socket in workload containers by default.
- Mount only the task workspace and explicitly approved caches.
- Use read-only root filesystem where practical, drop Linux capabilities, enable seccomp, and set CPU, memory, PID, disk, and wall-clock limits.
- Use egress controls where possible; deny access to cloud metadata endpoints and control-plane internals.
- Never place API keys or Git tokens in the project checkout. Inject short-lived task-scoped secrets only when required.
- Treat repository files, package scripts, build output, and model-generated commands as untrusted.
- Record destructive operations and ownership decisions in an audit log.
- The control plane must not expose arbitrary Podman API access to the agent.

## External access and previews

Cloudflare Tunnel routes to the authenticated control-plane API/UI. It is not a tunnel to the Podman socket. Workload containers are private by default. A preview service should be a reverse proxy managed by the control plane with per-task random hostnames, allowlisted ports, expiration, and explicit cleanup. Do not allow an agent to publish arbitrary host ports directly.

## Execution flow

1. Authenticate the caller and validate repository access.
2. Create a task record and enqueue it.
3. Allocate a unique workspace and clone the repository.
4. Record the base commit and create a task branch.
5. Create a resource-limited workload container with ownership labels.
6. Start the agent with task-scoped tools and a bounded context.
7. Stream structured task events and logs to the control plane.
8. Run applicable validation commands and capture exit codes.
9. Review the diff, commit changes, and collect artifacts.
10. Generate the report.
11. Stop/delete the container according to retention policy.
12. Mark the task complete or failed; retain audit and report records.

## Suggested package boundaries

Keep the Go backend and React frontend. Refactor toward these packages:

- `internal/auth`: principal/session authentication and authorization
- `internal/task`: task model, state machine, orchestration
- `internal/scheduler`: queue, concurrency, cancellation, retries
- `internal/container`: runtime interface and lifecycle
- `internal/podman`: Podman implementation of the runtime interface
- `internal/workspace`: safe clone, checkout, workspace management
- `internal/worker`: agent execution within a workload container
- `internal/tools`: task-scoped filesystem, process, Git, and validation tools
- `internal/report`: result and artifact collection
- `internal/events`: task event stream
- `internal/server`: versioned HTTP API and WebSocket/SSE handlers
- `internal/audit`: append-only security and lifecycle events

The UI should center on Projects, Tasks, Task Detail, Containers, Images, and Settings. A task detail view should show plan, live logs, changed files, Git status, validation, and final report.

## Delivery phases

1. **Safety foundation:** runtime interface, ownership registry, secure authentication, safe workspace isolation, no host socket in workloads.
2. **Task lifecycle:** task state machine, queue, create/cancel/retry, container provisioning and cleanup.
3. **Agent worker:** container-scoped tool executor, command sessions, project detection, bounded agent loop.
4. **Git and validation:** branch management, diff review, commits, tests/builds, artifact collection.
5. **Reports and UI:** task timeline, live logs, diff viewer, validation matrix, report export.
6. **External access:** Cloudflare Tunnel deployment guide, authenticated ingress, scoped preview proxy.
7. **Hardening:** integration tests, fault injection, quotas, audit, retention, backup and recovery.

## Acceptance criteria

- A task cannot access another task's workspace or container.
- An agent cannot delete a container it did not create.
- An agent container cannot access the Podman socket or host root.
- A task can clone a repository, modify files, run checks, create a commit, and return a report.
- Failed tests are reported as failures, not hidden by a successful agent response.
- Cancellation stops the agent and all child processes.
- Container cleanup is idempotent and ownership-checked.
- External HTTPS access reaches only the authenticated control plane.
- No default production credentials are shipped.
- Unit and integration tests cover authorization, path traversal, ownership, lifecycle, and task state transitions.
