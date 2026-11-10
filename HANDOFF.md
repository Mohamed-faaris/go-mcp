# go-mcp — Container-Native Agent Rework Handoff

## Source of truth

The repository is the specification and implementation source of truth. All agents must read `AGENTS.md`, `REWORK_PLAN.md`, `ARCHITECTURE.md`, `docs/agents/COORDINATION.md`, and `docs/agents/WORKSTREAMS.md` before coding.

Use `docs/agents/WEB_AGENT_BOOTSTRAP.md` to start another web agent or new chat. The coordination service is planned; until implemented, Git branches, PRs, and files under `docs/agents/handoffs/` are the durable communication/handoff mechanism. Do not claim live messaging/session resume exists before it is implemented.

## Product direction

Rework this repository into a container-native autonomous software engineering platform. The target agent works on a checked-out project in an isolated task container, runs checks, creates commits, and produces a report. It is not primarily a desktop-control agent.

## Existing foundations to retain

- Go backend and Podman client, subject to security hardening.
- Existing container lifecycle and image management where correct.
- React + TypeScript + Vite dashboard, redesigned around tasks.
- OpenAI-compatible LLM client and agent loop where reusable.
- MCP adapter as an optional integration, not the core filesystem/terminal interface.

## Required capabilities

- Runtime presets: Node.js + pnpm, Bun, Python + uv, Node.js + npm, Go, Ubuntu Base.
- Repository detection and task workspaces.
- Existing Git remote detection. GitHub remotes use host-brokered, temporary repository-scoped GitHub App tokens. Local-only repositories continue to work without remote credentials.
- Container-owned tools for files, shell, processes, search, Git, tests, builds, and reports.
- UI-configurable tool policies: automatic, approval required, disabled.
- One-time approvals for permitted high-impact operations. Deleting another managed task's container requires manager-mediated approval. Unmanaged/host containers and host escape operations remain hard-denied.
- Persistent agent sessions, project/task messaging, work claims, path leases, and versioned context snapshots so a new web agent/chat can resume work.
- Task-first React/shadcn UI, monochrome black-and-white theme, advanced settings in collapsibles/dialogs.
- Authenticated API ingress through Cloudflare Tunnel. Only the control plane is publicly reachable.

## Security constraints

The control plane alone accesses the Podman socket. Workload agents have broad development access inside their assigned container only. Never expose host root, the Podman socket, or another task's workspace. Do not pass permanent credentials into a workload. Verify ownership from the manager registry rather than container names.

## Required task lifecycle

1. Authenticate/authorize.
2. Create and queue task.
3. Clone or snapshot project and record base commit.
4. Select runtime preset and provision isolated container.
5. Issue scoped task tools and repository credentials if needed.
6. Run agent with cancellation, event streaming, and bounded resource/time limits.
7. Run applicable formatting/lint/typecheck/build/test commands.
8. Review diff, commit changes, and push only with allowed credentials/policy.
9. Generate structured report.
10. Clean up container/credentials according to policy.

## Multi-agent implementation

Parallel workstreams are defined in `docs/agents/WORKSTREAMS.md`. Each agent owns separate paths and a separate branch. Shared contracts live in `docs/contracts/`. The integration coordinator owns cross-workstream merges and full acceptance. Every chat/session handoff records branch, base/latest commit, changed paths, checks, blockers, and next actions.

## Completion criteria

- Tasks can run in the selected runtime presets.
- Existing Git remotes are handled safely with scoped host-managed credentials.
- Agent tools are scoped to the current task and enforced server-side.
- Approval-required tools pause until explicit one-time authorization.
- Multi-agent messages and context snapshots persist across chats once the coordination service is implemented.
- Another agent can resume from a committed handoff and current repository state even before the live coordination service exists.
- Dashboard is monochrome, task-first, and hides advanced settings behind collapsibles/dialogs.
- Reports distinguish passed, failed, skipped, and not-run checks.
