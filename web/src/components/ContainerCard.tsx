import React, { useState } from 'react';
import { Play, Square, Terminal, Copy, Trash2, Folder, HardDrive, Cpu, ShieldCheck } from 'lucide-react';
import { ContainerSummary } from '../types';

interface ContainerCardProps {
  container: ContainerSummary;
  isAdmin: boolean;
  onStart: (id: string) => Promise<void>;
  onStop: (id: string) => Promise<void>;
  onDuplicate: (container: ContainerSummary) => void;
  onDelete: (container: ContainerSummary) => void;
}

export const ContainerCard: React.FC<ContainerCardProps> = ({
  container,
  isAdmin,
  onStart,
  onStop,
  onDuplicate,
  onDelete,
}) => {
  const [loading, setLoading] = useState(false);

  const cleanName = container.Names && container.Names.length > 0
    ? container.Names[0].replace(/^\//, '')
    : container.Id.slice(0, 12);

  const isRunning = container.State.toLowerCase() === 'running';

  // Extract MCP tools from labels if present
  let mcpToolsCount = 0;
  if (container.Labels && container.Labels['gomcp.mcp_tools']) {
    try {
      const tools = JSON.parse(container.Labels['gomcp.mcp_tools']);
      mcpToolsCount = Array.isArray(tools) ? tools.length : 0;
    } catch {
      // ignore
    }
  }

  const handleToggleState = async () => {
    setLoading(true);
    try {
      if (isRunning) {
        await onStop(container.Id);
      } else {
        await onStart(container.Id);
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="group relative bg-[#111827]/90 border border-slate-800/90 hover:border-slate-700/80 rounded-2xl p-5 shadow-xl transition-all duration-300 hover:shadow-2xl hover:shadow-blue-500/5 hover:-translate-y-0.5 flex flex-col justify-between">
      {/* Top row */}
      <div>
        <div className="flex items-start justify-between gap-3 mb-3">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <h3 className="text-base font-semibold text-slate-100 truncate group-hover:text-blue-400 transition">
                {cleanName}
              </h3>
              {container.Labels?.['gomcp.managed'] === 'true' && (
                <span title="Managed by go-mcp" className="text-blue-400">
                  <ShieldCheck className="w-4 h-4" />
                </span>
              )}
            </div>
            <p className="text-xs font-mono text-slate-500 truncate mt-0.5">
              ID: {container.Id.slice(0, 12)}
            </p>
          </div>

          {/* Status pill */}
          <div
            className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border ${
              isRunning
                ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20 shadow-sm shadow-emerald-500/10'
                : 'bg-slate-800 text-slate-400 border-slate-700'
            }`}
          >
            <span
              className={`w-2 h-2 rounded-full ${
                isRunning ? 'bg-emerald-400 animate-pulse' : 'bg-slate-500'
              }`}
            />
            <span className="capitalize">{container.State}</span>
          </div>
        </div>

        {/* Specs metadata badges */}
        <div className="grid grid-cols-2 gap-2 text-xs my-4">
          <div className="flex items-center gap-2 p-2 rounded-lg bg-slate-900/60 border border-slate-800/60 text-slate-300">
            <Cpu className="w-3.5 h-3.5 text-blue-400 flex-shrink-0" />
            <span className="truncate font-mono" title={container.Image}>
              {container.Image.split('/').pop()}
            </span>
          </div>

          <div className="flex items-center gap-2 p-2 rounded-lg bg-slate-900/60 border border-slate-800/60 text-slate-300">
            <HardDrive className="w-3.5 h-3.5 text-indigo-400 flex-shrink-0" />
            <span className="truncate">
              {mcpToolsCount > 0 ? `${mcpToolsCount} MCP Tools` : 'Standard'}
            </span>
          </div>
        </div>

        {/* Mapped Project Path */}
        {container.projectDir ? (
          <div className="flex items-center gap-2 text-xs text-slate-400 p-2 rounded-lg bg-slate-900/40 border border-slate-800/40 mb-4 truncate">
            <Folder className="w-3.5 h-3.5 text-amber-400 flex-shrink-0" />
            <span className="truncate font-mono" title={container.projectDir}>
              {container.projectDir}
            </span>
          </div>
        ) : (
          <div className="flex items-center gap-2 text-xs text-slate-500 p-2 rounded-lg bg-slate-900/40 border border-slate-800/40 mb-4">
            <Folder className="w-3.5 h-3.5 text-slate-600 flex-shrink-0" />
            <span className="italic">No host project mapped</span>
          </div>
        )}
      </div>

      {/* Action toolbar */}
      <div className="pt-3 border-t border-slate-800/80 flex items-center justify-between gap-2">
        <a
          href={`#/containers/${container.Id}`}
          className="flex-1 flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-xs font-semibold bg-blue-600/10 hover:bg-blue-600 text-blue-400 hover:text-white border border-blue-500/20 hover:border-blue-500 transition shadow-sm"
        >
          <Terminal className="w-3.5 h-3.5" />
          <span>Interactive Shell</span>
        </a>

        {isAdmin && (
          <div className="flex items-center gap-1">
            <button
              onClick={handleToggleState}
              disabled={loading}
              title={isRunning ? 'Stop Container' : 'Start Container'}
              className={`p-2 rounded-lg text-xs font-medium border transition disabled:opacity-50 ${
                isRunning
                  ? 'bg-amber-500/10 text-amber-400 border-amber-500/20 hover:bg-amber-500/20'
                  : 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20 hover:bg-emerald-500/20'
              }`}
            >
              {loading ? (
                <div className="w-3.5 h-3.5 border-2 border-current border-t-transparent rounded-full animate-spin" />
              ) : isRunning ? (
                <Square className="w-3.5 h-3.5" />
              ) : (
                <Play className="w-3.5 h-3.5" />
              )}
            </button>

            <button
              onClick={() => onDuplicate(container)}
              title="Duplicate Container"
              className="p-2 rounded-lg text-slate-400 hover:text-slate-200 bg-slate-800/50 hover:bg-slate-800 border border-slate-700/50 transition"
            >
              <Copy className="w-3.5 h-3.5" />
            </button>

            <button
              onClick={() => onDelete(container)}
              title="Destroy Container"
              className="p-2 rounded-lg text-slate-400 hover:text-rose-400 bg-slate-800/50 hover:bg-rose-500/10 border border-slate-700/50 hover:border-rose-500/30 transition"
            >
              <Trash2 className="w-3.5 h-3.5" />
            </button>
          </div>
        )}
      </div>
    </div>
  );
};
