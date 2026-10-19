import React, { useState, useEffect } from 'react';
import { FolderGit2, Plus, Copy, Folder, Clock, HardDrive, RefreshCw, Box } from 'lucide-react';
import { apiRequest } from '../api';
import { ProjectsResponse, ProjectItem, AuthUser } from '../types';
import { Modal } from '../components/Modal';

interface ProjectsPageProps {
  user: AuthUser | null;
}

export const ProjectsPage: React.FC<ProjectsPageProps> = ({ user }) => {
  const [data, setData] = useState<ProjectsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [cloneSource, setCloneSource] = useState<ProjectItem | null>(null);
  const [targetName, setTargetName] = useState('');
  const [cloneLocation, setCloneLocation] = useState<'projects' | 'tmp'>('projects');
  const [cloneLoading, setCloneLoading] = useState(false);

  const isAdmin = user?.role === 'admin';

  const fetchProjects = async () => {
    setLoading(true);
    try {
      const res = await apiRequest<ProjectsResponse>('/api/projects');
      setData(res);
    } catch (err: any) {
      console.error('Failed to fetch projects:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProjects();
  }, []);

  const handleOpenClone = (item: ProjectItem) => {
    setCloneSource(item);
    setTargetName(`${item.name}-clone`);
    setCloneLocation(item.type === 'tmp' ? 'tmp' : 'projects');
  };

  const handleConfirmClone = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!cloneSource) return;
    setCloneLoading(true);
    try {
      await apiRequest('/api/projects/clone', {
        method: 'POST',
        body: JSON.stringify({
          sourcePath: cloneSource.path,
          targetName,
          location: cloneLocation,
        }),
      });
      setCloneSource(null);
      await fetchProjects();
    } catch (err: any) {
      alert(`Clone failed: ${err.message}`);
    } finally {
      setCloneLoading(false);
    }
  };

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8 animate-in fade-in duration-300">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-white font-['Outfit'] flex items-center gap-3">
            <span>Project Directories</span>
            <span className="text-xs font-semibold px-2.5 py-1 rounded-full bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
              Host Bind Mounts
            </span>
          </h1>
          <p className="text-slate-400 text-sm mt-1">
            Host directories available to be bind-mounted into agent containers at{' '}
            <code className="text-slate-300 font-mono">/workspace</code>.
          </p>
        </div>

        <button
          onClick={fetchProjects}
          className="self-start md:self-auto flex items-center gap-2 px-3.5 py-2 rounded-xl bg-slate-900 hover:bg-slate-800 border border-slate-800 text-slate-300 transition"
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin text-blue-400' : ''}`} />
          <span className="text-xs font-semibold">Rescan Folders</span>
        </button>
      </div>

      {/* Directory Paths Summary */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs font-mono">
        <div className="p-4 rounded-xl bg-[#111827] border border-slate-800 flex items-center gap-3">
          <Folder className="w-5 h-5 text-amber-400 flex-shrink-0" />
          <div className="truncate">
            <span className="text-slate-400 block text-[11px]">Host Projects Path:</span>
            <span className="text-slate-200">{data?.projectsDir || '/host/projects'}</span>
          </div>
        </div>
        <div className="p-4 rounded-xl bg-[#111827] border border-slate-800 flex items-center gap-3">
          <HardDrive className="w-5 h-5 text-indigo-400 flex-shrink-0" />
          <div className="truncate">
            <span className="text-slate-400 block text-[11px]">Scratch Temporary Path:</span>
            <span className="text-slate-200">{data?.tmpDir || '/host/tmp-podmans'}</span>
          </div>
        </div>
      </div>

      {/* List */}
      {loading ? (
        <div className="py-20 flex flex-col items-center justify-center gap-2 text-slate-500">
          <div className="w-7 h-7 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
          <p className="text-xs">Scanning host file system...</p>
        </div>
      ) : !data?.items || data.items.length === 0 ? (
        <div className="py-16 text-center border border-dashed border-slate-800 rounded-2xl bg-[#0d121f]/50 p-6">
          <FolderGit2 className="w-10 h-10 text-slate-600 mx-auto mb-3" />
          <h3 className="text-base font-semibold text-slate-300">No Project Folders Found</h3>
          <p className="text-xs text-slate-500 max-w-sm mx-auto mt-1">
            Place directories in your host projects path or create scratch folders when deploying sandboxes.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {data.items.map((item) => (
            <div
              key={item.path}
              className="bg-[#111827] border border-slate-800 hover:border-slate-700 rounded-2xl p-5 shadow-lg flex flex-col justify-between transition group"
            >
              <div>
                <div className="flex items-start justify-between gap-3 mb-3">
                  <div className="flex items-center gap-2.5">
                    <div
                      className={`w-9 h-9 rounded-xl flex items-center justify-center ${
                        item.type === 'project'
                          ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                          : 'bg-indigo-500/10 text-indigo-400 border border-indigo-500/20'
                      }`}
                    >
                      <Folder className="w-5 h-5" />
                    </div>
                    <div>
                      <h3 className="font-semibold text-slate-200 text-sm group-hover:text-blue-400 transition">
                        {item.name}
                      </h3>
                      <span className="text-[10px] uppercase font-bold text-slate-500">
                        {item.type === 'project' ? 'Project Mount' : 'Scratch Tmp'}
                      </span>
                    </div>
                  </div>
                </div>

                <p className="text-xs font-mono text-slate-400 bg-slate-900/60 p-2 rounded-lg border border-slate-800/60 truncate mb-3" title={item.path}>
                  {item.path}
                </p>

                {item.modified && (
                  <div className="flex items-center gap-1.5 text-[11px] text-slate-500 mb-4">
                    <Clock className="w-3.5 h-3.5" />
                    <span>Modified: {item.modified}</span>
                  </div>
                )}
              </div>

              <div className="pt-3 border-t border-slate-800/80 flex items-center justify-between gap-2">
                <a
                  href={`#/containers/new`}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold bg-blue-600/10 hover:bg-blue-600 text-blue-400 hover:text-white transition"
                >
                  <Box className="w-3.5 h-3.5" />
                  <span>Launch Sandbox</span>
                </a>

                {isAdmin && (
                  <button
                    onClick={() => handleOpenClone(item)}
                    title="Clone Project Directory"
                    className="p-1.5 rounded-lg text-slate-400 hover:text-slate-200 bg-slate-900 border border-slate-800 transition"
                  >
                    <Copy className="w-3.5 h-3.5" />
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Clone Project Modal */}
      <Modal
        isOpen={!!cloneSource}
        onClose={() => setCloneSource(null)}
        title="Clone Project Directory"
      >
        <form onSubmit={handleConfirmClone} className="space-y-4">
          <p className="text-xs text-slate-400">
            Duplicate folder <span className="text-slate-200 font-mono">{cloneSource?.name}</span> to a new target directory.
          </p>

          <div>
            <label className="block text-xs font-medium text-slate-300 mb-1.5">
              New Folder Name
            </label>
            <input
              type="text"
              required
              value={targetName}
              onChange={(e) => setTargetName(e.target.value)}
              className="w-full px-3.5 py-2.5 bg-slate-900 border border-slate-800 rounded-xl text-sm text-slate-200 focus:outline-none"
            />
          </div>

          <div>
            <label className="block text-xs font-medium text-slate-300 mb-1.5">
              Target Destination
            </label>
            <div className="grid grid-cols-2 gap-3">
              <label
                className={`p-3 rounded-xl border cursor-pointer text-xs transition ${
                  cloneLocation === 'projects'
                    ? 'bg-blue-600/10 border-blue-500 text-blue-400'
                    : 'bg-slate-900 border-slate-800 text-slate-400'
                }`}
              >
                <input
                  type="radio"
                  name="cloneLoc"
                  checked={cloneLocation === 'projects'}
                  onChange={() => setCloneLocation('projects')}
                  className="sr-only"
                />
                <span className="font-semibold block text-slate-200">Projects Directory</span>
                <span className="text-[11px] text-slate-400">Persistent storage</span>
              </label>

              <label
                className={`p-3 rounded-xl border cursor-pointer text-xs transition ${
                  cloneLocation === 'tmp'
                    ? 'bg-blue-600/10 border-blue-500 text-blue-400'
                    : 'bg-slate-900 border-slate-800 text-slate-400'
                }`}
              >
                <input
                  type="radio"
                  name="cloneLoc"
                  checked={cloneLocation === 'tmp'}
                  onChange={() => setCloneLocation('tmp')}
                  className="sr-only"
                />
                <span className="font-semibold block text-slate-200">Tmp Scratch</span>
                <span className="text-[11px] text-slate-400">Temporary sandbox</span>
              </label>
            </div>
          </div>

          <div className="flex items-center justify-end gap-3 pt-3">
            <button
              type="button"
              onClick={() => setCloneSource(null)}
              className="px-4 py-2 rounded-xl text-sm text-slate-400"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={cloneLoading}
              className="flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-semibold bg-blue-600 hover:bg-blue-500 text-white"
            >
              {cloneLoading ? 'Cloning...' : 'Clone Folder'}
            </button>
          </div>
        </form>
      </Modal>
    </div>
  );
};
