import React, { useState, useEffect } from 'react';
import { Camera, RefreshCw, HardDrive, CheckCircle2, ShieldCheck, Box, ExternalLink } from 'lucide-react';
import { apiRequest } from '../api';
import { HostDevice, AuthUser } from '../types';

interface DevicesPageProps {
  user: AuthUser | null;
}

export const DevicesPage: React.FC<DevicesPageProps> = ({ user }) => {
  const [devices, setDevices] = useState<HostDevice[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchDevices = async () => {
    setLoading(true);
    try {
      const data = await apiRequest<HostDevice[]>('/api/devices');
      setDevices(data || []);
    } catch (err: any) {
      console.error('Failed to query host devices:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchDevices();
  }, []);

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8 animate-in fade-in duration-300">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-white font-['Outfit'] flex items-center gap-3">
            <span>Host Hardware & Cameras</span>
            <span className="text-xs font-semibold px-2.5 py-1 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
              V4L2 Linux Subsystem
            </span>
          </h1>
          <p className="text-slate-400 text-sm mt-1">
            Detected camera and media devices available for pass-through into agent sandboxes.
          </p>
        </div>

        <button
          onClick={fetchDevices}
          className="self-start md:self-auto flex items-center gap-2 px-3.5 py-2 rounded-xl bg-slate-900 hover:bg-slate-800 border border-slate-800 text-slate-300 transition"
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin text-blue-400' : ''}`} />
          <span className="text-xs font-semibold">Rescan Devices</span>
        </button>
      </div>

      {/* Hardware explanation card */}
      <div className="bg-gradient-to-r from-blue-900/20 via-slate-900/40 to-emerald-900/20 border border-slate-800 rounded-2xl p-6">
        <div className="flex items-start gap-4">
          <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400 flex-shrink-0">
            <ShieldCheck className="w-5 h-5" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-slate-200">
              Container Hardware Pass-Through Enabled
            </h3>
            <p className="text-xs text-slate-400 mt-1 leading-relaxed">
              When spinning up a sandbox, selected host devices are mapped via{' '}
              <code className="text-emerald-400 font-mono">--device /dev/videoX</code> directly into
              the container. This gives autonomous agents and computer vision models (OpenCV, FFmpeg,
              V4L2) direct, low-latency access to Econ Systems cameras and video feeds.
            </p>
          </div>
        </div>
      </div>

      {/* Devices list */}
      {loading ? (
        <div className="py-20 flex flex-col items-center justify-center gap-2 text-slate-500">
          <div className="w-7 h-7 border-2 border-emerald-500 border-t-transparent rounded-full animate-spin" />
          <p className="text-xs">Scanning /sys/class/video4linux...</p>
        </div>
      ) : devices.length === 0 ? (
        <div className="p-12 text-center border border-dashed border-slate-800 rounded-2xl bg-[#0d121f]/50 text-slate-500 text-xs">
          No V4L2 video devices detected on this host.
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {devices.map((dev) => (
            <div
              key={dev.path}
              className="bg-[#111827] border border-slate-800 hover:border-slate-700 rounded-2xl p-5 shadow-lg flex flex-col justify-between transition group"
            >
              <div>
                <div className="flex items-start justify-between gap-3 mb-3">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-xl bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 flex items-center justify-center">
                      <Camera className="w-5 h-5" />
                    </div>
                    <div>
                      <h3 className="font-semibold text-slate-100 text-sm group-hover:text-emerald-400 transition">
                        {dev.name}
                      </h3>
                      <span className="text-[10px] font-mono uppercase px-1.5 py-0.5 rounded bg-slate-900 border border-slate-800 text-slate-400">
                        {dev.type}
                      </span>
                    </div>
                  </div>
                </div>

                <div className="space-y-2 text-xs font-mono mb-4">
                  <div className="p-2 rounded-lg bg-slate-900/80 border border-slate-800/80 text-emerald-400 flex items-center justify-between">
                    <span>Node:</span>
                    <span className="font-bold">{dev.path}</span>
                  </div>
                  {dev.busInfo && (
                    <div className="p-2 rounded-lg bg-slate-900/60 border border-slate-800/60 text-slate-400 flex items-center justify-between">
                      <span>Stream:</span>
                      <span>{dev.busInfo}</span>
                    </div>
                  )}
                </div>
              </div>

              <div className="pt-3 border-t border-slate-800 flex items-center justify-between">
                <div className="flex items-center gap-1.5 text-[11px] text-emerald-400">
                  <CheckCircle2 className="w-3.5 h-3.5" />
                  <span>Available for pass-through</span>
                </div>

                <a
                  href="#/containers/new"
                  className="flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs font-semibold bg-emerald-500/10 hover:bg-emerald-500 text-emerald-400 hover:text-white transition"
                >
                  <Box className="w-3 h-3" />
                  <span>Attach</span>
                </a>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
