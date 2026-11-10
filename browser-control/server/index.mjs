import { createHash, timingSafeEqual } from 'node:crypto';
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

const dynamicPageTools = new Map();
let dynamicPageTabId;

function jsonSchemaToZod(schema = {}) {
  if (!schema || typeof schema !== 'object') return z.unknown();
  if (Array.isArray(schema.anyOf || schema.oneOf)) {
    const variants = (schema.anyOf || schema.oneOf).map(jsonSchemaToZod);
    return variants.length > 1 ? z.union(variants) : (variants[0] || z.unknown());
  }
  if (schema.const !== undefined) return z.literal(schema.const);
  if (Array.isArray(schema.enum) && schema.enum.length) {
    const variants = schema.enum.map(value => z.literal(value));
    return variants.length > 1 ? z.union(variants) : variants[0];
  }
  let result;
  const type = schema.type || (schema.properties ? 'object' : undefined);
  if (type === 'object') {
    const shape = {};
    const required = new Set(schema.required || []);
    for (const [key, child] of Object.entries(schema.properties || {})) {
      const childSchema = jsonSchemaToZod(child);
      shape[key] = required.has(key) ? childSchema : childSchema.optional();
    }
    result = z.object(shape);
    if (schema.additionalProperties === true) result = result.passthrough();
    else if (schema.additionalProperties && typeof schema.additionalProperties === 'object') result = result.catchall(jsonSchemaToZod(schema.additionalProperties));
  } else if (type === 'array') {
    result = z.array(jsonSchemaToZod(schema.items || {}));
    if (Number.isInteger(schema.minItems)) result = result.min(schema.minItems);
    if (Number.isInteger(schema.maxItems)) result = result.max(schema.maxItems);
  } else if (type === 'string') {
    result = z.string();
    if (Number.isInteger(schema.minLength)) result = result.min(schema.minLength);
    if (Number.isInteger(schema.maxLength)) result = result.max(schema.maxLength);
    if (schema.format === 'email') result = result.email();
    if (schema.format === 'uri' || schema.format === 'url') result = result.url();
    if (schema.format === 'uuid') result = result.uuid();
  } else if (type === 'integer') {
    result = z.number().int();
    if (typeof schema.minimum === 'number') result = result.min(schema.minimum);
    if (typeof schema.maximum === 'number') result = result.max(schema.maximum);
  } else if (type === 'number') {
    result = z.number();
    if (typeof schema.minimum === 'number') result = result.min(schema.minimum);
    if (typeof schema.maximum === 'number') result = result.max(schema.maximum);
  } else if (type === 'boolean') result = z.boolean();
  else if (type === 'null') result = z.null();
  else result = z.unknown();
  if (schema.default !== undefined && result.default) result = result.default(schema.default);
  if (schema.nullable === true && result.nullable) result = result.nullable();
  return result;
}

function clearDynamicPageTools() {
  for (const registration of dynamicPageTools.values()) registration.remove();
  dynamicPageTools.clear();
}

function exposeDynamicPageTools(tabId, pageTools) {
  clearDynamicPageTools();
  dynamicPageTabId = tabId;
  for (const pageTool of pageTools) {
    if (!pageTool?.name || typeof pageTool.name !== 'string') continue;
    const suffix = createHash('sha256').update(String(pageTool.origin || '') + ':' + pageTool.name).digest('hex').slice(0, 8);
    const slug = pageTool.name.toLowerCase().replace(/[^a-z0-9_-]+/g, '_').slice(0, 36) || 'tool';
    const exposedName = 'webmcp_' + slug + '_' + suffix;
    const annotations = pageTool.annotations || {};
    const registration = mcp.registerTool(exposedName, {
      title: pageTool.title || pageTool.name,
      description: 'Website WebMCP tool from ' + (pageTool.origin || 'the active page') + ': ' + (pageTool.description || pageTool.name),
      inputSchema: jsonSchemaToZod(pageTool.inputSchema || { type: 'object', properties: {} }),
      annotations: {
        readOnlyHint: Boolean(annotations.readOnlyHint),
        destructiveHint: Boolean(annotations.consequentialHint),
        openWorldHint: true,
        title: pageTool.title || pageTool.name
      }
    }, async args => textResult(await browserRequest('executePageTool', {
      name: pageTool.name, args, origin: pageTool.origin, tabId: dynamicPageTabId
    }, 60000)));
    dynamicPageTools.set(exposedName, registration);
  }
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

mcp.registerTool('browser_activate_tab', {
  description: 'Activate and focus a Chrome tab by tab ID.',
  inputSchema: z.object({ tabId: z.number().int().positive() })
}, async ({ tabId }) => textResult(await browserRequest('activateTab', { tabId })));

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
  description: 'Discover WebMCP tools exposed by the current page and accessible frames. Dynamically registers each discovered website tool as its own MCP tool; removes stale tools from the previously inspected page.',
  inputSchema: z.object({ tabId: z.number().int().optional() })
}, async ({ tabId }) => {
  const result = await browserRequest('listPageTools', { tabId });
  const pageTools = Array.isArray(result?.tools) ? result.tools : [];
  exposeDynamicPageTools(result?.tabId, pageTools);
  return textResult({ ...result, exposedMcpTools: [...dynamicPageTools.keys()] });
});

mcp.registerTool('browser_execute_page_tool', {
  description: 'Execute a WebMCP tool discovered on the selected page. Use browser_list_page_tools first. This is the preferred path when the page exposes a suitable tool.',
  inputSchema: z.object({ name: z.string(), args: z.record(z.string(), z.unknown()).default({}), origin: z.string().optional(), tabId: z.number().int().optional() })
}, async args => textResult(await browserRequest('executePageTool', args, 60000)));

const transport = new StdioServerTransport();
await mcp.connect(transport);
httpServer.listen(PORT, HOST, () => console.error(`Browser bridge listening on ws://${HOST}:${PORT}/bridge`));

process.on('SIGINT', () => { httpServer.close(); process.exit(0); });
process.on('SIGTERM', () => { httpServer.close(); process.exit(0); });
