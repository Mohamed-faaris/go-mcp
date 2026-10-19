import React, { useState, useEffect } from 'react';
import { Layers, Download, Check, RefreshCw, HardDrive, Cpu, AlertCircle, Plus } from 'lucide-react';
import { apiRequest } from '../api';
import { ImagesResponse, AuthUser } from '../types';
import { Modal } from '../components/Modal';

interface ImagesPageProps {
  user: AuthUser | null;
}

export const ImagesPage: React.FC<ImagesPageProps> = ({ user }) => {
  const [data, setData] = useState<ImagesResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [isPullOpen, setIsPullOpen] = useState(false);
  const [pullReference, setPullReference] = useState('');
  const [pullLoading, setPullLoading] = useState(false);
  const [pullError, setPullError] = useState<string | null>(null);

  const isAdmin = user?.role === 'admin';

  const fetchImages = async () => {
    setLoading(true);
    try {
      const res = await apiRequest<ImagesResponse>('/api/images');
      setData(res);
    } catch (err: any) {
      console.error('Failed to load images:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchImages();
  }, []);

  const handlePullImage = async (e: React.FormEvent) => {
    e.preventDefault();
    setPullError(null);
    setPullLoading(true);
    try {
      await apiRequest('/api/images/pull', {
        method: 'POST',
        body: JSON.stringify({ reference: pullReference }),
      });
      setIsPullOpen(false);
      setPullReference('');
      await fetchImages();
    } catch (err: any) {
      setPullError(err.message || 'Image pull failed');
    } finally {
      setPullLoading(false);
    }
  };

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8 animate-in fade-in duration-300">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-white font-['Outfit'] flex items-center gap-3">
            <span>Container Images</span>
            <span className="text-xs font-semibold px-2.5 py-1 rounded-full bg-blue-500/10 text-blue-400 border border-blue-500/20">
              Podman Registry
            </span>
          </h1>
          <p className="text-slate-400 text-sm mt-1">
            Curated base environments and locally cached Podman container images.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={fetchImages}
            className="flex items-center gap-2 px-3.5 py-2 rounded-xl bg-slate-900 hover:bg-slate-800 border border-slate-800 text-slate-300 transition"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin text-blue-400' : ''}`} />
            <span className="text-xs font-semibold">Refresh</span>
          </button>

          {isAdmin && (
            <button
              onClick={() => setIsPullOpen(true)}
              className="flex items-center gap-2 px-4 py-2 rounded-xl font-semibold text-xs bg-blue-600 hover:bg-blue-500 text-white shadow-lg shadow-blue-500/25 transition active:scale-95"
            >
              <Download className="w-4 h-4" />
              <span>Pull Image</span>
            </button>
          )}
        </div>
      </div>

      {/* Curated Images */}
      <div>
        <h2 className="text-base font-semibold text-slate-200 mb-4 flex items-center gap-2">
          <Layers className="w-4 h-4 text-blue-400" />
          <span>Configured Base Environments</span>
        </h2>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {data?.configured?.map((cfg) => (
            <div
              key={cfg.image}
              className="bg-[#111827] border border-slate-800 rounded-2xl p-5 shadow-lg flex flex-col justify-between"
            >
              <div>
                <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider block mb-1">
                  Base Image
                </span>
                <h3 className="text-base font-bold text-slate-100 mb-2">{cfg.name}</h3>
                <p className="text-xs font-mono text-slate-400 bg-slate-900/80 p-2 rounded-lg border border-slate-800/60 truncate mb-4">
                  {cfg.image}
                </p>
              </div>

              <a
                href={`#/containers/new`}
                className="w-full flex items-center justify-center gap-1.5 py-2 rounded-xl text-xs font-semibold bg-blue-600/10 hover:bg-blue-600 text-blue-400 hover:text-white transition"
              >
                <span>Deploy with this image</span>
              </a>
            </div>
          ))}
        </div>
      </div>

      {/* Locally Cached Images */}
      <div>
        <h2 className="text-base font-semibold text-slate-200 mb-4 flex items-center gap-2">
          <HardDrive className="w-4 h-4 text-emerald-400" />
          <span>Local Image Cache ({data?.local?.length || 0})</span>
        </h2>

        {!data?.local || data.local.length === 0 ? (
          <div className="p-8 text-center border border-dashed border-slate-800 rounded-2xl bg-[#0d121f]/50 text-slate-500 text-xs">
            No local images pulled yet.
          </div>
        ) : (
          <div className="bg-[#111827] border border-slate-800 rounded-2xl overflow-hidden shadow-xl">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs">
                <thead className="bg-slate-900/80 border-b border-slate-800 text-slate-400 uppercase font-semibold">
                  <tr>
                    <th className="px-6 py-3.5">Repository Tag</th>
                    <th className="px-6 py-3.5">Image ID</th>
                    <th className="px-6 py-3.5">Size</th>
                    <th className="px-6 py-3.5 text-right">Action</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800/60 text-slate-300 font-mono">
                  {data.local.map((img) => {
                    const tag = img.RepoTags && img.RepoTags.length > 0 ? img.RepoTags[0] : '<none>';
                    const sizeMB = img.Size ? (img.Size / (1024 * 1024)).toFixed(1) : '—';
                    return (
                      <tr key={img.Id} className="hover:bg-slate-800/30 transition">
                        <td className="px-6 py-4 font-semibold text-slate-200">{tag}</td>
                        <td className="px-6 py-4 text-slate-400">{img.Id.slice(0, 16)}</td>
                        <td className="px-6 py-4 text-slate-400">{sizeMB} MB</td>
                        <td className="px-6 py-4 text-right">
                          <a
                            href={`#/containers/new`}
                            className="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg bg-blue-600/10 hover:bg-blue-600 text-blue-400 hover:text-white transition text-xs font-sans font-semibold"
                          >
                            <span>Use</span>
                          </a>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>

      {/* Pull Image Modal */}
      <Modal isOpen={isPullOpen} onClose={() => setIsPullOpen(false)} title="Pull Container Image">
        <form onSubmit={handlePullImage} className="space-y-4">
          <p className="text-xs text-slate-400">
            Download an image from Docker Hub, Quay.io, or GitHub Packages directly to your local Podman daemon.
          </p>

          {pullError && (
            <div className="p-3 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-300 text-xs">
              {pullError}
            </div>
          )}

          <div>
            <label className="block text-xs font-medium text-slate-300 mb-1.5">
              Image Reference Tag
            </label>
            <input
              type="text"
              required
              placeholder="e.g. docker.io/library/ubuntu:24.04 or alpine:latest"
              value={pullReference}
              onChange={(e) => setPullReference(e.target.value)}
              className="w-full px-3.5 py-2.5 bg-slate-900 border border-slate-800 rounded-xl text-sm text-slate-200 focus:outline-none"
            />
          </div>

          <div className="flex items-center justify-end gap-3 pt-3">
            <button
              type="button"
              onClick={() => setIsPullOpen(false)}
              className="px-4 py-2 rounded-xl text-sm text-slate-400"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={pullLoading}
              className="flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-semibold bg-blue-600 hover:bg-blue-500 text-white"
            >
              {pullLoading ? (
                <>
                  <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                  <span>Pulling image...</span>
                </>
              ) : (
                <>
                  <Download className="w-4 h-4" />
                  <span>Pull Image</span>
                </>
              )}
            </button>
          </div>
        </form>
      </Modal>
    </div>
  );
};
