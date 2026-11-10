# go-mcp — Container-Native Agent Rework Handoff

## Product direction

Rework this repository into a container-native autonomous software engineering platform. Read `ARCHITECTURE.md` before implementation. The target is not a general Ubuntu desktop-control agent. The target is an agent that works on a checked-out project inside an isolated task container, runs checks, commits changes, and produces a report.

## Existing foundations to retain

- Go backend and existing Podman client.
- Existing container lifecycle and image management where correct.
- React + TypeScript + Vite dashboard.
- OpenAI-compatible LLM client and agent loop where reusable.
- Structured logging and MCP adapter where useful.

Do not preserve unsafe behavior merely for compatibility. Audit all existing routes and lifecycle code before exposing them to agents.

## Required architecture

- The go-mcp-server process is the trusted control plane and the only component allowed to access the Podman socket.
- Each task receives a unique workspace and isolated workload container.
- The project is cloned into the workspace; do not mount the developer's live working tree read-write by default.
- Agent tools operate only inside the assigned workspace/container.
- Agent-created containers are registered with owner/task metadata. An agent can inspect or delete only containers owned by its task identity.
- Global cleanup remains a manager-only operation.
- The agent can run shell commands, manage processes, edit/search files, use Git, run tests/builds, and collect artifacts.
- The agent commits successful work to a task branch and returns a structured report.
- Cloudflare Tunnel routes HTTPS to the authenticated control plane only. Never expose Podman API or raw container ports publicly.

## Security requirements

"Full access" means broad development access inside the workload container, not host access. Do not mount the Podman socket into workload containers. Do not use privileged containers, host namespaces, or host devices by default. Enforce CPU, memory, PID, disk, and wall-clock limits. Restrict filesystem paths to the task workspace. Use task-scoped, short-lived Git credentials. Verify ownership from the manager registry, not container name prefixes or caller-supplied labels alone.

Replace default production credentials and base64 localStorage auth with secure authentication and per-resource authorization before public deployment.

## Required task lifecycle

1. Authenticate and authorize the caller.
2. Create a task record and enqueue it.
3. Clone the repository into a unique workspace; record base commit and task branch.
4. Provision a resource-limited workload container with immutable owner/task labels.
5. Run the agent with task-scoped tools and cancellation support.
6. Capture logs, process exit codes, changed files, Git diff, and artifacts.
7. Run applicable format/lint/typecheck/build/test commands.
8. Review the diff and commit changes on the task branch.
9. Generate a report with passed/failed/skipped/not-run checks.
10. Stop and clean up the container according to retention policy.

## Implementation order

1. Add a container runtime interface and harden Podman ownership/lifecycle behavior.
2. Add task models, task state machine, scheduler, cancellation, and durable registry.
3. Add safe project workspace cloning and task-scoped agent tool execution.
4. Add Git branch/commit and test/build orchestration.
5. Add structured reports, task events, and dashboard task views.
6. Add authenticated versioned API and Cloudflare Tunnel deployment configuration.
7. Add unit/integration tests for path traversal, authorization, ownership, cancellation, cleanup, and task transitions.

## Acceptance criteria

- Agent cannot access host root, Podman socket, another task's workspace, or another task's container.
- Agent cannot delete containers outside its ownership scope.
- Task can clone, edit, test, commit, and report.
- Failed checks remain visible in the report.
- Cancellation kills child processes and cleans up predictably.
- Public ingress reaches only authenticated control-plane routes.
- Tests cover the security boundary and lifecycle failure cases.

## Current repository notes

The current repository already has a Podman client, a container dashboard, and a separate agent loop that uses configured MCP servers. The rework must integrate these into one task lifecycle. Do not add a second independent container manager or leave the old desktop-agent path as the default execution route.
