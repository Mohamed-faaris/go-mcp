# go-mcp

Autonomous AI agent that controls an Ubuntu computer through MCP servers, powered by any OpenAI-compatible API.

## Architecture

```
┌─────────────────────────────────────────────────────┐
│  CLI                                                │
│  go-mcp "open firefox and search for X"             │
├─────────────────────────────────────────────────────┤
│  Agent Loop                                         │
│  Planning → LLM calls → Tool selection → Context    │
├──────────┬──────────┬───────────────────────────────┤
│ Permission│ Context  │ Screenshot                    │
│ Manager   │ Manager  │ Handler                       │
├───────────┴──────────┴──────────────────────────────┤
│  MCP Client Layer                                   │
│  Server connections → Tool discovery → Execution    │
├──────────┬──────────┬───────────────────────────────┤
│ Desktop  │Playwright│ Ubuntu GUI                     │
│Commander │  MCP     │   MCP                          │
├──────────┴──────────┴───────────────────────────────┤
│  Ubuntu Operating System                            │
│  Files · Apps · Browser · Terminal · Desktop         │
└─────────────────────────────────────────────────────┘
```

## Quick start

```bash
# 1. Clone and enter the repo
git clone https://github.com/Mohamed-faaris/go-mcp.git
cd go-mcp

# 2. Configure
cp .env.example .env
chmod 600 .env
# Edit .env with your API key and model settings

# 3. Configure MCP servers
# Edit config/mcp_servers.json with your MCP server commands

# 4. Build
go build -o go-mcp ./cmd/go-mcp

# 5. Check MCP connections
./go-mcp --check

# 6. Run a task
./go-mcp "open the browser and search for Go tutorials"
```

## Configuration

All settings are loaded from `.env` (see [`.env.example`](.env.example)):

| Variable | Description | Default |
|---|---|---|
| `OPENAI_BASE_URL` | API endpoint URL | (required) |
| `OPENAI_API_KEY` | API key | (required) |
| `OPENAI_MODEL` | Model name | (required) |
| `MCP_CONFIG_PATH` | Path to MCP server config | `config/mcp_servers.json` |
| `MAX_STEPS` | Max agent loop iterations | `12` |
| `MAX_TOOL_CALLS` | Max total tool invocations | `24` |
| `REQUEST_TIMEOUT_SECONDS` | HTTP request timeout | `90` |
| `APPROVAL_MODE` | `all` / `dangerous` / `none` | `all` |
| `ALLOW_SHELL` | Allow shell command execution | `true` |
| `LOG_LEVEL` | `DEBUG` / `INFO` / `WARN` / `ERROR` | `INFO` |
| `LOG_FILE` | Log file path | `logs/agent.log` |

## MCP servers

Tools are discovered at runtime from MCP servers defined in `config/mcp_servers.json`:

| Server | Responsibility |
|---|---|
| **Desktop Commander** | Terminal commands, file operations, process management |
| **Playwright MCP** | Browser navigation, clicking, forms, page content |
| **Ubuntu GUI MCP** | Screenshots, mouse, keyboard, desktop interaction |

The agent prefers structured browser tools (Playwright) and falls back to screenshot-based GUI control when needed.

## Approval modes

| Mode | Behavior |
|---|---|
| `all` | Every tool call requires explicit user approval |
| `dangerous` | Only high-risk actions (shell, file writes, GUI clicks) need approval |
| `none` | Fully autonomous — no approvals required |

## Project structure

```
go-mcp/
├── cmd/go-mcp/          # CLI entry point
│   └── main.go
├── internal/
│   ├── agent/           # Autonomous execution loop
│   ├── config/          # Configuration loading and validation
│   ├── context/         # Conversation history management
│   ├── llm/             # OpenAI-compatible API client
│   ├── logger/          # Structured JSON logging
│   ├── mcp/             # MCP protocol client and registry
│   ├── permission/      # Approval policies and risk checks
│   └── screenshot/      # Screenshot capture and encoding
├── config/
│   └── mcp_servers.json # MCP server definitions
├── .env.example         # Configuration template
└── README.md
```

## Development

```bash
# Run tests
go test ./...

# Build
go build -o go-mcp ./cmd/go-mcp

# Run directly
go run ./cmd/go-mcp "your task here"
```

## Security notes

- `.env` is in `.gitignore` — never commit secrets
- The API key is never sent to the model or included in tool arguments
- Permission checks are enforced in Go before every tool call
- Shell execution can be disabled with `ALLOW_SHELL=false`
- Set `APPROVAL_MODE=all` until you trust the agent's behavior

## Web UI & Podman Sandbox Manager

`go-mcp-server` provides a web interface to spin up, monitor, and interact with isolated Ubuntu sandbox containers running Podman:

- **Dashboard**: Live container metrics, states, and project mount tracking.
- **Interactive Shell**: Low-latency browser terminal powered by `xterm.js` and WebSockets.
- **Host Hardware Pass-Through**: Direct V4L2 device pass-through (`/dev/videoX`) for Econ Systems cameras (e.g. See3CAM_24CUG) and USB webcams.
- **Container Lifecycle**: Spin up, duplicate/clone, start, stop, and destroy sandboxes.
- **Projects & Scratch**: Bind-mount host project directories into `/workspace`.

### Building and Running the Web UI

```bash
# 1. Build frontend (in web/)
cd web
bun install # or npm install
bun run build # outputs to web/dist
cd ..

# 2. Build and run server
go build -o go-mcp-server ./cmd/go-mcp-server
./go-mcp-server --config config/web_config.yaml
```

Access the UI at `http://localhost:8080`. Default login credentials:
- **Admin**: `admin` / `admin123`
- **Viewer**: `viewer` / `viewer123`

## Roadmap

- [x] Phase 1: Configuration and MCP connectivity
- [x] Phase 2: LLM tool-calling loop
- [x] Phase 3: GUI and vision support
- [x] Phase 4: Permissions and reliability
- [x] Phase 5: Task planning and recovery
- [x] Phase 6: Web UI dashboard & Podman sandboxing
- [x] Phase 7: Hardware & V4L2 camera pass-through

## License

MIT
