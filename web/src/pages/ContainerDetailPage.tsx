import React, { useState, useEffect } from 'react';
import { ArrowLeft, Play, Square, Copy, Trash2, Terminal, FileText, Info, RefreshCw, Folder, Cpu, HardDrive, ShieldCheck, Check, AlertCircle } from 'lucide-react';
import { apiRequest } from '../api';
import { ContainerInspect, AuthUser } from '../types';
import { TerminalView } from '../components/TerminalView';
import { Modal } from '../components/Modal';

interface ContainerDetailPageProps {
  containerId: string;
  user: AuthUser | null;
}

export const ContainerDetailPage: React.FC<ContainerDetailPageProps> = ({ containerId, user }) => {
  const [inspect, setInspect] = useState<ContainerInspect | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'terminal' | 'logs' | 'inspect'>('terminal');

  // Logs state
  const [logs, setLogs] = useState<string>('');
  const [logsLoading, setLogsLoading] = useState(false);
  const [logTail, setLogTail] = useState(200);
  const [logFilter, setLogFilter] = useState('');
  const [autoRefreshLogs, setAutoRefreshLogs] = useState(false);

  // Duplicate modal
  const [isDuplicateOpen, setIsDuplicateOpen] = useState(false);
  const [duplicateName, setDuplicateName] = useState('');
  const [duplicateLoading, setDuplicateLoading] = useState(false);

  // Delete modal
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [deleteLoading, setDeleteLoading] = useState(false);

  const [actionLoading, setActionLoading] = useState(false);
  const isAdmin = user?.role === 'admin';

  const fetchContainer = async () => {
    try {
      const data = await apiRequest<ContainerInspect>(`/api/containers/${containerId}`);
      setInspect(data);
      setError(null);
    } catch (err: any) {
      setError(err.message || 'Failed to inspect container');
    } finally {
      setLoading(false);
    }
  };

  const fetchLogs = async () => {
    setLogsLoading(true);
    try {
      const res = await apiRequest<{ logs: string }>(
        `/api/containers/${containerId}/logs?tail=${logTail}`
      );
      setLogs(res.logs || '');
    } catch (err: any) {
      console.error('Failed to fetch logs:', err);
    } finally {
      setLogsLoading(false);
    }
  };

  useEffect(() => {
    fetchContainer();
  }, [containerId]);

  useEffect(() => {
    if (activeTab === 'logs') {
      fetchLogs();
    }
  }, [activeTab, logTail]);

  useEffect(() => {
    let interval: any = null;
    if (activeTab === 'logs' && autoRefreshLogs) {
      interval = setInterval(() => {
        fetchLogs();
      }, 3000);
    }
    return () => {
      if (interval) clearInterval(interval);
    };
  }, [activeTab, autoRefreshLogs, logTail]);

  const handleStart = async () => {
    setActionLoading(true);
    try {
      await apiRequest(`/api/containers/${containerId}/start`, { method: 'POST' });
      await fetchContainer();
    } catch (err: any) {
      alert(`Start failed: ${err.message}`);
    } finally {
      setActionLoading(false);
    }
  };

  const handleStop = async () => {
    setActionLoading(true);
    try {
      await apiRequest(`/api/containers/${containerId}/stop`, { method: 'POST' });
      await fetchContainer();
    } catch (err: any) {
      alert(`Stop failed: ${err.message}`);
    } finally {
      setActionLoading(false);
    }
  };

  const handleDuplicate = async (e: React.FormEvent) => {
    e.preventDefault();
    setDuplicateLoading(true);
    try {
      const res = await apiRequest<{ id: string }>(`/api/containers/${containerId}/duplicate`, {
        method: 'POST',
        body: JSON.stringify({ newName: duplicateName }),
      });
      setIsDuplicateOpen(false);
      window.location.hash = `#/containers/${res.id}`;
    } catch (err: any) {
      alert(`Duplicate failed: ${err.message}`);
    } finally {
      setDuplicateLoading(false);
    }
  };

  const handleDelete = async () => {
    setDeleteLoading(true);
    try {
      await apiRequest(`/api/containers/${containerId}`, { method: 'DELETE' });
      setIsDeleteOpen(false);
      window.location.hash = '#/';
    } catch (err: any) {
      alert(`Delete failed: ${err.message}`);
    } finally {
      setDeleteLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="max-w-7xl mx-auto px-4 py-24 flex flex-col items-center justify-center gap-3 text-slate-500">
        <div className="w-8 h-8 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
        <p className="text-sm">Inspecting container specifications...</p>
      </div>
    );
  }

  if (error || !inspect) {
    return (
      <div className="max-w-4xl mx-auto px-4 py-16">
        <div className="p-6 rounded-2xl bg-rose-500/10 border border-rose-500/20 text-rose-300">
          <div className="flex items-center gap-3 mb-3">
            <AlertCircle className="w-6 h-6 text-rose-400" />
            <h2 className="text-lg font-bold">Container Not Found or Inaccessible</h2>
          </div>
          <p className="text-sm mb-4">{error || 'Unable to retrieve container state from Podman.'}</p>
          <a
            href="#/"
            className="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-slate-900 text-slate-200 text-xs font-semibold border border-slate-800"
          >
            <ArrowLeft className="w-4 h-4" />
            <span>Return to Sandboxes</span>
          </a>
        </div>
      </div>
    );
  }

  const cleanName = inspect.Name.replace(/^\//, '');
  const isRunning = inspect.State.Running;
  const projectMount = inspect.Mounts?.find((m) => m.Destination === '/workspace');

  // Filtered logs
  const filteredLogs = logs
    ? logs
        .split('\n')
        .filter((line) => line.toLowerCase().includes(logFilter.toLowerCase()))
        .join('\n')
    : 'No logs captured.';

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-6 animate-in fade-in duration-300">
      {/* Breadcrumb navigation */}
      <div className="flex items-center gap-2 text-xs text-slate-400">
        <a href="#/" className="hover:text-slate-200 transition">
          Sandboxes
        </a>
        <span>/</span>
        <span className="text-slate-200 font-mono font-medium">{cleanName}</span>
      </div>

      {/* Header Info & Actions */}
      <div className="bg-[#111827]/90 border border-slate-800 rounded-2xl p-6 shadow-xl flex flex-col md:flex-row md:items-center justify-between gap-6">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold text-white font-['Outfit'] tracking-tight">
              {cleanName}
            </h1>

            {/* Status Pill */}
            <div
              className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-semibold border ${
                isRunning
                  ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                  : 'bg-slate-800 text-slate-400 border-slate-700'
              }`}
            >
              <span
                className={`w-2 h-2 rounded-full ${
                  isRunning ? 'bg-emerald-400 animate-pulse' : 'bg-slate-500'
                }`}
              />
              <span className="capitalize">{inspect.State.Status}</span>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-4 text-xs text-slate-400 mt-2">
            <span className="font-mono">ID: {inspect.Id.slice(0, 12)}</span>
            <span>&bull;</span>
            <span className="font-mono">Image: {inspect.Config?.Image}</span>
            {projectMount && (
              <>
                <span>&bull;</span>
                <span className="flex items-center gap-1 font-mono text-amber-400/90">
                  <Folder className="w-3.5 h-3.5" />
                  <span>{projectMount.Source}</span>
                </span>
              </>
            )}
          </div>
        </div>

        {/* Action Toolbar */}
        {isAdmin && (
          <div className="flex items-center gap-2">
            {isRunning ? (
              <button
                onClick={handleStop}
                disabled={actionLoading}
                className="flex items-center gap-2 px-3.5 py-2 rounded-xl text-xs font-semibold bg-amber-500/10 text-amber-400 border border-amber-500/20 hover:bg-amber-500/20 transition disabled:opacity-50"
              >
                <Square className="w-3.5 h-3.5" />
                <span>Stop Sandbox</span>
              </button>
            ) : (
              <button
                onClick={handleStart}
                disabled={actionLoading}
                className="flex items-center gap-2 px-3.5 py-2 rounded-xl text-xs font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 hover:bg-emerald-500/20 transition disabled:opacity-50"
              >
                <Play className="w-3.5 h-3.5" />
                <span>Start Sandbox</span>
              </button>
            )}

            <button
              onClick={() => {
                setDuplicateName(`${cleanName}-clone`);
                setIsDuplicateOpen(true);
              }}
              className="flex items-center gap-2 px-3.5 py-2 rounded-xl text-xs font-semibold bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700 transition"
            >
              <Copy className="w-3.5 h-3.5" />
              <span>Clone</span>
            </button>

            <button
              onClick={() => setIsDeleteOpen(true)}
              className="flex items-center gap-2 px-3.5 py-2 rounded-xl text-xs font-semibold bg-rose-500/10 hover:bg-rose-500/20 text-rose-400 border border-rose-500/20 transition"
            >
              <Trash2 className="w-3.5 h-3.5" />
              <span>Destroy</span>
            </button>
          </div>
        )}
      </div>

      {/* Tabs Switcher */}
      <div className="flex items-center gap-2 border-b border-slate-800 pb-2">
        <button
          onClick={() => setActiveTab('terminal')}
          className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-semibold transition ${
            activeTab === 'terminal'
              ? 'bg-blue-600 text-white shadow-md shadow-blue-500/20'
              : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'
          }`}
        >
          <Terminal className="w-4 h-4" />
          <span>Interactive Shell</span>
        </button>

        <button
          onClick={() => setActiveTab('logs')}
          className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-semibold transition ${
            activeTab === 'logs'
              ? 'bg-blue-600 text-white shadow-md shadow-blue-500/20'
              : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'
          }`}
        >
          <FileText className="w-4 h-4" />
          <span>Container Logs</span>
        </button>

        <button
          onClick={() => setActiveTab('inspect')}
          className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-semibold transition ${
            activeTab === 'inspect'
              ? 'bg-blue-600 text-white shadow-md shadow-blue-500/20'
              : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'
          }`}
        >
          <Info className="w-4 h-4" />
          <span>Inspect & Specs</span>
        </button>
      </div>

      {/* Tab 1: Terminal */}
      {activeTab === 'terminal' && (
        <div className="space-y-3">
          {!isRunning && (
            <div className="p-4 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-300 text-xs flex items-center justify-between">
              <span>This sandbox is currently stopped. Start it to initiate an interactive shell session.</span>
              {isAdmin && (
                <button
                  onClick={handleStart}
                  className="px-3 py-1 rounded-lg bg-amber-500/20 hover:bg-amber-500/30 font-semibold"
                >
                  Start Now
                </button>
              )}
            </div>
          )}
          <TerminalView containerId={containerId} containerName={cleanName} />
        </div>
      )}

      {/* Tab 2: Logs */}
      {activeTab === 'logs' && (
        <div className="space-y-4">
          {/* Logs Controls */}
          <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 bg-[#111827] border border-slate-800 p-3 rounded-xl">
            <div className="flex items-center gap-3">
              <input
                type="text"
                placeholder="Filter logs..."
                value={logFilter}
                onChange={(e) => setLogFilter(e.target.value)}
                className="px-3 py-1.5 bg-slate-900 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none"
              />

              <select
                value={logTail}
                onChange={(e) => setLogTail(Number(e.target.value))}
                className="px-3 py-1.5 bg-slate-900 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none"
              >
                <option value={50}>Last 50 lines</option>
                <option value={100}>Last 100 lines</option>
                <option value={200}>Last 200 lines</option>
                <option value={500}>Last 500 lines</option>
              </select>
            </div>

            <div className="flex items-center gap-3">
              <label className="flex items-center gap-2 text-xs text-slate-400 cursor-pointer">
                <input
                  type="checkbox"
                  checked={autoRefreshLogs}
                  onChange={(e) => setAutoRefreshLogs(e.target.checked)}
                  className="rounded text-blue-500 focus:ring-blue-500"
                />
                <span>Auto-refresh (3s)</span>
              </label>

              <button
                onClick={fetchLogs}
                disabled={logsLoading}
                className="p-1.5 rounded-lg bg-slate-900 hover:bg-slate-800 border border-slate-800 text-slate-300"
                title="Refresh Logs"
              >
                <RefreshCw className={`w-3.5 h-3.5 ${logsLoading ? 'animate-spin text-blue-400' : ''}`} />
              </button>
            </div>
          </div>

          {/* Logs viewer */}
          <div className="bg-[#0a0d14] border border-slate-800 rounded-xl p-4 font-mono text-xs text-slate-300 h-[500px] overflow-y-auto whitespace-pre-wrap leading-relaxed">
            {filteredLogs}
          </div>
        </div>
      )}

      {/* Tab 3: Inspect / Config */}
      {activeTab === 'inspect' && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="bg-[#111827] border border-slate-800 rounded-2xl p-6 space-y-4">
            <h3 className="text-sm font-semibold text-slate-200 flex items-center gap-2">
              <Cpu className="w-4 h-4 text-blue-400" />
              <span>Container Configuration</span>
            </h3>

            <div className="space-y-3 text-xs">
              <div className="flex justify-between py-1.5 border-b border-slate-800">
                <span className="text-slate-400">Created At</span>
                <span className="text-slate-200 font-mono">
                  {new Date(inspect.Created).toLocaleString()}
                </span>
              </div>
              <div className="flex justify-between py-1.5 border-b border-slate-800">
                <span className="text-slate-400">Exit Code</span>
                <span className="text-slate-200 font-mono">{inspect.State.ExitCode}</span>
              </div>
              <div className="flex justify-between py-1.5 border-b border-slate-800">
                <span className="text-slate-400">Memory Limit</span>
                <span className="text-slate-200 font-mono">
                  {inspect.HostConfig?.Memory
                    ? `${(inspect.HostConfig.Memory / (1024 * 1024)).toFixed(0)} MB`
                    : 'Unlimited'}
                </span>
              </div>
              <div className="flex justify-between py-1.5 border-b border-slate-800">
                <span className="text-slate-400">Workspace Mount</span>
                <span className="text-slate-200 font-mono truncate max-w-[200px]" title={projectMount?.Source}>
                  {projectMount?.Source || 'None'}
                </span>
              </div>
            </div>

            {/* Labels */}
            {inspect.Config?.Labels && (
              <div className="pt-2">
                <span className="text-xs font-semibold text-slate-400 block mb-2">Labels</span>
                <div className="space-y-1">
                  {Object.entries(inspect.Config.Labels).map(([k, v]) => (
                    <div
                      key={k}
                      className="p-2 rounded bg-slate-900 border border-slate-800/60 font-mono text-[11px] text-slate-300 truncate"
                    >
                      <span className="text-blue-400">{k}:</span> {v}
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>

          {/* Raw JSON */}
          <div className="bg-[#111827] border border-slate-800 rounded-2xl p-6 flex flex-col">
            <h3 className="text-sm font-semibold text-slate-200 mb-3 flex items-center gap-2">
              <Info className="w-4 h-4 text-indigo-400" />
              <span>Full Libpod JSON</span>
            </h3>
            <pre className="flex-1 p-3 rounded-xl bg-[#0a0d14] border border-slate-800 font-mono text-[11px] text-slate-300 overflow-auto max-h-[450px]">
              {JSON.stringify(inspect, null, 2)}
            </pre>
          </div>
        </div>
      )}

      {/* Duplicate Modal */}
      <Modal
        isOpen={isDuplicateOpen}
        onClose={() => setIsDuplicateOpen(false)}
        title="Duplicate Sandbox Container"
      >
        <form onSubmit={handleDuplicate} className="space-y-4">
          <div>
            <label className="block text-xs font-medium text-slate-300 mb-1.5">
              New Sandbox Name
            </label>
            <input
              type="text"
              required
              value={duplicateName}
              onChange={(e) => setDuplicateName(e.target.value)}
              className="w-full px-3.5 py-2.5 bg-slate-900 border border-slate-800 rounded-xl text-sm text-slate-200 focus:outline-none"
            />
          </div>

          <div className="flex items-center justify-end gap-3 pt-3">
            <button
              type="button"
              onClick={() => setIsDuplicateOpen(false)}
              className="px-4 py-2 rounded-xl text-sm text-slate-400 hover:text-slate-200"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={duplicateLoading}
              className="flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-semibold bg-blue-600 hover:bg-blue-500 text-white"
            >
              {duplicateLoading ? 'Cloning...' : 'Clone Container'}
            </button>
          </div>
        </form>
      </Modal>

      {/* Delete Modal */}
      <Modal
        isOpen={isDeleteOpen}
        onClose={() => setIsDeleteOpen(false)}
        title="Destroy Sandbox Container"
      >
        <div className="space-y-4">
          <p className="text-sm text-slate-300">
            Permanently stop and delete container <span className="font-mono font-bold text-rose-400">{cleanName}</span>?
          </p>
          <div className="flex items-center justify-end gap-3 pt-3">
            <button
              type="button"
              onClick={() => setIsDeleteOpen(false)}
              className="px-4 py-2 rounded-xl text-sm text-slate-400"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={handleDelete}
              disabled={deleteLoading}
              className="px-4 py-2 rounded-xl text-sm font-semibold bg-rose-600 text-white"
            >
              {deleteLoading ? 'Destroying...' : 'Confirm Destroy'}
            </button>
          </div>
        </div>
      </Modal>
    </div>
  );
};
