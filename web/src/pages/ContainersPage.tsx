import React, { useState, useEffect } from 'react';
import { Box, Play, Square, RefreshCw, Search, Plus, HardDrive, Cpu, AlertCircle, Copy, Trash2 } from 'lucide-react';
import { ContainerSummary, AuthUser } from '../types';
import { apiRequest } from '../api';
import { ContainerCard } from '../components/ContainerCard';
import { Modal } from '../components/Modal';

interface ContainersPageProps {
  user: AuthUser | null;
}

export const ContainersPage: React.FC<ContainersPageProps> = ({ user }) => {
  const [containers, setContainers] = useState<ContainerSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Search & Filter
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | 'running' | 'stopped'>('all');

  // Modals state
  const [duplicateTarget, setDuplicateTarget] = useState<ContainerSummary | null>(null);
  const [duplicateName, setDuplicateName] = useState('');
  const [duplicateLoading, setDuplicateLoading] = useState(false);

  const [deleteTarget, setDeleteTarget] = useState<ContainerSummary | null>(null);
  const [deleteLoading, setDeleteLoading] = useState(false);

  const isAdmin = user?.role === 'admin';

  const fetchContainers = async (showRefresh = false) => {
    if (showRefresh) setRefreshing(true);
    else setLoading(true);
    setError(null);

    try {
      const data = await apiRequest<ContainerSummary[]>('/api/containers');
      setContainers(data || []);
    } catch (err: any) {
      setError(err.message || 'Failed to load containers');
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  useEffect(() => {
    fetchContainers();
  }, []);

  const handleStartContainer = async (id: string) => {
    try {
      await apiRequest(`/api/containers/${id}/start`, { method: 'POST' });
      await fetchContainers(true);
    } catch (err: any) {
      alert(`Start failed: ${err.message}`);
    }
  };

  const handleStopContainer = async (id: string) => {
    try {
      await apiRequest(`/api/containers/${id}/stop`, { method: 'POST' });
      await fetchContainers(true);
    } catch (err: any) {
      alert(`Stop failed: ${err.message}`);
    }
  };

  const openDuplicateModal = (c: ContainerSummary) => {
    const rawName = c.Names && c.Names[0] ? c.Names[0].replace(/^\//, '') : c.Id.slice(0, 8);
    setDuplicateTarget(c);
    setDuplicateName(`${rawName}-copy`);
  };

  const handleConfirmDuplicate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!duplicateTarget) return;
    setDuplicateLoading(true);
    try {
      await apiRequest(`/api/containers/${duplicateTarget.Id}/duplicate`, {
        method: 'POST',
        body: JSON.stringify({ newName: duplicateName }),
      });
      setDuplicateTarget(null);
      await fetchContainers(true);
    } catch (err: any) {
      alert(`Duplicate failed: ${err.message}`);
    } finally {
      setDuplicateLoading(false);
    }
  };

  const openDeleteModal = (c: ContainerSummary) => {
    setDeleteTarget(c);
  };

  const handleConfirmDelete = async () => {
    if (!deleteTarget) return;
    setDeleteLoading(true);
    try {
      await apiRequest(`/api/containers/${deleteTarget.Id}`, { method: 'DELETE' });
      setDeleteTarget(null);
      await fetchContainers(true);
    } catch (err: any) {
      alert(`Delete failed: ${err.message}`);
    } finally {
      setDeleteLoading(false);
    }
  };

  // Filtered containers
  const filtered = containers.filter((c) => {
    const name = (c.Names?.[0] || '').toLowerCase();
    const id = c.Id.toLowerCase();
    const img = c.Image.toLowerCase();
    const matchesSearch = name.includes(searchQuery.toLowerCase()) || id.includes(searchQuery.toLowerCase()) || img.includes(searchQuery.toLowerCase());

    const isRunning = c.State.toLowerCase() === 'running';
    if (statusFilter === 'running') return matchesSearch && isRunning;
    if (statusFilter === 'stopped') return matchesSearch && !isRunning;
    return matchesSearch;
  });

  const totalRunning = containers.filter((c) => c.State.toLowerCase() === 'running').length;
  const totalStopped = containers.length - totalRunning;
  const mappedProjects = new Set(containers.map((c) => c.projectDir).filter(Boolean)).size;

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8 animate-in fade-in duration-300">
      {/* Top Title & Stats Banner */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-white font-['Outfit'] flex items-center gap-3">
            <span>Agent Sandboxes</span>
            <span className="text-xs font-semibold px-2.5 py-1 rounded-full bg-blue-500/10 text-blue-400 border border-blue-500/20">
              Podman Libpod
            </span>
          </h1>
          <p className="text-slate-400 text-sm mt-1">
            Isolated Ubuntu environments executing MCP tools and autonomous computer control.
          </p>
        </div>

        {isAdmin && (
          <a
            href="#/containers/new"
            className="self-start md:self-auto flex items-center gap-2 px-4 py-2.5 rounded-xl font-semibold text-sm bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white shadow-lg shadow-blue-500/25 transition active:scale-95"
          >
            <Plus className="w-4 h-4" />
            <span>Spin Up New Sandbox</span>
          </a>
        )}
      </div>

      {/* Metrics Row */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="bg-[#111827]/80 border border-slate-800 rounded-2xl p-4 flex items-center gap-4">
          <div className="w-12 h-12 rounded-xl bg-blue-500/10 border border-blue-500/20 flex items-center justify-center text-blue-400">
            <Box className="w-6 h-6" />
          </div>
          <div>
            <p className="text-xs text-slate-400">Total Sandboxes</p>
            <p className="text-2xl font-bold text-slate-100 font-mono">{containers.length}</p>
          </div>
        </div>

        <div className="bg-[#111827]/80 border border-slate-800 rounded-2xl p-4 flex items-center gap-4">
          <div className="w-12 h-12 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400">
            <Play className="w-6 h-6" />
          </div>
          <div>
            <p className="text-xs text-slate-400">Active / Running</p>
            <p className="text-2xl font-bold text-emerald-400 font-mono">{totalRunning}</p>
          </div>
        </div>

        <div className="bg-[#111827]/80 border border-slate-800 rounded-2xl p-4 flex items-center gap-4">
          <div className="w-12 h-12 rounded-xl bg-slate-800 border border-slate-700 flex items-center justify-center text-slate-400">
            <Square className="w-6 h-6" />
          </div>
          <div>
            <p className="text-xs text-slate-400">Stopped</p>
            <p className="text-2xl font-bold text-slate-300 font-mono">{totalStopped}</p>
          </div>
        </div>

        <div className="bg-[#111827]/80 border border-slate-800 rounded-2xl p-4 flex items-center gap-4">
          <div className="w-12 h-12 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400">
            <HardDrive className="w-6 h-6" />
          </div>
          <div>
            <p className="text-xs text-slate-400">Mapped Projects</p>
            <p className="text-2xl font-bold text-indigo-300 font-mono">{mappedProjects}</p>
          </div>
        </div>
      </div>

      {/* Filter and Search Bar */}
      <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 bg-[#111827]/60 border border-slate-800/80 p-3 rounded-2xl">
        <div className="relative flex-1">
          <Search className="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-500" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search by container name, ID, or image..."
            className="w-full pl-10 pr-4 py-2 bg-slate-900/90 border border-slate-800 rounded-xl text-sm text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500/40"
          />
        </div>

        <div className="flex items-center gap-2">
          {/* Status filter tabs */}
          <div className="flex items-center p-1 bg-slate-900/90 border border-slate-800 rounded-xl text-xs">
            {(['all', 'running', 'stopped'] as const).map((tab) => (
              <button
                key={tab}
                onClick={() => setStatusFilter(tab)}
                className={`px-3 py-1.5 rounded-lg capitalize font-medium transition ${
                  statusFilter === tab
                    ? 'bg-blue-600 text-white shadow-sm'
                    : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                {tab}
              </button>
            ))}
          </div>

          {/* Refresh button */}
          <button
            onClick={() => fetchContainers(true)}
            disabled={refreshing}
            className="p-2.5 rounded-xl bg-slate-900/90 hover:bg-slate-800 border border-slate-800 text-slate-300 hover:text-white transition disabled:opacity-50"
            title="Refresh Container List"
          >
            <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin text-blue-400' : ''}`} />
          </button>
        </div>
      </div>

      {/* Error Notice */}
      {error && (
        <div className="flex items-center gap-3 p-4 rounded-2xl bg-rose-500/10 border border-rose-500/20 text-rose-300 text-sm">
          <AlertCircle className="w-5 h-5 flex-shrink-0" />
          <div className="flex-1">
            <p className="font-semibold">Unable to fetch containers from Podman socket</p>
            <p className="text-xs text-rose-300/80 mt-0.5">{error}</p>
          </div>
          <button
            onClick={() => fetchContainers(true)}
            className="px-3 py-1.5 rounded-lg bg-rose-500/20 hover:bg-rose-500/30 text-rose-200 text-xs font-semibold"
          >
            Retry
          </button>
        </div>
      )}

      {/* Grid or Empty State */}
      {loading ? (
        <div className="py-24 flex flex-col items-center justify-center gap-3 text-slate-500">
          <div className="w-8 h-8 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
          <p className="text-sm">Querying Podman daemon...</p>
        </div>
      ) : filtered.length === 0 ? (
        <div className="py-20 border border-dashed border-slate-800 rounded-3xl flex flex-col items-center justify-center p-8 text-center bg-[#0d121f]/50">
          <div className="w-16 h-16 rounded-2xl bg-slate-900 border border-slate-800 flex items-center justify-center text-slate-500 mb-4">
            <Box className="w-8 h-8" />
          </div>
          <h3 className="text-lg font-semibold text-slate-200">No Containers Found</h3>
          <p className="text-sm text-slate-400 max-w-sm mt-1 mb-6">
            {searchQuery
              ? 'No agent sandboxes match your search query or filter.'
              : 'No agent sandboxes are currently deployed in your local Podman environment.'}
          </p>
          {isAdmin && (
            <a
              href="#/containers/new"
              className="flex items-center gap-2 px-4 py-2.5 rounded-xl font-semibold text-sm bg-blue-600 hover:bg-blue-500 text-white shadow-lg shadow-blue-500/25 transition"
            >
              <Plus className="w-4 h-4" />
              <span>Spin Up First Sandbox</span>
            </a>
          )}
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filtered.map((c) => (
            <ContainerCard
              key={c.Id}
              container={c}
              isAdmin={isAdmin}
              onStart={handleStartContainer}
              onStop={handleStopContainer}
              onDuplicate={openDuplicateModal}
              onDelete={openDeleteModal}
            />
          ))}
        </div>
      )}

      {/* Duplicate Modal */}
      <Modal
        isOpen={!!duplicateTarget}
        onClose={() => setDuplicateTarget(null)}
        title="Duplicate Sandbox Container"
      >
        <form onSubmit={handleConfirmDuplicate} className="space-y-4">
          <p className="text-xs text-slate-400">
            This will snapshot the current container's filesystem and create a new isolated sibling
            sandbox running with identical MCP tools.
          </p>

          <div>
            <label className="block text-xs font-medium text-slate-300 mb-1.5">
              New Sandbox Name
            </label>
            <input
              type="text"
              required
              value={duplicateName}
              onChange={(e) => setDuplicateName(e.target.value)}
              className="w-full px-3.5 py-2.5 bg-slate-900 border border-slate-800 rounded-xl text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500/40"
            />
          </div>

          <div className="flex items-center justify-end gap-3 pt-3">
            <button
              type="button"
              onClick={() => setDuplicateTarget(null)}
              className="px-4 py-2 rounded-xl text-sm text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={duplicateLoading}
              className="flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-semibold bg-blue-600 hover:bg-blue-500 text-white shadow-lg shadow-blue-500/25 transition disabled:opacity-50"
            >
              {duplicateLoading ? (
                <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
              ) : (
                <>
                  <Copy className="w-4 h-4" />
                  <span>Clone Container</span>
                </>
              )}
            </button>
          </div>
        </form>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        title="Destroy Sandbox Container"
      >
        <div className="space-y-4">
          <p className="text-sm text-slate-300">
            Are you sure you want to stop and permanently destroy sandbox{' '}
            <span className="font-mono font-bold text-rose-400">
              {deleteTarget?.Names?.[0] || deleteTarget?.Id.slice(0, 12)}
            </span>
            ?
          </p>
          <p className="text-xs text-slate-400">
            Host files mapped under <code className="text-slate-300 font-mono">/workspace</code> will
            remain untouched on your host machine.
          </p>

          <div className="flex items-center justify-end gap-3 pt-3">
            <button
              type="button"
              onClick={() => setDeleteTarget(null)}
              className="px-4 py-2 rounded-xl text-sm text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={handleConfirmDelete}
              disabled={deleteLoading}
              className="flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-semibold bg-rose-600 hover:bg-rose-500 text-white shadow-lg shadow-rose-600/25 transition disabled:opacity-50"
            >
              {deleteLoading ? (
                <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
              ) : (
                <>
                  <Trash2 className="w-4 h-4" />
                  <span>Destroy Sandbox</span>
                </>
              )}
            </button>
          </div>
        </div>
      </Modal>
    </div>
  );
};
