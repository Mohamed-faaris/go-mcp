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

## What it does

- Creates an isolated container for each task.
- Clones a Git repository into a task workspace.
- Runs an autonomous coding agent with terminal, filesystem, process, search, and Git tools.
- Runs formatting, lint, type-check, build, and test commands.
- Creates task branches and commits completed work.
- Streams task status and logs to the dashboard.
- Produces a report with changed files, commits, checks, errors, and artifacts.
- Deletes only task-owned containers from the agent tool surface; the manager handles global cleanup.

## Repository

- Go backend and agent runtime.
- Podman container manager.
- React + TypeScript + Vite dashboard.
- MCP remains an optional adapter for external tools, not the foundation of project filesystem and terminal access.

## Development

```bash
go test ./...
go build ./cmd/go-mcp ./cmd/go-mcp-server

cd web
bun install
bun run build
```

See [ARCHITECTURE.md](ARCHITECTURE.md) for the target design, security boundaries, task lifecycle, API, tool surface, package boundaries, and acceptance criteria.

## Deployment

Run the control plane on a trusted host with access to the Podman socket. For remote access, route Cloudflare Tunnel to the authenticated control-plane HTTP server. Never expose the Podman socket or workload container ports directly to the public internet.

Do not use the sample local credentials from development configuration in a deployed environment. Configure unique secrets and secure authentication before exposing the service.
