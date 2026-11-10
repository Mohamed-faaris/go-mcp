# go-mcp Web UI & Podman Sandbox Architecture — Agent Handoff

> **CRITICAL DIRECTIVE**: Read this entire document before modifying code.
> This codebase implements an autonomous Ubuntu computer control agent via MCP in Go (`go-mcp`)
> and its Web UI container manager (`go-mcp-server`) that spins up, views, duplicates, and manages
> sub-containers running Podman with bundled MCP tools.

---

## 1. Project Context & Current State

- **Repository**: `github.com/Mohamed-faaris/go-mcp`
- **Location**: `/home/mfk01/Projects/econ-systems/tools/go-mcp`
- **Language**: Go 1.22+ (Backend) & TypeScript / React + Vite + Bun + Tailwind/shadcn (Frontend in `web/`)
- **Key Philosophy**: Clean Go standard library patterns, minimal external dependencies (`gopkg.in/yaml.v3`, `nhooyr.io/websocket`).

### Core Existing Packages (CLI Agent - Completed & Tested)
- `internal/config`: Loads `.env`, validates runtime configuration.
- `internal/logger`: Structured JSON logging to stderr and log file.
- `internal/mcp`: JSON-RPC 2.0 stdio MCP client & tool registry with dynamic discovery.
- `internal/llm`: OpenAI-compatible API client with function/tool calling.
- `internal/context`: Context manager preserving system prompt and trimming old messages.
- `internal/permission`: Interactive approval modes (`all`, `dangerous`, `none`) & shell blocking.
- `internal/screenshot`: Screenshot capture via connected MCP servers.
- `internal/agent`: Autonomous execution loop.
- `cmd/go-mcp`: CLI entry point (`./go-mcp "task"`, `./go-mcp --check`).

---

## 2. Podman Container Management Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Host Machine                                                │
│  - Projects Folder: /home/user/projects (bind mount)        │
│  - Optional scratch: /home/user/tmp-podmans                 │
│  - Podman socket: /run/podman/podman.sock                   │
└──────────────────────────────┬──────────────────────────────┘
                               │ (socket & dir mounts)
┌──────────────────────────────▼──────────────────────────────┐
│ Main Container ("go-mcp-ui" / go-mcp-server)                │
│  - React Frontend (Bun + Vite + shadcn/ui)                  │
│  - Go REST & WebSocket API Server                           │
│  - Podman Socket Client (/v4.0.0/libpod REST API)           │
│  - Roles & Auth: Viewer vs Admin                            │
└──────────────────────────────┬──────────────────────────────┘
                               │ Spawns sibling containers via Podman API
       ┌───────────────────────┼───────────────────────┐
       ▼                       ▼                       ▼
┌──────────────┐        ┌──────────────┐        ┌──────────────┐
│ gomcp-agent1 │        │ gomcp-agent2 │        │ gomcp-agent3 │
│ Image:       │        │ Image:       │        │ Custom Image │
│ ubuntu:24.04 │        │ node/python  │        │ Bundled MCP  │
│ /workspace   │        │ /workspace   │        │ /workspace   │
│ Bundled MCP  │        │ Bundled MCP  │        │ Bundled MCP  │
└──────────────┘        └──────────────┘        └──────────────┘
```

### Access Control Matrix
| Feature | Viewer | Admin |
|---|:---:|:---:|
| List running / stopped containers | ✅ | ✅ |
| View container logs | ✅ | ✅ |
| Access live interactive terminal (xterm.js WebSocket) | ✅ | ✅ |
| View screenshots captured from container | ✅ | ✅ |
| Spin up new Podman container (select image, dir) | ❌ | ✅ |
| Start / Stop / Restart container | ❌ | ✅ |
| Duplicate / Clone container | ❌ | ✅ |
| Delete / Destroy container | ❌ | ✅ |
| Clone or link project directories | ❌ | ✅ |
| Configure custom MCP tools & pull images | ❌ | ✅ |

### Naming & Tagging Convention
All agent containers managed by this system are named with prefix `gomcp-` and labeled with:
- `gomcp.managed`: `"true"`
- `gomcp.project`: `<mapped-project-path>`
- `gomcp.created_at`: `<timestamp>`

---

## 3. Web UI Backend Architecture (`internal/`)

1. **`internal/webconfig`**:
   - Loads `config/web_config.yaml` using `gopkg.in/yaml.v3`.
   - Expands environment variables like `${ADMIN_PASSWORD}` and `${VIEWER_PASSWORD}`.
   - Configures server port, host, podman socket path, projects dir, default MCP tools.

2. **`internal/podman`**:
   - Directly connects to `/run/podman/podman.sock` using `net.Dial("unix", socketPath)` with standard `http.Client`.
   - Implements Podman Libpod v4 REST endpoints:
     - `GET /v4.0.0/libpod/containers/json?all=true`
     - `GET /v4.0.0/libpod/containers/{name}/json`
     - `POST /v4.0.0/libpod/containers/create`
     - `POST /v4.0.0/libpod/containers/{name}/start`
     - `POST /v4.0.0/libpod/containers/{name}/stop`
     - `DELETE /v4.0.0/libpod/containers/{name}?force=true`
     - `GET /v4.0.0/libpod/containers/{name}/logs`
     - `POST /v4.0.0/libpod/containers/{name}/exec` (interactive tty sessions)
     - `GET /v4.0.0/libpod/images/json`
     - `POST /v4.0.0/libpod/images/pull`

3. **`internal/server`**:
   - Go 1.22 `http.ServeMux` with pattern matching (`GET /api/...`, `POST /api/...`).
   - Basic HTTP Auth supporting `viewer` and `admin` roles.
   - REST API endpoints for container lifecycle, project folder browsing/linking, image listing.
   - WebSocket terminal proxy via `nhooyr.io/websocket` connecting browser xterm.js to `podman exec`.
   - Fallback SPA handler for `web/dist`.

4. **`cmd/go-mcp-server/main.go`**:
   - Flags: `--config config/web_config.yaml`.
   - Graceful shutdown handling.

---

## 4. Frontend Architecture (`web/`)

- Built with **Bun** + **Vite** + **React** + **TypeScript** + **TailwindCSS** + **shadcn/ui**.
- Dark aesthetic, modern HUD styling with rich visual cues.
- Real-time terminal with `@xterm/xterm` + `@xterm/addon-fit`.
- Routes:
  - `/login`: Role login (Admin / Viewer credentials stored base64 in localStorage).
  - `/`: Container grid with real-time status indicators (CPU, Memory, State, Project link).
  - `/containers/new`: Admin wizard to spin up container (choose image, choose project dir or tmp, select bundled MCP tools).
  - `/containers/:id`: Container details, logs, screenshot viewer, live interactive bash terminal.
  - `/projects`: Project directory linker & clone tool.
  - `/images`: Base image manager & puller.

---

## 5. Development & Execution Commands

```bash
# Backend dependencies
go get gopkg.in/yaml.v3
go get nhooyr.io/websocket

# Build backend
go build -o go-mcp-server ./cmd/go-mcp-server

# Frontend dependencies & build
cd web
bun install
bun run build # builds into web/dist
cd ..

# Run server locally
./go-mcp-server --config config/web_config.yaml
```


---

## 6. Local Chrome Browser Control (Optional)

The `browser-control/` directory adds a separate Node.js MCP server and Chrome Manifest V3 extension. It intentionally does not replace the existing Go MCP client, Podman server, or React dashboard.

### Architecture

- MCP server: `browser-control/server/index.mjs` using `@modelcontextprotocol/server` v2 and stdio transport.
- Local bridge: authenticated WebSocket bound only to `127.0.0.1:8765`.
- Extension: `browser-control/extension/`, with a Manifest V3 service worker.
- MCP host starts the Node server; the extension connects outbound to its loopback WebSocket endpoint.
- Authentication uses the same random `BROWSER_BRIDGE_TOKEN` in the server environment and extension options.

### WebMCP-first behavior

- `browser_list_page_tools` injects a MAIN-world function into the selected tab and checks `document.modelContext`.
- It discovers page tools and their JSON schemas, including tools from frames when Chrome exposes them through WebMCP permissions.
- Each discovered page tool is registered as a separate dynamic `webmcp_*` MCP tool. The SDK emits tool-list change notifications when the registrations change.
- If no WebMCP tools exist, the stable generic browser tools remain available and operate through Chrome tabs/scripting APIs.
- The generic `browser_execute_page_tool` remains available for MCP clients whose tool-list cache does not refresh dynamically.

### Setup

See `browser-control/README.md`. For the existing Go agent, use `config/mcp_servers.browser.json`, set `BROWSER_BRIDGE_TOKEN`, and set `MCP_CONFIG_PATH=./config/mcp_servers.browser.json`. This config is opt-in and does not alter the default MCP server list.

### Security and limitations

- Never bind the bridge to a non-loopback interface or expose port 8765 publicly.
- The extension has broad page permissions because it controls arbitrary tabs. Users must load it unpacked and configure the shared token.
- WebMCP remains experimental and is feature-detected. Normal DOM fallback works when WebMCP is absent.
- DOM events are synthetic, not physical keyboard/mouse input. Browser-level shortcuts are not supported by `browser_press_key`.
- Screenshot capture is limited to the active visible tab.
- The MCP host should require approval for consequential actions such as sending messages, submitting forms, purchases, and closing tabs.
