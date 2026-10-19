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

## Roadmap

- [ ] Phase 1 ✅ Configuration and MCP connectivity
- [ ] Phase 2 ✅ LLM tool-calling loop
- [ ] Phase 3 ✅ GUI and vision support
- [ ] Phase 4 ✅ Permissions and reliability
- [ ] Phase 5: Task planning and recovery
- [ ] Phase 6: Web UI dashboard

## License

MIT
