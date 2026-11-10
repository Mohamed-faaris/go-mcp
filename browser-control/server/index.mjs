import { randomBytes, timingSafeEqual } from 'node:crypto';
import { createServer } from 'node:http';
import { URL } from 'node:url';
import { WebSocketServer, WebSocket } from 'ws';
import { McpServer } from '@modelcontextprotocol/server';
import { StdioServerTransport } from '@modelcontextprotocol/server/stdio';
import * as z from 'zod/v4';

const HOST = '127.0.0.1';
const PORT = Number(process.env.BROWSER_BRIDGE_PORT || 8765);
const TOKEN = process.env.BROWSER_BRIDGE_TOKEN;
if (!TOKEN || TOKEN.length < 32) {
  console.error('BROWSER_BRIDGE_TOKEN must be set to a random value of at least 32 characters.');
  process.exit(1);
}

let extension = null;
let nextId = 0;
const pending = new Map();

function safeEqual(a, b) {
  const aa = Buffer.from(String(a));
  const bb = Buffer.from(String(b));
  return aa.length === bb.length && timingSafeEqual(aa, bb);
}

const httpServer = createServer((req, res) => {
  if (req.url === '/health') {
    res.writeHead(200, { 'content-type': 'application/json', 'access-control-allow-origin': 'null' });
    res.end(JSON.stringify({ ok: true, extensionConnected: Boolean(extension && extension.readyState === WebSocket.OPEN) }));
    return;
  }
  res.writeHead(404);
  res.end('Not found');
});

const wss = new WebSocketServer({ noServer: true, maxPayload: 2 * 1024 * 1024 });
httpServer.on('upgrade', (req, socket, head) => {
  const url = new URL(req.url || '/', 'http://127.0.0.1');
  const remote = req.socket.remoteAddress;
  const isLoopback = remote === '127.0.0.1' || remote === '::1' || remote === '::ffff:127.0.0.1';
  if (url.pathname !== '/bridge' || !isLoopback || !safeEqual(url.searchParams.get('token') || '', TOKEN)) {
    socket.write('HTTP/1.1 403 Forbidden\r\n\r\n');
    socket.destroy();
    return;
  }
  wss.handleUpgrade(req, socket, head, ws => wss.emit('connection', ws));
});

wss.on('connection', ws => {
  if (extension && extension.readyState === WebSocket.OPEN) extension.close(4001, 'Replaced by a newer extension connection');
  extension = ws;
  console.error('Browser extension connected');
  ws.on('message', raw => {
    let message;
    try { message = JSON.parse(raw.toString()); } catch { return; }
    if (message.type !== 'response' || typeof message.id !== 'number') return;
    const waiter = pending.get(message.id);
    if (!waiter) return;
    pending.delete(message.id);
    clearTimeout(waiter.timer);
    if (message.ok) waiter.resolve(message.result);
    else waiter.reject(new Error(message.error || 'Browser action failed'));
  });
  ws.on('close', () => {
    if (extension === ws) extension = null;
    for (const [id, waiter] of pending) {
      clearTimeout(waiter.timer);
      waiter.reject(new Error('Chrome extension disconnected'));
      pending.delete(id);
    }
    console.error('Browser extension disconnected');
  });
});

function browserRequest(action, payload = {}, timeoutMs = 20000) {
  if (!extension || extension.readyState !== WebSocket.OPEN) {
    throw new Error('Chrome extension is not connected. Open the extension settings and connect it to this MCP bridge.');
  }
  const id = ++nextId;
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      pending.delete(id);
      reject(new Error(`Browser action timed out: ${action}`));
    }, timeoutMs);
    pending.set(id, { resolve, reject, timer });
    extension.send(JSON.stringify({ type: 'request', id, action, payload }));
  });
}

function textResult(value) {
  return { content: [{ type: 'text', text: JSON.stringify(value, null, 2) }] };
}

const mcp = new McpServer({ name: 'go-mcp-browser-control', version: '0.1.0' });

mcp.registerTool('browser_status', {
  description: 'Check whether the local Chrome extension is connected to the browser-control bridge.',
  inputSchema: z.object({})
}, async () => textResult({ connected: Boolean(extension && extension.readyState === WebSocket.OPEN), bridge: `ws://${HOST}:${PORT}/bridge` }));

mcp.registerTool('browser_list_tabs', {
  description: 'List tabs currently open in the connected Chrome browser.',
  inputSchema: z.object({})
}, async () => textResult(await browserRequest('listTabs')));

mcp.registerTool('browser_get_active_tab', {
  description: 'Return the active Chrome tab and its URL/title.',
  inputSchema: z.object({})
}, async () => textResult(await browserRequest('getActiveTab')));

mcp.registerTool('browser_create_tab', {
  description: 'Create a new Chrome tab, optionally navigating to a URL.',
  inputSchema: z.object({ url: z.string().url().optional() })
}, async ({ url }) => textResult(await browserRequest('createTab', { url })));

mcp.registerTool('browser_close_tab', {
  description: 'Close a Chrome tab by tab ID. This action is destructive.',
  inputSchema: z.object({ tabId: z.number().int().positive() })
}, async ({ tabId }) => textResult(await browserRequest('closeTab', { tabId })));

mcp.registerTool('browser_navigate', {
  description: 'Navigate a tab to a URL. If tabId is omitted, use the active tab.',
  inputSchema: z.object({ url: z.string().url(), tabId: z.number().int().optional() })
}, async ({ url, tabId }) => textResult(await browserRequest('navigate', { url, tabId })));

mcp.registerTool('browser_read_page', {
  description: 'Read visible page text and basic metadata from the active tab or a specified tab.',
  inputSchema: z.object({ tabId: z.number().int().optional(), maxChars: z.number().int().min(100).max(50000).optional() })
}, async ({ tabId, maxChars }) => textResult(await browserRequest('readPage', { tabId, maxChars })));

mcp.registerTool('browser_click', {
  description: 'Click a visible element by CSS selector or exact visible text. Prefer a page WebMCP tool when one exists.',
  inputSchema: z.object({ selector: z.string().optional(), text: z.string().optional(), tabId: z.number().int().optional() }).refine(v => Boolean(v.selector || v.text), 'Provide selector or text')
}, async args => textResult(await browserRequest('click', args)));

mcp.registerTool('browser_type', {
  description: 'Type text into an input or contenteditable element selected by CSS selector.',
  inputSchema: z.object({ selector: z.string(), text: z.string(), tabId: z.number().int().optional(), clear: z.boolean().optional() })
}, async args => textResult(await browserRequest('type', args)));

mcp.registerTool('browser_press_key', {
  description: 'Press a keyboard key in the active page, such as Enter, Escape, or Tab.',
  inputSchema: z.object({ key: z.string().min(1).max(32), tabId: z.number().int().optional() })
}, async args => textResult(await browserRequest('pressKey', args)));

mcp.registerTool('browser_screenshot', {
  description: 'Capture the visible viewport of the active tab and return a PNG data URL.',
  inputSchema: z.object({ tabId: z.number().int().optional() })
}, async args => textResult(await browserRequest('screenshot', args, 30000)));

mcp.registerTool('browser_list_page_tools', {
  description: 'Discover WebMCP tools exposed by the current page and accessible frames. Returns an empty list when WebMCP is unavailable.',
  inputSchema: z.object({ tabId: z.number().int().optional() })
}, async ({ tabId }) => textResult(await browserRequest('listPageTools', { tabId })));

mcp.registerTool('browser_execute_page_tool', {
  description: 'Execute a WebMCP tool discovered on the selected page. Use browser_list_page_tools first. This is the preferred path when the page exposes a suitable tool.',
  inputSchema: z.object({ name: z.string(), args: z.record(z.string(), z.unknown()).default({}), origin: z.string().optional(), tabId: z.number().int().optional() })
}, async args => textResult(await browserRequest('executePageTool', args, 60000)));

const transport = new StdioServerTransport();
await mcp.connect(transport);
httpServer.listen(PORT, HOST, () => console.error(`Browser bridge listening on ws://${HOST}:${PORT}/bridge`));

process.on('SIGINT', () => { httpServer.close(); process.exit(0); });
process.on('SIGTERM', () => { httpServer.close(); process.exit(0); });
