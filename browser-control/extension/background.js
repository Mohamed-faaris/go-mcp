const DEFAULT_URL = 'ws://127.0.0.1:8765/bridge';
let socket;
let reconnectTimer;

async function settings() {
  return chrome.storage.local.get({ bridgeUrl: DEFAULT_URL, token: '' });
}

async function connect() {
  const { bridgeUrl, token } = await settings();
  if (!token || socket?.readyState === WebSocket.OPEN || socket?.readyState === WebSocket.CONNECTING) return;
  try {
    socket = new WebSocket(`${bridgeUrl}?token=${encodeURIComponent(token)}`);
    socket.onopen = () => { chrome.action.setBadgeText({ text: 'ON' }); chrome.action.setBadgeBackgroundColor({ color: '#16803d' }); };
    socket.onclose = () => { chrome.action.setBadgeText({ text: 'OFF' }); scheduleReconnect(); };
    socket.onerror = () => socket?.close();
    socket.onmessage = async event => {
      let request;
      try { request = JSON.parse(event.data); } catch { return; }
      if (request.type !== 'request' || typeof request.id !== 'number') return;
      try {
        const result = await handle(request.action, request.payload || {});
        socket?.send(JSON.stringify({ type: 'response', id: request.id, ok: true, result }));
      } catch (error) {
        socket?.send(JSON.stringify({ type: 'response', id: request.id, ok: false, error: String(error?.message || error) }));
      }
    };
  } catch { scheduleReconnect(); }
}

function scheduleReconnect() {
  clearTimeout(reconnectTimer);
  reconnectTimer = setTimeout(connect, 2500);
}

async function getTab(tabId) {
  if (tabId) {
    const tab = await chrome.tabs.get(tabId);
    if (!tab?.id) throw new Error('Tab not found');
    return tab;
  }
  const [tab] = await chrome.tabs.query({ active: true, lastFocusedWindow: true });
  if (!tab?.id) throw new Error('No active tab');
  return tab;
}

async function runMain(tabId, func, args = []) {
  const results = await chrome.scripting.executeScript({ target: { tabId }, world: 'MAIN', func, args });
  return results?.[0]?.result;
}

async function discoverWebMCP(tabId) {
  const frames = await chrome.webNavigation?.getAllFrames?.({ tabId }).catch(() => []) || [];
  const origins = [...new Set(frames.map(frame => { try { return new URL(frame.url).origin; } catch { return null; } }).filter(Boolean))];
  return runMain(tabId, async frameOrigins => {
    if (!('modelContext' in document) || !document.modelContext?.getTools) return [];
    try {
      const tools = await document.modelContext.getTools({ fromOrigins: frameOrigins });
      return tools.map(tool => ({
        name: tool.name, title: tool.title || '', description: tool.description || '',
        inputSchema: tool.inputSchema || { type: 'object', properties: {} },
        origin: tool.origin || location.origin,
        annotations: tool.annotations || {}
      }));
    } catch (error) { return { error: String(error) }; }
  }, [origins]);
}

async function executeWebMCP(tabId, { name, args = {}, origin }) {
  return runMain(tabId, async ({ name, args, origin }) => {
    if (!('modelContext' in document) || !document.modelContext?.getTools || !document.modelContext?.executeTool) {
      throw new Error('WebMCP is unavailable in this page');
    }
    const origins = origin ? [origin] : undefined;
    const tools = await document.modelContext.getTools(origins ? { fromOrigins: origins } : {});
    const tool = tools.find(item => item.name === name && (!origin || item.origin === origin));
    if (!tool) throw new Error(`WebMCP tool not found: ${name}`);
    return await document.modelContext.executeTool(tool, args);
  }, [{ name, args, origin }]);
}

async function handle(action, payload) {
  if (action === 'listTabs') return (await chrome.tabs.query({})).map(({ id, title, url, active, windowId }) => ({ id, title, url, active, windowId }));
  if (action === 'getActiveTab') {
    const tab = await getTab();
    return { id: tab.id, title: tab.title, url: tab.url, active: tab.active };
  }
  if (action === 'createTab') return chrome.tabs.create({ url: payload.url || 'chrome://newtab', active: true });
  if (action === 'closeTab') { await chrome.tabs.remove(payload.tabId); return { closed: payload.tabId }; }
  if (action === 'navigate') {
    const tab = await getTab(payload.tabId);
    await chrome.tabs.update(tab.id, { url: payload.url });
    return { tabId: tab.id, navigatingTo: payload.url };
  }
  if (action === 'screenshot') {
    const tab = await getTab(payload.tabId);
    const dataUrl = await chrome.tabs.captureVisibleTab(tab.windowId, { format: 'png' });
    return { tabId: tab.id, dataUrl };
  }
  const tab = await getTab(payload.tabId);
  if (action === 'listPageTools') return { tabId: tab.id, tools: await discoverWebMCP(tab.id) };
  if (action === 'executePageTool') return { tabId: tab.id, result: await executeWebMCP(tab.id, payload) };
  if (action === 'readPage') return runMain(tab.id, maxChars => ({
    title: document.title, url: location.href, text: (document.body?.innerText || '').slice(0, maxChars || 12000)
  }), [payload.maxChars || 12000]);
  if (action === 'click') return runMain(tab.id, ({ selector, text }) => {
    let element = selector ? document.querySelector(selector) : [...document.querySelectorAll('button,a,[role="button"],input[type="submit"]')].find(el => (el.innerText || el.getAttribute('aria-label') || el.value || '').trim() === text);
    if (!element) throw new Error('Clickable element not found');
    if (element instanceof HTMLElement) { element.scrollIntoView({ block: 'center' }); element.click(); }
    return { clicked: true, tag: element.tagName, text: (element.innerText || '').trim().slice(0, 200) };
  }, [{ selector: payload.selector, text: payload.text }]);
  if (action === 'type') return runMain(tab.id, ({ selector, text, clear }) => {
    const element = document.querySelector(selector);
    if (!element) throw new Error('Input element not found');
    if (!(element instanceof HTMLInputElement || element instanceof HTMLTextAreaElement || element.isContentEditable)) throw new Error('Target is not an editable field');
    element.focus();
    if (element.isContentEditable) { if (clear) element.textContent = ''; document.execCommand('insertText', false, text); }
    else {
      const proto = element instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
      const setter = Object.getOwnPropertyDescriptor(proto, 'value')?.set;
      if (clear) setter?.call(element, '');
      setter?.call(element, (element.value || '') + text);
      element.dispatchEvent(new Event('input', { bubbles: true }));
      element.dispatchEvent(new Event('change', { bubbles: true }));
    }
    return { typed: true, selector };
  }, payload);
  if (action === 'pressKey') return runMain(tab.id, key => {
    const target = document.activeElement || document.body;
    target.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true }));
    target.dispatchEvent(new KeyboardEvent('keyup', { key, bubbles: true }));
    return { key, dispatched: true };
  }, [payload.key]);
  throw new Error(`Unknown browser action: ${action}`);
}

chrome.runtime.onInstalled.addListener(connect);
chrome.runtime.onStartup.addListener(connect);
chrome.storage.onChanged.addListener(connect);
connect();
