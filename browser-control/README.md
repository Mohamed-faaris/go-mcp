# go-mcp Browser Control

A local MCP server and Chrome Manifest V3 extension that connect AI agents to the user's existing Chrome session.

## Behavior

1. The extension connects outbound to a loopback-only WebSocket bridge.
2. The MCP server exposes stable browser tools over stdio.
3. For website-specific actions, call `browser_list_page_tools` and then `browser_execute_page_tool`.
4. When WebMCP tools are found, the MCP server registers each one as a separate `webmcp_*` tool with its page-provided input schema. MCP clients receive tool-list change notifications.
5. If the page has no WebMCP tools, use the generic browser tools such as `browser_read_page`, `browser_click`, and `browser_type`.
6. Page tools are discovered again when `browser_list_page_tools` is called, so route changes and dynamic registrations are reflected. The most recently inspected tab owns the dynamic `webmcp_*` tool set.

WebMCP is experimental and feature-detected. It is not assumed to exist on every site or Chrome build. The extension falls back to DOM operations when the page does not expose WebMCP. See the [Chrome WebMCP documentation](https://developer.chrome.com/docs/ai/webmcp).

## Requirements

- Node.js 20+
- Chrome with extension developer mode enabled
- A Chrome build with WebMCP enabled for WebMCP-specific discovery (open `chrome://flags/#enable-webmcp-testing`, enable it, then relaunch)
- A local MCP host that supports stdio servers

## 1. Install the MCP server

From the repository root:

```bash
cd browser-control
npm install
```

Generate a shared secret:

```bash
openssl rand -hex 32
```

Set it in the environment before starting the MCP host:

```bash
export BROWSER_BRIDGE_TOKEN='paste-the-generated-64-character-token'
```

The server refuses to start without a token of at least 32 characters. The WebSocket listener binds only to `127.0.0.1`.

## 2. Install the Chrome extension

1. Open `chrome://extensions`.
2. Enable **Developer mode**.
3. Select **Load unpacked**.
4. Select `browser-control/extension`.
5. Open the extension's **Details → Extension options**.
6. Paste the same token generated above and save.
7. Confirm the extension badge changes to **ON**.

Keep the MCP server process running. The extension reconnects automatically when the server restarts.

## 3. Connect the MCP server

### Generic MCP host

Use `browser-control/mcp.json` as a configuration template. Resolve `PLUGIN_ROOT` to the absolute `browser-control` directory and replace `REPLACE_WITH_GENERATED_RANDOM_TOKEN` with the generated token.

### Existing go-mcp CLI agent

Set the environment variable and point the agent at the opt-in browser configuration:

```bash
export BROWSER_BRIDGE_TOKEN='same-generated-token'
export MCP_CONFIG_PATH='./config/mcp_servers.browser.json'
go run ./cmd/go-mcp --check
```

Then run the agent normally. The existing Go MCP registry launches this Node server as a stdio child process and inherits the environment.

## MCP tools

| Tool | Purpose |
|---|---|
| `browser_status` | Check extension connection |
| `browser_list_tabs` | List Chrome tabs |
| `browser_get_active_tab` | Inspect active tab |
| `browser_create_tab` | Create a tab |
| `browser_close_tab` | Close a tab |
| `browser_navigate` | Navigate to HTTP(S) URL |
| `browser_read_page` | Read visible page text |
| `browser_list_page_tools` | Discover WebMCP tools |
| `browser_execute_page_tool` | Execute a discovered WebMCP tool |
| `browser_click` | Click by selector or exact visible text |
| `browser_type` | Type into an input/contenteditable element |
| `browser_press_key` | Dispatch a synthetic page keyboard event |
| `browser_screenshot` | Capture the active visible tab |

## Security model and limitations

- The WebSocket bridge binds to loopback and requires a shared random token.
- Never expose port 8765 to a LAN or public interface.
- Keep the token private. Anyone who obtains it can control the connected browser.
- The extension requests broad page access because it must operate on arbitrary tabs. Chrome may restrict protected pages such as `chrome://` and the Chrome Web Store.
- The fallback click/type implementation is DOM-level automation, not trusted physical input. Synthetic keyboard events do not trigger browser-level shortcuts.
- Dynamic `webmcp_*` tools are announced through MCP tool-list change notifications. MCP clients must refresh tool lists to see them. The existing Go agent can still use `browser_execute_page_tool` even if its startup-time tool cache has not refreshed.
- Screenshot capture is limited to the active visible tab.
- Browser-level navigation, tab closing, form submission, purchases, messages, and other consequential actions should remain subject to the MCP host's approval controls.
- WebMCP tools in cross-origin frames are visible only when the frame and origin permissions allow them.
- This project does not bypass website authentication, CAPTCHAs, or access controls.

## Development checks

```bash
npm run check
```

Load the unpacked extension and test in this order: `browser_status`, `browser_list_tabs`, `browser_get_active_tab`, `browser_read_page`, then WebMCP discovery and execution on a WebMCP demo page.
