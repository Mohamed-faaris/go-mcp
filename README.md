# go-mcp

**Container-native autonomous software engineering platform.**

go-mcp accepts a development task, provisions an isolated project container, runs an agent inside that workspace, executes validation, records Git changes, and returns a report.

## Architecture

```text
ChatGPT / API / Web UI
        | HTTPS
 Cloudflare Tunnel
        |
 go-mcp-server (control plane)
        |
 Task Scheduler + Agent Runtime
        |
 Container Manager (Podman)
        |
 Isolated project container
   | project checkout
   | shell + filesystem + processes
   | Git + build + tests
   | task-scoped agent tools
        |
 Report + commit + artifacts
```

The control plane owns the Podman socket and container lifecycle. Workload agents receive broad development access **inside their assigned container only**. They do not receive host root, the Podman socket, or access to other task workspaces.

## Source of truth and multi-agent development

The Git repository is the canonical source for requirements, architecture, interfaces, decisions, implementation, and handoffs. A chat transcript is temporary and is never the only place a requirement or decision may exist.

Before making changes, every coding/web agent must read:
1. `AGENTS.md`
2. `REWORK_PLAN.md`
3. `ARCHITECTURE.md`
4. `docs/agents/COORDINATION.md`
5. `docs/agents/WORKSTREAMS.md`
6. Relevant files under `docs/contracts/`

Use [the web-agent bootstrap prompt](docs/agents/WEB_AGENT_BOOTSTRAP.md) to start an independent agent/chat. Use a dedicated branch and handoff per workstream. The coordination service is planned to persist agent sessions, messages, claims, path leases, and context snapshots so a different web agent can resume after the original chat ends.

Until that service is implemented, use Git branches, PRs, and `docs/agents/handoffs/` as the durable coordination mechanism. Do not assume live agent communication is already available.

## Product capabilities

- Creates an isolated container for each task.
- Supports runtime presets for Node.js + pnpm, Bun, Python + uv, Node.js + npm, Go, and Ubuntu Base.
- Detects project metadata and Git remotes.
- Uses host-managed, repository-scoped GitHub credentials for supported GitHub remotes.
- Runs an autonomous coding agent with terminal, filesystem, process, search, Git, build, and test tools.
- Applies configurable per-tool policies and one-time human approvals for high-impact operations.
- Produces reports with changed files, commits, checks, errors, and artifacts.
- Coordinates multiple agents through persistent project/task messages and versioned context handoffs.

## Repository layout

- Go backend, agent runtime, and task/control-plane services.
- Podman container manager and isolated workload runtime.
- React + TypeScript + Vite frontend using shadcn/ui conventions.
- `docs/agents/`: coding-agent instructions, workstream assignments, and context handoffs.
- `docs/contracts/`: shared cross-workstream API and interface contracts.
- `docs/decisions/`: accepted architectural decisions.

## Development

```bash
go test ./...
go build ./cmd/go-mcp ./cmd/go-mcp-server

cd web
bun install
bun run build
```

See [REWORK_PLAN.md](REWORK_PLAN.md), [ARCHITECTURE.md](ARCHITECTURE.md), and [HANDOFF.md](HANDOFF.md).

## Deployment

Run the control plane on a trusted host with access to the Podman socket. For remote access, route Cloudflare Tunnel to the authenticated control-plane HTTP server. Never expose the Podman socket or workload container ports directly to the public internet.

Do not use sample local credentials from development configuration in a deployed environment. Configure unique secrets and secure authentication before exposing the service.
