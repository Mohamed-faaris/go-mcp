import React, { useEffect, useRef, useState } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { RefreshCw, Trash2, Maximize2, Minimize2, Terminal as TerminalIcon } from 'lucide-react';
import { getTerminalWsUrl } from '../api';

interface TerminalViewProps {
  containerId: string;
  containerName?: string;
  onExit?: () => void;
}

export const TerminalView: React.FC<TerminalViewProps> = ({ containerId, containerName }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);

  const [connectionStatus, setConnectionStatus] = useState<'connecting' | 'connected' | 'disconnected'>('connecting');
  const [isFullscreen, setIsFullscreen] = useState(false);

  const connectWebSocket = () => {
    if (!termRef.current) return;
    const term = termRef.current;

    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    setConnectionStatus('connecting');
    term.write('\r\n\x1b[33mConnecting to interactive agent shell...\x1b[0m\r\n');

    const url = getTerminalWsUrl(containerId);
    const ws = new WebSocket(url);
    ws.binaryType = 'arraybuffer';
    wsRef.current = ws;

    ws.onopen = () => {
      setConnectionStatus('connected');
      term.write('\x1b[32m✔ Connected to container shell session.\x1b[0m\r\n\r\n');
      if (fitAddonRef.current) {
        fitAddonRef.current.fit();
      }
    };

    ws.onmessage = (event) => {
      if (event.data instanceof ArrayBuffer) {
        const uint8View = new Uint8Array(event.data);
        term.write(uint8View);
      } else if (typeof event.data === 'string') {
        term.write(event.data);
      }
    };

    ws.onerror = (err) => {
      console.error('Terminal WebSocket error:', err);
      term.write('\r\n\x1b[31m[!] Terminal connection error\x1b[0m\r\n');
    };

    ws.onclose = (event) => {
      setConnectionStatus('disconnected');
      term.write(`\r\n\x1b[90m[-] Session disconnected (code ${event.code}).\x1b[0m\r\n`);
    };
  };

  useEffect(() => {
    if (!containerRef.current) return;

    // Initialize xterm instance
    const term = new Terminal({
      cursorBlink: true,
      cursorStyle: 'block',
      fontFamily: "'JetBrains Mono', monospace",
      fontSize: 14,
      lineHeight: 1.25,
      theme: {
        background: '#0a0d14',
        foreground: '#e2e8f0',
        cursor: '#38bdf8',
        cursorAccent: '#0a0d14',
        selectionBackground: 'rgba(56, 189, 248, 0.3)',
        black: '#1e293b',
        red: '#f43f5e',
        green: '#10b981',
        yellow: '#f59e0b',
        blue: '#3b82f6',
        magenta: '#d946ef',
        cyan: '#06b6d4',
        white: '#f8fafc',
        brightBlack: '#475569',
        brightRed: '#fb7185',
        brightGreen: '#34d399',
        brightYellow: '#fbbf24',
        brightBlue: '#60a5fa',
        brightMagenta: '#e879f9',
        brightCyan: '#22d3ee',
        brightWhite: '#ffffff',
      },
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);

    term.open(containerRef.current);
    fitAddon.fit();

    termRef.current = term;
    fitAddonRef.current = fitAddon;

    // Send keystrokes from terminal to WebSocket
    const onDataDisposable = term.onData((data) => {
      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
        wsRef.current.send(data);
      }
    });

    connectWebSocket();

    // Resize observer to keep xterm fitted to parent element
    const resizeObserver = new ResizeObserver(() => {
      try {
        fitAddon.fit();
      } catch (e) {
        // Ignored
      }
    });
    resizeObserver.observe(containerRef.current);

    return () => {
      resizeObserver.disconnect();
      onDataDisposable.dispose();
      if (wsRef.current) {
        wsRef.current.close();
      }
      term.dispose();
    };
  }, [containerId]);

  const handleClear = () => {
    if (termRef.current) {
      termRef.current.clear();
      termRef.current.focus();
    }
  };

  const handleReconnect = () => {
    connectWebSocket();
    if (termRef.current) {
      termRef.current.focus();
    }
  };

  return (
    <div
      className={`flex flex-col bg-[#0a0d14] border border-slate-800 rounded-xl overflow-hidden shadow-2xl transition-all ${
        isFullscreen ? 'fixed inset-4 z-50 rounded-2xl' : 'h-[540px]'
      }`}
    >
      {/* Terminal Title Bar */}
      <div className="flex items-center justify-between px-4 py-2.5 bg-slate-900/90 border-b border-slate-800 text-xs">
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-1.5">
            <div className="w-2.5 h-2.5 rounded-full bg-rose-500/80" />
            <div className="w-2.5 h-2.5 rounded-full bg-amber-500/80" />
            <div className="w-2.5 h-2.5 rounded-full bg-emerald-500/80" />
          </div>
          <div className="h-4 w-px bg-slate-800" />
          <div className="flex items-center gap-2 text-slate-300 font-mono font-medium">
            <TerminalIcon className="w-3.5 h-3.5 text-blue-400" />
            <span>{containerName || containerId.slice(0, 12)}</span>
            <span className="text-slate-500 font-normal">(/workspace)</span>
          </div>
        </div>

        <div className="flex items-center gap-3">
          {/* Connection status badge */}
          <div className="flex items-center gap-1.5 px-2 py-0.5 rounded-full bg-slate-800/80 border border-slate-700/50">
            <span
              className={`w-2 h-2 rounded-full ${
                connectionStatus === 'connected'
                  ? 'bg-emerald-400 animate-pulse'
                  : connectionStatus === 'connecting'
                  ? 'bg-amber-400 animate-ping'
                  : 'bg-rose-500'
              }`}
            />
            <span className="text-[11px] font-medium capitalize text-slate-300">
              {connectionStatus}
            </span>
          </div>

          <div className="h-4 w-px bg-slate-800" />

          {/* Action buttons */}
          <button
            onClick={handleReconnect}
            title="Reconnect Session"
            className="p-1 rounded text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition"
          >
            <RefreshCw className="w-3.5 h-3.5" />
          </button>
          <button
            onClick={handleClear}
            title="Clear Terminal Buffer"
            className="p-1 rounded text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition"
          >
            <Trash2 className="w-3.5 h-3.5" />
          </button>
          <button
            onClick={() => setIsFullscreen(!isFullscreen)}
            title={isFullscreen ? 'Exit Fullscreen' : 'Fullscreen'}
            className="p-1 rounded text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition"
          >
            {isFullscreen ? <Minimize2 className="w-3.5 h-3.5" /> : <Maximize2 className="w-3.5 h-3.5" />}
          </button>
        </div>
      </div>

      {/* Terminal Canvas */}
      <div
        ref={containerRef}
        className="flex-1 w-full h-full p-2 overflow-hidden cursor-text"
        onClick={() => termRef.current?.focus()}
      />
    </div>
  );
};
