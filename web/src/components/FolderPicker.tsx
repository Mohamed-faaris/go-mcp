import React, { useState, useEffect, useCallback } from 'react';
import { Folder, FolderOpen, ArrowUp, ChevronRight, Check, Loader2, FolderTree } from 'lucide-react';
import { apiRequest } from '../api';
import { Modal } from './Modal';
import { ProjectItem } from '../types';

interface BrowseResponse {
  path: string;
  parent: string;
  root: 'projects' | 'tmp';
  items: ProjectItem[];
}

interface FolderPickerProps {
  isOpen: boolean;
  onClose: () => void;
  onSelect: (path: string) => void;
  projectsDir: string;
  tmpDir: string;
  /** Where to start browsing; defaults to the projects root */
  startPath?: string;
  title?: string;
}

export const FolderPicker: React.FC<FolderPickerProps> = ({
  isOpen,
  onClose,
  onSelect,
  projectsDir,
  tmpDir,
  startPath,
  title = 'Pick a Folder',
}) => {
  const [path, setPath] = useState('');
  const [parent, setParent] = useState('');
  const [root, setRoot] = useState<'projects' | 'tmp'>('projects');
  const [items, setItems] = useState<ProjectItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async (p: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await apiRequest<BrowseResponse>(
        `/api/projects/browse?path=${encodeURIComponent(p)}`
      );
      setPath(res.path);
      setParent(res.parent);
      setRoot(res.root);
      setItems(res.items);
    } catch (err: any) {
      setError(err.message || 'Failed to read folder');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (isOpen) {
      load(startPath || projectsDir);
    }
  }, [isOpen, startPath, projectsDir, load]);

  // Breadcrumb segments below the active root
  const activeRoot = root === 'tmp' ? tmpDir : projectsDir;
  const relative = path.startsWith(activeRoot) ? path.slice(activeRoot.length) : '';
  const segments = relative.split('/').filter(Boolean);

  const segmentPath = (idx: number) =>
    `${activeRoot}/${segments.slice(0, idx + 1).join('/')}`;

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={title} maxWidth="max-w-2xl">
      {/* Root tabs */}
      <div className="flex gap-2 mb-3">
        <button
          type="button"
          onClick={() => load(projectsDir)}
          className={`px-3 py-1.5 rounded-lg text-xs font-medium transition ${
            root === 'projects'
              ? 'bg-blue-600 text-white'
              : 'bg-slate-900 border border-slate-800 text-slate-400 hover:text-slate-200'
          }`}
        >
          <FolderTree className="w-3.5 h-3.5 inline mr-1.5 -mt-0.5" />
          Host Projects
        </button>
        <button
          type="button"
          onClick={() => load(tmpDir)}
          className={`px-3 py-1.5 rounded-lg text-xs font-medium transition ${
            root === 'tmp'
              ? 'bg-amber-500 text-slate-950'
              : 'bg-slate-900 border border-slate-800 text-slate-400 hover:text-slate-200'
          }`}
        >
          Temporary Scratch
        </button>
        {parent && (
          <button
            type="button"
            onClick={() => load(parent)}
            className="px-3 py-1.5 rounded-lg text-xs font-medium bg-slate-900 border border-slate-800 text-slate-400 hover:text-slate-200 transition"
          >
            <ArrowUp className="w-3.5 h-3.5 inline mr-1 -mt-0.5" />
            Up
          </button>
        )}
      </div>

      {/* Breadcrumb */}
      <div className="flex items-center gap-1 flex-wrap text-xs font-mono text-slate-400 bg-slate-900/70 border border-slate-800 rounded-lg px-3 py-2 mb-3">
        <button
          type="button"
          onClick={() => load(activeRoot)}
          className="hover:text-slate-200 transition"
        >
          {root === 'tmp' ? 'scratch' : 'projects'}
        </button>
        {segments.map((seg, i) => (
          <React.Fragment key={`${seg}-${i}`}>
            <ChevronRight className="w-3 h-3 text-slate-600" />
            <button
              type="button"
              onClick={() => load(segmentPath(i))}
              className="hover:text-slate-200 transition"
            >
              {seg}
            </button>
          </React.Fragment>
        ))}
        {loading && <Loader2 className="w-3.5 h-3.5 ml-auto text-blue-400 animate-spin" />}
      </div>

      {error && (
        <div className="mb-3 p-3 rounded-lg bg-rose-500/10 border border-rose-500/20 text-xs text-rose-300">
          {error}
        </div>
      )}

      {/* Folder list */}
      <div className="border border-slate-800 rounded-xl divide-y divide-slate-800/60 max-h-80 overflow-y-auto">
        {items.length === 0 && !loading ? (
          <div className="p-6 text-center text-xs text-slate-500">
            No subfolders in this directory.
          </div>
        ) : (
          items.map((item) => (
            <div
              key={item.path}
              className="flex items-center gap-2 px-3 py-2 hover:bg-slate-900/80 group"
            >
              <button
                type="button"
                onClick={() => load(item.path)}
                className="flex-1 flex items-center gap-2.5 min-w-0 text-left"
              >
                <Folder className="w-4 h-4 text-amber-400 flex-shrink-0" />
                <span className="text-sm text-slate-200 truncate">{item.name}</span>
                {item.modified && (
                  <span className="text-[10px] text-slate-500 flex-shrink-0 ml-2">
                    {item.modified}
                  </span>
                )}
                <FolderOpen className="w-3.5 h-3.5 text-slate-600 opacity-0 group-hover:opacity-100 transition flex-shrink-0" />
              </button>
              <button
                type="button"
                onClick={() => onSelect(item.path)}
                title="Select this folder"
                className="opacity-0 group-hover:opacity-100 focus:opacity-100 text-emerald-400 hover:text-emerald-300 p-1 transition flex-shrink-0"
              >
                <Check className="w-4 h-4" />
              </button>
            </div>
          ))
        )}
      </div>

      {/* Footer */}
      <div className="flex items-center justify-between gap-3 mt-4">
        <p className="text-[11px] font-mono text-slate-500 truncate flex-1" title={path}>
          {path}
        </p>
        <div className="flex gap-2 flex-shrink-0">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 rounded-xl text-xs text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition"
          >
            Cancel
          </button>
          <button
            type="button"
            disabled={!path || loading}
            onClick={() => onSelect(path)}
            className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-xs font-semibold bg-blue-600 hover:bg-blue-500 text-white transition disabled:opacity-50"
          >
            <Check className="w-3.5 h-3.5" />
            Select This Folder
          </button>
        </div>
      </div>
    </Modal>
  );
};
