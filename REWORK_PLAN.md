# go-mcp Rework Plan

**Status:** Architecture and workstream plan  
**Implementation branch:** `feat/container-native-agent-rework`  
**Canonical repository:** `Mohamed-faaris/go-mcp`

This document is the source of truth for product scope and implementation priorities. Every coding agent must read `AGENTS.md` first and then this plan.

## 1. Product goal

Rebuild go-mcp as a container-native autonomous software engineering platform. A user selects a repository or local project folder, enters a task, and starts an agent. The control plane provisions an isolated environment, supplies scoped tools and credentials, monitors the work, runs validations, collects commits and artifacts, and produces a report.

The agent is a software engineering worker. It is not a general-purpose remote desktop controller. Desktop/browser automation can be added later as an optional capability, but it is not the core development execution model.

## 2. Decisions and alternatives

| Area | Selected direction | Alternative / trade-off |
|---|---|---|
| Execution unit | Isolated container per task, with optional explicit retention | Persistent per-project container can be faster but risks cross-task contamination |
| Container runtime | Podman behind a Go runtime interface | Docker can be added as another adapter later |
| Project workspace | Unique clone or snapshot per task | Direct read-write mount of a developer's live folder is not the default |
| GitHub authentication | Host-side GitHub App installation-token broker | Host `gh auth token` is development-only and may expose broad personal permissions |
| Git credential delivery | Prefer manager-side Git proxy/credential broker so raw tokens never enter agent context | Short-lived askpass helper is a fallback if proxy support is impractical |
| Agent tools | Native scoped filesystem/process/Git/test tools | MCP remains an optional adapter, not the foundation |
| Permission model | Per-tool policy plus one-time approval requests | One global dangerous/safe switch is too coarse |
| UI | Task-first React application using shadcn/ui patterns and reusable local components | Existing sandbox manager screens remain available under runtime/admin views |
| Theme | Monochrome black, white, and neutral greys | Keep semantic status contrast without decorative color |
| Public access | Cloudflare Tunnel to authenticated control plane | Never publish Podman socket or direct workload ports |

## 3. Main architecture

```text
Web UI / ChatGPT App / API Client
                 |
                HTTPS
                 |
          Cloudflare Tunnel
                 |
       go-mcp-server (Control Plane)
          |       |          |
        Auth   Task API   Event Store
          |       |          |
          +--- Scheduler / Agent Orchestrator
                         |
                 Container Manager
                         |
                       Podman
                         |
            Isolated Task Workload
            - project snapshot/clone
            - runtime preset
            - agent and task tools
            - Git / build / tests
                         |
              logs / results / report
```

The control plane is the only process with Podman socket access. Agents run inside workload containers and have broad development access only within their own container and workspace. The manager owns task/container identity, approvals, GitHub credentials, and cleanup.

## 4. Runtime presets

Presets are data-driven environment definitions containing an ID, name, description, OCI image, supported tools, pinned versions, setup steps, resource defaults, and project-detection hints. Keep preset details behind a collapsible environment selector in the task form.

Initial presets:

| ID | Display name | Includes / purpose |
|---|---|---|
| `node-pnpm` | Node.js + pnpm | Node LTS, Corepack, pnpm, Git, common build tools |
| `bun` | Bun | Bun runtime/package manager, Git, common build tools |
| `python-uv` | Python + uv | Python 3.12, uv, Git, common build tools |
| `node-npm` | Node.js + npm | Node LTS, npm, Git |
| `go` | Go | Go toolchain, Git |
| `ubuntu-base` | Ubuntu Base | Ubuntu LTS, Git, shell utilities |

Use pinned versions in production images. Build and publish reusable OCI images in CI rather than reinstalling large toolchains for every task. The agent may install project dependencies inside its container. Do not silently execute arbitrary install scripts on the host.

### Project detection

Inspect repository metadata and lockfiles to recommend a preset:
- `pnpm-lock.yaml` => Node.js + pnpm
- `bun.lock` / `bun.lockb` => Bun
- `package-lock.json` => Node.js + npm
- `uv.lock` or `pyproject.toml` => Python + uv
- `go.mod` => Go
- otherwise => Ubuntu Base or explicit user selection

Detection is advisory. Explicit user selection wins. Multiple project types in a monorepo should be represented as a monorepo with workspace-aware commands, not by guessing one runtime.

## 5. Repository and Git behavior

### Existing remote

When a selected folder contains a Git remote:
1. Read `git remote -v` and identify the provider and canonical repository URL.
2. Never assume every remote is GitHub. Validate the host before requesting credentials.
3. For GitHub remotes, ask the host-side GitHub App credential broker for an installation token restricted to that repository and the minimum required permissions.
4. Clone/snapshot into a unique task workspace and record the base commit.
5. Create a task branch.
6. Run local Git operations inside the container.
7. Push through the control-plane credential broker or scoped Git proxy. Do not place a long-lived token in the repository, agent prompt, logs, or command arguments.
8. Revoke/discard task credential material at completion; GitHub installation tokens expire after a short lifetime.

A local folder may contain uncommitted changes. Detect this explicitly. Let the user choose whether the task should use a clean clone from the remote or a snapshot that includes local changes. Never discard local edits silently.

### No remote

- If a folder is a Git repository without a remote, retain Git metadata where safe and allow local branches/commits.
- If it is not a Git repository, snapshot it and initialize Git only when user/project policy requests it.
- Do not pretend a local commit has been pushed.
- For non-GitHub providers, use a separate credential adapter. Never send GitHub tokens to arbitrary remotes.

### Credential ownership

GitHub authentication belongs to the Container Manager. The agent only receives a restricted Git operation capability for the assigned repository/task. Prefer a host-side proxy that injects the installation token into Git transport, keeping the raw token out of the workload. If a temporary credential helper is used, keep it outside the checkout, use restrictive permissions, redact logs, and remove it on cleanup.

## 6. Agent tools and permission policy

Tool policies are configurable per tool through the UI and API:
- `automatic`: executes within the explicitly assigned task/container scope.
- `approval_required`: create an approval request for the exact action; execution pauses until an authorized human approves it.
- `disabled`: tool is not offered to the agent and invocation is rejected.

Each request is single-use, short-lived, bound to immutable resource IDs and a hash of exact arguments. An approval cannot be replayed for modified arguments.

### Default policy examples

| Tool/action | Default policy | Boundary |
|---|---|---|
| Read/write/search project files | Automatic | Assigned workspace only |
| Run shell commands | Automatic | Assigned workload container only |
| Process/session management | Automatic | Assigned workload container only |
| Git status/diff/log/branch/local commit | Automatic | Assigned repository checkout |
| Push branch to assigned GitHub repo | Approval required initially | Task-scoped GitHub credential |
| Create an additional managed container | Approval required | Quota and allowlisted image |
| Stop/delete own task container | Automatic at task cleanup | Exact owner/task match |
| Delete a different task's container | Approval required | Manager mediated; managed containers only |
| Delete unmanaged/host container | Disabled | Never available to agent |
| Host root, host file deletion, Podman socket in workload | Disabled | Hard security deny |
| Change global tool/security settings | Approval required | Admin only |

"Approval is enough" means the agent may request a one-time approval for an otherwise permitted high-impact action. It does not mean every operation is approvable. Protected resources and host escape operations remain hard-denied.

### Native tool catalog

- Task: create/list/inspect/cancel task, task status and report.
- Container: inspect/list own task containers, start/stop/delete own container, request another managed container.
- Filesystem: read/write/list/move/info/edit/search inside workspace.
- Terminal: run bounded command, start/read/interact/terminate process session, list processes.
- Git: status/diff/log/create branch/commit/push through broker.
- Validation: detect project commands, formatter/lint/typecheck/build/test.
- Coordination: register/resume agent session, read project context, send/receive messages, publish artifacts, create handoff, claim/release work, publish progress.
- Approval: request, list pending, approve/deny, inspect history.

All tools are task-scoped by default. MCP server definitions remain a separate optional integration category.

## 7. Multi-agent communication and context continuity

Web agents may operate in different browser tabs, separate ChatGPT conversations, or different web-agent sessions. They must not depend on one chat transcript remaining alive. A shared coordinator API and persistent project context make handoff possible.

### What is canonical

- Git repository: canonical source for code, specifications, architecture, interface contracts, accepted decisions, workstream assignments, and committed handoff documents.
- Control-plane database/event log: durable runtime state for tasks, agent sessions, messages, work claims, locks, progress, approvals, artifacts, and context snapshots.
- A chat transcript is not canonical and must never be the only place a requirement or decision exists.

### Session lifecycle

1. An agent registers a session with project ID, task/workstream ID, branch, current commit, and capability metadata.
2. The coordinator returns a session ID, channel IDs, latest project context version, current claims, and a context bundle.
3. The agent reads repository source-of-truth files and the current workstream handoff.
4. The agent posts progress and messages to the shared project/task channel.
5. At chat/task end, the agent creates a context snapshot/handoff.
6. A new agent session resumes by requesting the latest context bundle. It receives the summary, decisions, relevant messages, open tasks, branch/commit, changed files, tests, blockers, and repo paths.
7. The new agent verifies the repo state and context version before resuming. If the branch moved, it refreshes and reconciles rather than blindly applying stale instructions.

### Context bundle format

A context snapshot contains:
- project ID and task/workstream ID
- repository URL, base branch, work branch, latest commit SHA
- current goal and concise progress summary
- accepted decisions with links to repository decision records
- non-negotiable architecture and security constraints
- assigned/blocked/completed work items
- files changed and their owners
- recent messages relevant to this task
- tests/checks with observed results and commit SHA
- open questions, blockers, risks, and exact next actions
- previous snapshot ID and creation timestamp

Snapshots are versioned. Use recent relevant messages plus a compact summary, not an unbounded transcript dump. Context must be tenant/project scoped and must not mix unrelated projects.

### Agent communication tools

Provide these stable agent-facing actions:
- `register_agent_session`: create a session and join project/task channels.
- `resume_agent_session`: restore the latest session/context after a chat restart.
- `get_project_context`: retrieve repo metadata, current decisions, active work, and context bundle.
- `list_project_agents`: show active/idle agents and workstream ownership.
- `send_agent_message`: post a message to project/task/workstream channel or a specific agent.
- `read_agent_messages`: read messages after a cursor.
- `publish_agent_progress`: send progress, blocker, or completion event.
- `publish_agent_artifact`: attach a report, diff summary, test output, or design contract.
- `create_agent_handoff`: write a versioned context snapshot and optional repository handoff file.
- `claim_work_item` / `release_work_item`: coordinate ownership using expiring leases.
- `acquire_path_lock` / `renew_path_lock` / `release_path_lock`: avoid concurrent edits to shared files.
- `request_agent_review`: ask another agent to review a commit, diff, or artifact.

### Proposed API resources

Version routes under `/api/v1`:
- `POST /agent-sessions` register a web agent session.
- `POST /agent-sessions/{id}/resume` resume with latest context bundle.
- `GET /projects/{projectID}/context` get canonical context.
- `GET /projects/{projectID}/agents` list sessions/agents.
- `POST /projects/{projectID}/messages` publish a message.
- `GET /projects/{projectID}/messages?after=<cursor>` read messages.
- `POST /work-items/{id}/claim` claim work with lease.
- `POST /work-items/{id}/release` release work.
- `POST /locks`, `POST /locks/{id}/renew`, `DELETE /locks/{id}` manage path locks.
- `POST /agent-sessions/{id}/handoffs` store a handoff snapshot.
- `GET /agent-sessions/{id}/context` return the context bundle.
- `GET /projects/{projectID}/events` SSE/WebSocket project event stream.

### Coordination rules

- Every message has project ID, task/workstream ID, sender agent/session ID, recipient/channel, timestamp, and correlation ID.
- Messages are persisted and cursor-paginated. WebSocket/SSE is delivery acceleration, not the storage layer.
- Work claims and path locks use leases and expire when an agent disappears.
- Claiming a task does not grant filesystem access to another task's container.
- Different workstreams use separate branches/worktrees and merge through a designated integrator.
- Concurrent agents must not edit the same path. A lock conflict requires coordination, not force override.
- Accepted architectural decisions are written to `docs/decisions/` and committed.
- Context restoration always validates the current commit/branch and checks whether a newer handoff exists.

## 8. UI plan

Use React, TypeScript, Vite, Tailwind, and shadcn/ui components/blocks. Build reusable local components using the shadcn conventions. Use neutral design tokens and accessible Radix-based primitives. The UI should feel like a focused developer tool, not a dense operations HUD.

### Visual system

- Monochrome black, white, and neutral greys.
- Use light and dark neutral surfaces with clear hierarchy.
- Use color only for semantic status when contrast requires it; no decorative gradients or glowing cards.
- Typography should be compact and readable. Consistent spacing, borders, focus rings, keyboard navigation.
- Prefer shadcn patterns: Button, Card, Badge, Dialog, AlertDialog, DropdownMenu, Sheet, Tabs, Table, Accordion, Collapsible, Form, Input, Select, Textarea, Tooltip, Separator, Skeleton, and Sonner.
- Keep shadcn component source in the repository; do not require an external runtime dependency to shadcn registry.

### Navigation

- Overview
- Tasks
- Projects
- Containers
- Approvals
- Tools
- Settings

### Overview dashboard

Only show task-level essentials:
- running, queued, waiting approval, completed counts
- recent tasks and task activity
- primary "New task" action
- useful empty state

Do not put every operational option on the dashboard. Hide advanced details behind collapsibles, dialogs, detail pages, and admin-only settings. Host devices, image management, preset customization, runtime limits, and raw container diagnostics are not dashboard widgets.

### New task dialog/page

Always-visible essentials: project/repository picker, task prompt, detected/selected preset, submit action.

Collapsible "Advanced": branch/base ref, CPU/memory/disk/time limits, environment variables or secret references, optional MCP integrations, network/preview access, retention policy, approval policy overrides. Sensitive values must be write-only or represented by secret references.

Use dialogs for custom preset editing, confirmation, repository integration, and destructive operations.

### Task detail

Tabs: Overview/Plan, Activity, Changes, Checks, Commits, Report. Show assigned agent sessions and handoffs in an Agents panel. Approval requests appear inline. Terminal/logs and container metadata are detail views, not dashboard clutter.

### Tools and permissions

The Tools page contains a searchable catalog with categories, description, scope, policy, enabled state, and source (native or MCP). Editing a tool or changing its policy opens a dialog. Approval inbox shows exact requested action, target resource, arguments summary, requesting agent/task, expiry, and approve/deny controls.

### Agent collaboration UI

Project detail includes an Agents & Activity view:
- active agent sessions and last heartbeat
- workstream/task claim
- branch and latest commit
- last published progress
- handoff history and resume action
- project/task message stream
- file locks/conflict indicators

The UI should allow an operator to start another web agent with a generated context bundle/bootstrap prompt. A new chat can resume using the context snapshot ID or task/project ID; it must not rely on the old chat being accessible.

## 9. Workstream delivery sequence

1. Source-of-truth docs and contracts.
2. Runtime abstraction, ownership registry, container hardening.
3. Presets, workspace bootstrap, project detection, Git provider interface.
4. GitHub App credential broker and Git transport.
5. Task model, scheduler, worker lifecycle, cancellation.
6. Native task-scoped tools and approval engine.
7. Persistent agent sessions, messaging, claims, locks, handoff snapshots.
8. React/shadcn UI redesign and approval/agent collaboration screens.
9. Integration, security tests, deployment, Cloudflare Tunnel and preview routing.

Parallelization is permitted only along the path ownership defined in `docs/agents/WORKSTREAMS.md`.

## 10. Acceptance criteria

- A task can select Node+pnpm, Bun, Python+uv, Node+npm, Go, or Ubuntu Base.
- A repo with a GitHub remote can receive a repository-scoped task credential from the host manager; raw token is not exposed in agent prompt/logs.
- A local folder without remote still supports local work and commits.
- An agent can run tools in its task workspace and cannot reach host files or another task container.
- Deleting another task's container requires a one-time approval and is manager-mediated; deleting unmanaged or protected containers is impossible.
- Tool policies can be changed in the UI and are enforced server-side.
- Two browser agents can exchange persisted messages and coordinate claims/locks.
- A new chat can resume from a persisted context bundle and current repository state.
- Dashboard is task-first, monochrome, and not overloaded.
- Advanced settings use collapsibles and dialogs.
- Reports accurately distinguish passed, failed, skipped, and not-run checks.
- Public HTTPS ingress reaches only the authenticated control plane.
