import React, { useState, useEffect, useCallback } from 'react';
import { Sparkles, Plus, Pencil, Trash2, Shield, Users, Settings2, ShieldAlert, Package } from 'lucide-react';
import { apiRequest } from '../api';
import { Modal } from '../components/Modal';
import { AuthUser, MCPRegistryTool, MCPToolsResponse } from '../types';

interface MCPAccessCenterPageProps {
  user: AuthUser;
}

const emptyForm = { name: '', command: '', args: '', description: '', level: 'user' as 'admin' | 'user' };

export const MCPAccessCenterPage: React.FC<MCPAccessCenterPageProps> = ({ user }) => {
  const [tools, setTools] = useState<MCPRegistryTool[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Editor modal
  const [editorOpen, setEditorOpen] = useState(false);
  const [editing, setEditing] = useState<MCPRegistryTool | null>(null);
  const [form, setForm] = useState(emptyForm);
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const isAdmin = user.role === 'admin';

  const loadTools = useCallback(async () => {
    setLoading(true);
    try {
      const res = await apiRequest<MCPToolsResponse>('/api/mcp-tools');
      setTools(res.tools || []);
      setError(null);
    } catch (err: any) {
      setError(err.message || 'Failed to load MCP tools');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadTools();
  }, [loadTools]);

  const canManage = (t: MCPRegistryTool) => isAdmin || (!t.builtIn && t.createdBy === user.username);

  const openCreate = () => {
    setEditing(null);
    setForm({ ...emptyForm, level: isAdmin ? 'user' : 'user' });
    setFormError(null);
    setEditorOpen(true);
  };

  const openEdit = (t: MCPRegistryTool) => {
    setEditing(t);
    setForm({
      name: t.name,
      command: t.command,
      args: t.args.join(' '),
      description: t.description,
      level: t.level,
    });
    setFormError(null);
    setEditorOpen(true);
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setFormError(null);
    try {
      const payload = {
        name: form.name.trim(),
        command: form.command.trim(),
        args: form.args.split(' ').filter(Boolean),
        level: form.level,
        description: form.description.trim(),
      };
      if (editing) {
        await apiRequest(`/api/mcp-tools/${editing.id}`, {
          method: 'PUT',
          body: JSON.stringify(payload),
        });
      } else {
        await apiRequest('/api/mcp-tools', {
          method: 'POST',
          body: JSON.stringify(payload),
        });
      }
      setEditorOpen(false);
      await loadTools();
    } catch (err: any) {
      setFormError(err.message || 'Failed to save tool');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (t: MCPRegistryTool) => {
    if (!window.confirm(`Delete MCP tool "${t.name}"? Containers created later will no longer offer it.`)) {
      return;
    }
    try {
      await apiRequest(`/api/mcp-tools/${t.id}`, { method: 'DELETE' });
      await loadTools();
    } catch (err: any) {
      setError(err.message || 'Failed to delete tool');
    }
  };

  const adminCount = tools.filter((t) => t.level === 'admin').length;
  const userCount = tools.filter((t) => t.level === 'user').length;

  return (
    <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-8 animate-in fade-in duration-300">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-4 mb-8">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-white font-['Outfit'] flex items-center gap-3">
            <Sparkles className="w-7 h-7 text-indigo-400" />
            MCP Access Center
          </h1>
          <p className="text-sm text-slate-400 mt-1">
            Custom MCP server tools with access levels — attach them to any sandbox from the creation wizard.
          </p>
        </div>
        <button
          onClick={openCreate}
          className="flex items-center gap-2 px-5 py-2.5 rounded-xl text-sm font-semibold bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white shadow-lg shadow-blue-500/25 transition active:scale-95"
        >
          <Plus className="w-4 h-4" />
          New MCP Tool
        </button>
      </div>

      {/* Level legend */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 mb-6">
        <div className="flex items-start gap-3 p-4 rounded-2xl bg-rose-500/5 border border-rose-500/20">
          <Shield className="w-5 h-5 text-rose-400 flex-shrink-0 mt-0.5" />
          <div>
            <span className="text-sm font-semibold text-slate-200">Admin Level ({adminCount})</span>
            <p className="text-xs text-slate-400 mt-0.5">
              Visible and attachable by admins only. Only admins can create or edit these.
            </p>
          </div>
        </div>
        <div className="flex items-start gap-3 p-4 rounded-2xl bg-emerald-500/5 border border-emerald-500/20">
          <Users className="w-5 h-5 text-emerald-400 flex-shrink-0 mt-0.5" />
          <div>
            <span className="text-sm font-semibold text-slate-200">User Level ({userCount})</span>
            <p className="text-xs text-slate-400 mt-0.5">
              Available to every role. Users can create, edit, and delete their own user-level tools.
            </p>
          </div>
        </div>
      </div>

      {error && (
        <div className="mb-6 flex items-center gap-3 p-4 rounded-2xl bg-rose-500/10 border border-rose-500/20 text-rose-300 text-sm">
          <ShieldAlert className="w-5 h-5 flex-shrink-0" />
          <p>{error}</p>
        </div>
      )}

      {/* Tool grid */}
      {loading ? (
        <div className="flex items-center justify-center py-20 text-slate-500">
          <div className="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
        </div>
      ) : tools.length === 0 ? (
        <div className="py-20 text-center text-slate-500 text-sm">
          No MCP tools registered yet. Create your first one above.
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
          {tools.map((t) => (
            <div
              key={t.id}
              className={`p-4 rounded-2xl border bg-[#111827]/90 transition hover:border-slate-700 ${
                t.level === 'admin' ? 'border-rose-500/20' : 'border-emerald-500/20'
              }`}
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="font-semibold text-sm text-slate-100">{t.name}</span>
                    <span
                      className={`text-[10px] uppercase font-bold px-1.5 py-0.5 rounded border flex items-center gap-1 ${
                        t.level === 'admin'
                          ? 'bg-rose-500/15 text-rose-300 border-rose-500/30'
                          : 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30'
                      }`}
                    >
                      {t.level === 'admin' ? <Shield className="w-2.5 h-2.5" /> : <Users className="w-2.5 h-2.5" />}
                      {t.level}
                    </span>
                    {t.builtIn && (
                      <span className="text-[10px] uppercase font-bold px-1.5 py-0.5 rounded bg-slate-800 text-slate-400 border border-slate-700 flex items-center gap-1">
                        <Package className="w-2.5 h-2.5" />
                        built-in
                      </span>
                    )}
                  </div>
                  <p className="text-[13px] font-mono text-indigo-300/90 mt-1.5 truncate" title={`${t.command} ${t.args.join(' ')}`}>
                    {t.command} {t.args.join(' ')}
                  </p>
                  {t.description && <p className="text-xs text-slate-400 mt-1">{t.description}</p>}
                  <p className="text-[11px] text-slate-600 mt-1.5">
                    by {t.createdBy} · {t.createdAt ? new Date(t.createdAt).toLocaleString() : '—'}
                  </p>
                </div>
                {canManage(t) && (
                  <div className="flex gap-1 flex-shrink-0">
                    <button
                      onClick={() => openEdit(t)}
                      title="Edit tool"
                      className="p-2 rounded-lg text-slate-400 hover:text-blue-300 hover:bg-blue-500/10 transition"
                    >
                      <Pencil className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => handleDelete(t)}
                      title="Delete tool"
                      className="p-2 rounded-lg text-slate-400 hover:text-rose-400 hover:bg-rose-500/10 transition"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Create / Edit modal */}
      <Modal
        isOpen={editorOpen}
        onClose={() => setEditorOpen(false)}
        title={editing ? `Edit MCP Tool — ${editing.name}` : 'New MCP Tool'}
      >
        <form onSubmit={handleSave} className="space-y-4">
          {formError && (
            <div className="p-3 rounded-lg bg-rose-500/10 border border-rose-500/20 text-xs text-rose-300">
              {formError}
            </div>
          )}

          <div>
            <label className="block text-xs font-medium text-slate-300 mb-1.5">Tool Name</label>
            <input
              type="text"
              required
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="e.g. ros2-mcp"
              className="w-full px-3.5 py-2.5 bg-slate-900 border border-slate-800 rounded-xl text-sm text-slate-200 font-mono focus:outline-none focus:ring-2 focus:ring-blue-500/40"
            />
          </div>

          <div>
            <label className="block text-xs font-medium text-slate-300 mb-1.5">Command</label>
            <input
              type="text"
              required
              value={form.command}
              onChange={(e) => setForm({ ...form, command: e.target.value })}
              placeholder="e.g. npx or python3"
              className="w-full px-3.5 py-2.5 bg-slate-900 border border-slate-800 rounded-xl text-sm text-slate-200 font-mono focus:outline-none focus:ring-2 focus:ring-blue-500/40"
            />
          </div>

          <div>
            <label className="block text-xs font-medium text-slate-300 mb-1.5">Arguments (space-separated)</label>
            <input
              type="text"
              value={form.args}
              onChange={(e) => setForm({ ...form, args: e.target.value })}
              placeholder="-y @modelcontextprotocol/server-filesystem /workspace"
              className="w-full px-3.5 py-2.5 bg-slate-900 border border-slate-800 rounded-xl text-sm text-slate-200 font-mono focus:outline-none focus:ring-2 focus:ring-blue-500/40"
            />
          </div>

          <div>
            <label className="block text-xs font-medium text-slate-300 mb-1.5">Description</label>
            <input
              type="text"
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
              placeholder="What this MCP server does"
              className="w-full px-3.5 py-2.5 bg-slate-900 border border-slate-800 rounded-xl text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500/40"
            />
          </div>

          <div>
            <label className="block text-xs font-medium text-slate-300 mb-1.5">Access Level</label>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => setForm({ ...form, level: 'user' })}
                className={`flex-1 flex items-center justify-center gap-2 px-3 py-2.5 rounded-xl text-sm font-medium transition ${
                  form.level === 'user'
                    ? 'bg-emerald-500/15 border border-emerald-500/50 text-emerald-300'
                    : 'bg-slate-900 border border-slate-800 text-slate-400 hover:text-slate-200'
                }`}
              >
                <Users className="w-4 h-4" />
                User Level
              </button>
              <button
                type="button"
                disabled={!isAdmin}
                title={isAdmin ? 'Admin-only tool' : 'Admins only can create admin-level tools'}
                onClick={() => isAdmin && setForm({ ...form, level: 'admin' })}
                className={`flex-1 flex items-center justify-center gap-2 px-3 py-2.5 rounded-xl text-sm font-medium transition ${
                  form.level === 'admin'
                    ? 'bg-rose-500/15 border border-rose-500/50 text-rose-300'
                    : 'bg-slate-900 border border-slate-800 text-slate-400 hover:text-slate-200'
                } ${!isAdmin ? 'opacity-40 cursor-not-allowed' : ''}`}
              >
                <Shield className="w-4 h-4" />
                Admin Level
              </button>
            </div>
            <p className="text-[11px] text-slate-500 mt-1.5 flex items-center gap-1">
              <Settings2 className="w-3 h-3" />
              Admin-level tools are hidden from non-admins everywhere — including sandbox configuration.
            </p>
          </div>

          <div className="flex justify-end gap-3 pt-2">
            <button
              type="button"
              onClick={() => setEditorOpen(false)}
              className="px-4 py-2.5 rounded-xl text-sm text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={saving}
              className="flex items-center gap-2 px-5 py-2.5 rounded-xl text-sm font-semibold bg-blue-600 hover:bg-blue-500 text-white transition disabled:opacity-50"
            >
              {saving ? (
                <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
              ) : (
                <>
                  <Sparkles className="w-4 h-4" />
                  {editing ? 'Save Changes' : 'Create Tool'}
                </>
              )}
            </button>
          </div>
        </form>
      </Modal>
    </div>
  );
};
