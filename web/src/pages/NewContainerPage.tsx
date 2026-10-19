import React, { useState, useEffect } from 'react';
import { Box, ArrowLeft, Cpu, HardDrive, Check, Sparkles, Folder, Plus, Trash2, Camera, ShieldAlert } from 'lucide-react';
import { apiRequest } from '../api';
import { MCPToolSpec, PublicConfig, ProjectsResponse, HostDevice } from '../types';

export const NewContainerPage: React.FC = () => {
  const [name, setName] = useState(`gomcp-agent-${Math.floor(1000 + Math.random() * 9000)}`);
  const [selectedImage, setSelectedImage] = useState('docker.io/library/ubuntu:24.04');
  const [customImage, setCustomImage] = useState('');
  const [isCustomImage, setIsCustomImage] = useState(false);

  // Mount
  const [mountMode, setMountMode] = useState<'project' | 'tmp' | 'none'>('project');
  const [selectedProject, setSelectedProject] = useState('');
  const [tmpDirName, setTmpDirName] = useState(`scratch-${Math.floor(1000 + Math.random() * 9000)}`);

  // Resources
  const [memoryLimit, setMemoryLimit] = useState('2g');
  const [cpuLimit, setCpuLimit] = useState('2');
  const [autoStart, setAutoStart] = useState(true);

  // MCP Tools
  const [mcpTools, setMcpTools] = useState<MCPToolSpec[]>([
    { name: 'desktop-commander', command: 'npx', args: ['-y', '@anthropic/desktop-commander-mcp'] },
    { name: 'playwright', command: 'npx', args: ['-y', '@anthropic/playwright-mcp'] },
    { name: 'ubuntu-gui', command: 'npx', args: ['-y', '@anthropic/ubuntu-gui-mcp'] },
  ]);
  const [customToolName, setCustomToolName] = useState('');
  const [customToolCmd, setCustomToolCmd] = useState('');
  const [customToolArgs, setCustomToolArgs] = useState('');

  // Host Devices (Cameras / V4L2)
  const [availableDevices, setAvailableDevices] = useState<HostDevice[]>([]);
  const [selectedDevices, setSelectedDevices] = useState<string[]>([]);

  // Projects list & config
  const [projects, setProjects] = useState<ProjectsResponse | null>(null);
  const [config, setConfig] = useState<PublicConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // Fetch system config
    apiRequest<PublicConfig>('/api/config')
      .then((cfg) => {
        setConfig(cfg);
        if (cfg.defaults?.base_image) {
          setSelectedImage(cfg.defaults.base_image);
        }
        if (cfg.defaults?.mcp_tools) {
          setMcpTools(cfg.defaults.mcp_tools);
        }
      })
      .catch(() => {});

    // Fetch projects
    apiRequest<ProjectsResponse>('/api/projects')
      .then((res) => {
        setProjects(res);
        if (res.items && res.items.length > 0) {
          setSelectedProject(res.items[0].path);
        }
      })
      .catch(() => {});

    // Fetch host devices (cameras / V4L2)
    apiRequest<HostDevice[]>('/api/devices')
      .then((devs) => {
        setAvailableDevices(devs || []);
      })
      .catch(() => {});
  }, []);

  const handleToggleTool = (toolName: string) => {
    const exists = mcpTools.some((t) => t.name === toolName);
    if (exists) {
      setMcpTools(mcpTools.filter((t) => t.name !== toolName));
    } else {
      // Find tool in default config if available
      const defTool = config?.defaults.mcp_tools?.find((t) => t.name === toolName);
      if (defTool) {
        setMcpTools([...mcpTools, defTool]);
      } else {
        setMcpTools([...mcpTools, { name: toolName, command: 'npx', args: ['-y', `@anthropic/${toolName}-mcp`] }]);
      }
    }
  };

  const handleAddCustomTool = () => {
    if (!customToolName || !customToolCmd) return;
    const argsArray = customToolArgs ? customToolArgs.split(' ').filter(Boolean) : [];
    setMcpTools([...mcpTools, { name: customToolName, command: customToolCmd, args: argsArray }]);
    setCustomToolName('');
    setCustomToolCmd('');
    setCustomToolArgs('');
  };

  const handleRemoveTool = (toolName: string) => {
    setMcpTools(mcpTools.filter((t) => t.name !== toolName));
  };

  const handleToggleDevice = (devPath: string) => {
    if (selectedDevices.includes(devPath)) {
      setSelectedDevices(selectedDevices.filter((d) => d !== devPath));
    } else {
      setSelectedDevices([...selectedDevices, devPath]);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    let projectDir = '';
    if (mountMode === 'project') {
      projectDir = selectedProject;
    } else if (mountMode === 'tmp') {
      projectDir = config?.directories.tmp_podmans
        ? `${config.directories.tmp_podmans}/${tmpDirName}`
        : `/host/tmp-podmans/${tmpDirName}`;
    }

    const payload = {
      name,
      image: isCustomImage ? customImage : selectedImage,
      projectDir,
      memoryLimit,
      cpuLimit,
      autoStart,
      mcpTools,
      devices: selectedDevices,
    };

    try {
      const res = await apiRequest<{ id: string }>('/api/containers', {
        method: 'POST',
        body: JSON.stringify(payload),
      });
      window.location.hash = `#/containers/${res.id}`;
    } catch (err: any) {
      setError(err.message || 'Container creation failed');
      setLoading(false);
    }
  };

  const imageOptions = [
    {
      id: 'docker.io/library/ubuntu:24.04',
      title: 'Ubuntu 24.04 LTS',
      desc: 'Standard Ubuntu Noble environment, full MCP desktop compatibility.',
      tag: 'Recommended',
    },
    {
      id: 'docker.io/library/node:20-bookworm',
      title: 'Node.js 20 (Bookworm)',
      desc: 'Optimized for JavaScript, Playwright & Node MCP servers.',
    },
    {
      id: 'docker.io/library/python:3.12-bookworm',
      title: 'Python 3.12 (Bookworm)',
      desc: 'Pre-configured with Python 3.12, pip & build essentials.',
    },
    {
      id: 'docker.io/library/alpine:latest',
      title: 'Alpine Linux',
      desc: 'Lightweight minimal image (~5MB) for quick shell tasks.',
    },
  ];

  return (
    <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8 animate-in fade-in duration-300">
      {/* Back button */}
      <a
        href="#/"
        className="inline-flex items-center gap-2 text-sm text-slate-400 hover:text-slate-200 transition mb-6"
      >
        <ArrowLeft className="w-4 h-4" />
        <span>Back to Sandboxes</span>
      </a>

      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold tracking-tight text-white font-['Outfit']">
          Spin Up Agent Sandbox
        </h1>
        <p className="text-sm text-slate-400 mt-1">
          Deploy an isolated Podman container with bundled MCP tools, mounts, and host hardware pass-through.
        </p>
      </div>

      {error && (
        <div className="mb-6 flex items-center gap-3 p-4 rounded-2xl bg-rose-500/10 border border-rose-500/20 text-rose-300 text-sm">
          <ShieldAlert className="w-5 h-5 flex-shrink-0" />
          <p>{error}</p>
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-8">
        {/* Step 1: Identity & Name */}
        <div className="bg-[#111827]/90 border border-slate-800 rounded-2xl p-6">
          <h2 className="text-base font-semibold text-slate-200 mb-4 flex items-center gap-2">
            <Box className="w-5 h-5 text-blue-400" />
            <span>1. Container Identity</span>
          </h2>
          <div>
            <label className="block text-xs font-medium text-slate-300 mb-1.5">
              Container Name
            </label>
            <input
              type="text"
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. gomcp-agent-ubuntu"
              className="w-full max-w-md px-3.5 py-2.5 bg-slate-900 border border-slate-800 rounded-xl text-sm text-slate-200 font-mono focus:outline-none focus:ring-2 focus:ring-blue-500/40"
            />
            <p className="text-xs text-slate-500 mt-1">
              Prefix <code className="text-slate-400">gomcp-</code> will be automatically prefixed if omitted.
            </p>
          </div>
        </div>

        {/* Step 2: Base Image */}
        <div className="bg-[#111827]/90 border border-slate-800 rounded-2xl p-6">
          <h2 className="text-base font-semibold text-slate-200 mb-4 flex items-center gap-2">
            <Cpu className="w-5 h-5 text-indigo-400" />
            <span>2. Base Image</span>
          </h2>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-3 mb-4">
            {imageOptions.map((opt) => {
              const selected = !isCustomImage && selectedImage === opt.id;
              return (
                <div
                  key={opt.id}
                  onClick={() => {
                    setSelectedImage(opt.id);
                    setIsCustomImage(false);
                  }}
                  className={`p-4 rounded-xl border cursor-pointer transition relative ${
                    selected
                      ? 'bg-blue-600/10 border-blue-500/60 shadow-md shadow-blue-500/10'
                      : 'bg-slate-900/60 border-slate-800 hover:border-slate-700'
                  }`}
                >
                  <div className="flex items-start justify-between">
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-semibold text-sm text-slate-200">{opt.title}</span>
                        {opt.tag && (
                          <span className="text-[10px] uppercase font-bold px-1.5 py-0.5 rounded bg-blue-500/20 text-blue-400 border border-blue-500/30">
                            {opt.tag}
                          </span>
                        )}
                      </div>
                      <p className="text-xs text-slate-400 mt-1">{opt.desc}</p>
                    </div>
                    {selected && (
                      <div className="w-5 h-5 rounded-full bg-blue-500 flex items-center justify-center text-white flex-shrink-0">
                        <Check className="w-3.5 h-3.5" />
                      </div>
                    )}
                  </div>
                </div>
              );
            })}
          </div>

          {/* Custom image option */}
          <div
            onClick={() => setIsCustomImage(true)}
            className={`p-4 rounded-xl border cursor-pointer transition ${
              isCustomImage
                ? 'bg-blue-600/10 border-blue-500/60'
                : 'bg-slate-900/60 border-slate-800 hover:border-slate-700'
            }`}
          >
            <div className="flex items-center gap-2 mb-2">
              <span className="font-semibold text-sm text-slate-200">Custom Container Image</span>
            </div>
            <input
              type="text"
              value={customImage}
              onChange={(e) => {
                setCustomImage(e.target.value);
                setIsCustomImage(true);
              }}
              placeholder="e.g. docker.io/library/fedora:40 or ghcr.io/my-org/custom-agent:latest"
              className="w-full px-3.5 py-2 bg-slate-900 border border-slate-800 rounded-lg text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500/40"
            />
          </div>
        </div>

        {/* Step 3: Workspace Mount */}
        <div className="bg-[#111827]/90 border border-slate-800 rounded-2xl p-6">
          <h2 className="text-base font-semibold text-slate-200 mb-4 flex items-center gap-2">
            <Folder className="w-5 h-5 text-amber-400" />
            <span>3. Workspace Directory (/workspace)</span>
          </h2>

          <div className="space-y-3">
            <label className="flex items-center gap-3 p-3 rounded-xl bg-slate-900/60 border border-slate-800 cursor-pointer">
              <input
                type="radio"
                name="mountMode"
                checked={mountMode === 'project'}
                onChange={() => setMountMode('project')}
                className="text-blue-500 focus:ring-blue-500"
              />
              <div className="flex-1">
                <span className="text-sm font-medium text-slate-200">Link Existing Host Project</span>
                <p className="text-xs text-slate-400">Bind mount a folder from host projects directory</p>
              </div>
            </label>

            {mountMode === 'project' && (
              <div className="pl-6 pt-2">
                {projects?.items && projects.items.length > 0 ? (
                  <select
                    value={selectedProject}
                    onChange={(e) => setSelectedProject(e.target.value)}
                    className="w-full max-w-md px-3.5 py-2.5 bg-slate-900 border border-slate-800 rounded-xl text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500/40"
                  >
                    {projects.items.map((item) => (
                      <option key={item.path} value={item.path}>
                        {item.name} ({item.path})
                      </option>
                    ))}
                  </select>
                ) : (
                  <p className="text-xs text-slate-400">
                    No folders found in host projects directory. You can create a scratch directory below.
                  </p>
                )}
              </div>
            )}

            <label className="flex items-center gap-3 p-3 rounded-xl bg-slate-900/60 border border-slate-800 cursor-pointer">
              <input
                type="radio"
                name="mountMode"
                checked={mountMode === 'tmp'}
                onChange={() => setMountMode('tmp')}
                className="text-blue-500 focus:ring-blue-500"
              />
              <div className="flex-1">
                <span className="text-sm font-medium text-slate-200">Ephemeral Scratch Folder</span>
                <p className="text-xs text-slate-400">Auto-create a folder inside tmp-podmans</p>
              </div>
            </label>

            {mountMode === 'tmp' && (
              <div className="pl-6 pt-2">
                <input
                  type="text"
                  value={tmpDirName}
                  onChange={(e) => setTmpDirName(e.target.value)}
                  placeholder="scratch-folder-name"
                  className="w-full max-w-md px-3.5 py-2 bg-slate-900 border border-slate-800 rounded-xl text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500/40"
                />
              </div>
            )}

            <label className="flex items-center gap-3 p-3 rounded-xl bg-slate-900/60 border border-slate-800 cursor-pointer">
              <input
                type="radio"
                name="mountMode"
                checked={mountMode === 'none'}
                onChange={() => setMountMode('none')}
                className="text-blue-500 focus:ring-blue-500"
              />
              <div className="flex-1">
                <span className="text-sm font-medium text-slate-200">No Host Mount</span>
                <p className="text-xs text-slate-400">Container internal filesystem only (fully isolated)</p>
              </div>
            </label>
          </div>
        </div>

        {/* Step 4: Host Hardware / Cameras Pass-Through */}
        <div className="bg-[#111827]/90 border border-slate-800 rounded-2xl p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-base font-semibold text-slate-200 flex items-center gap-2">
              <Camera className="w-5 h-5 text-emerald-400" />
              <span>4. Host Camera & Video Hardware Pass-Through</span>
            </h2>
            <span className="text-xs font-semibold px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
              V4L2 Device Nodes
            </span>
          </div>

          <p className="text-xs text-slate-400 mb-4">
            Pass host camera devices directly into the Podman container so MCP vision or OpenCV tools
            can access live video streams.
          </p>

          {availableDevices.length === 0 ? (
            <div className="p-4 rounded-xl bg-slate-900/60 border border-slate-800 text-xs text-slate-400">
              No V4L2 video devices detected on host.
            </div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              {availableDevices.map((dev) => {
                const isChecked = selectedDevices.includes(dev.path);
                return (
                  <div
                    key={dev.path}
                    onClick={() => handleToggleDevice(dev.path)}
                    className={`flex items-start gap-3 p-3.5 rounded-xl border cursor-pointer transition ${
                      isChecked
                        ? 'bg-emerald-500/10 border-emerald-500/50 shadow-sm'
                        : 'bg-slate-900/60 border-slate-800 hover:border-slate-700'
                    }`}
                  >
                    <input
                      type="checkbox"
                      checked={isChecked}
                      onChange={() => {}} // handled by parent div
                      className="mt-0.5 rounded text-emerald-500 focus:ring-emerald-500"
                    />
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center justify-between">
                        <span className="text-xs font-semibold text-slate-200 truncate">
                          {dev.name}
                        </span>
                        <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-slate-800 text-slate-400">
                          {dev.type}
                        </span>
                      </div>
                      <p className="text-[11px] font-mono text-emerald-400/90 truncate mt-0.5">
                        {dev.path} {dev.busInfo && `(${dev.busInfo})`}
                      </p>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Step 5: Bundled MCP Tools */}
        <div className="bg-[#111827]/90 border border-slate-800 rounded-2xl p-6">
          <h2 className="text-base font-semibold text-slate-200 mb-4 flex items-center gap-2">
            <Sparkles className="w-5 h-5 text-blue-400" />
            <span>5. Bundled MCP Server Tools</span>
          </h2>

          <div className="space-y-3 mb-6">
            {[
              {
                name: 'desktop-commander',
                title: 'Desktop Commander MCP',
                desc: 'File management, system process control, shell execution',
              },
              {
                name: 'playwright',
                title: 'Playwright MCP',
                desc: 'Automated browser navigation, page interactions, screenshot capture',
              },
              {
                name: 'ubuntu-gui',
                title: 'Ubuntu GUI MCP',
                desc: 'X11/Wayland desktop automation, mouse click, keyboard typing',
              },
            ].map((t) => {
              const active = mcpTools.some((x) => x.name === t.name);
              return (
                <div
                  key={t.name}
                  onClick={() => handleToggleTool(t.name)}
                  className={`flex items-start gap-3 p-3.5 rounded-xl border cursor-pointer transition ${
                    active
                      ? 'bg-blue-600/10 border-blue-500/50 shadow-sm'
                      : 'bg-slate-900/60 border-slate-800 hover:border-slate-700'
                  }`}
                >
                  <input
                    type="checkbox"
                    checked={active}
                    onChange={() => {}}
                    className="mt-0.5 rounded text-blue-500 focus:ring-blue-500"
                  />
                  <div className="flex-1">
                    <span className="text-sm font-semibold text-slate-200">{t.title}</span>
                    <p className="text-xs text-slate-400">{t.desc}</p>
                  </div>
                </div>
              );
            })}
          </div>

          {/* Additional MCP tools list */}
          {mcpTools.length > 0 && (
            <div className="mb-4 space-y-2">
              <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">
                Configured Tools ({mcpTools.length})
              </span>
              <div className="space-y-1.5">
                {mcpTools.map((tool) => (
                  <div
                    key={tool.name}
                    className="flex items-center justify-between p-2.5 rounded-lg bg-slate-900 border border-slate-800 text-xs"
                  >
                    <div>
                      <span className="font-semibold text-slate-200">{tool.name}</span>
                      <span className="font-mono text-slate-500 ml-2">
                        {tool.command} {tool.args.join(' ')}
                      </span>
                    </div>
                    <button
                      type="button"
                      onClick={() => handleRemoveTool(tool.name)}
                      className="text-slate-500 hover:text-rose-400 p-1"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Add custom tool input */}
          <div className="p-3.5 rounded-xl bg-slate-900/60 border border-slate-800/80">
            <span className="text-xs font-semibold text-slate-300 block mb-2">
              + Add Custom MCP Server Tool
            </span>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
              <input
                type="text"
                placeholder="Tool Name (e.g. git-mcp)"
                value={customToolName}
                onChange={(e) => setCustomToolName(e.target.value)}
                className="px-3 py-1.5 bg-slate-900 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none"
              />
              <input
                type="text"
                placeholder="Command (e.g. npx or python)"
                value={customToolCmd}
                onChange={(e) => setCustomToolCmd(e.target.value)}
                className="px-3 py-1.5 bg-slate-900 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none"
              />
              <div className="flex gap-2">
                <input
                  type="text"
                  placeholder="Args (space-separated)"
                  value={customToolArgs}
                  onChange={(e) => setCustomToolArgs(e.target.value)}
                  className="flex-1 px-3 py-1.5 bg-slate-900 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none"
                />
                <button
                  type="button"
                  onClick={handleAddCustomTool}
                  className="px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold"
                >
                  Add
                </button>
              </div>
            </div>
          </div>
        </div>

        {/* Step 6: Resource Limits & Options */}
        <div className="bg-[#111827]/90 border border-slate-800 rounded-2xl p-6">
          <h2 className="text-base font-semibold text-slate-200 mb-4 flex items-center gap-2">
            <HardDrive className="w-5 h-5 text-cyan-400" />
            <span>6. Resource Limits & Execution</span>
          </h2>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-6 mb-4">
            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1.5">
                Memory Limit
              </label>
              <div className="flex items-center gap-2">
                {['512m', '1g', '2g', '4g'].map((m) => (
                  <button
                    type="button"
                    key={m}
                    onClick={() => setMemoryLimit(m)}
                    className={`px-3 py-1.5 rounded-lg text-xs uppercase font-medium transition ${
                      memoryLimit === m
                        ? 'bg-blue-600 text-white'
                        : 'bg-slate-900 border border-slate-800 text-slate-400 hover:text-slate-200'
                    }`}
                  >
                    {m}
                  </button>
                ))}
              </div>
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1.5">
                CPU Cores Quota
              </label>
              <div className="flex items-center gap-2">
                {['1', '2', '4', '8'].map((c) => (
                  <button
                    type="button"
                    key={c}
                    onClick={() => setCpuLimit(c)}
                    className={`px-3 py-1.5 rounded-lg text-xs font-medium transition ${
                      cpuLimit === c
                        ? 'bg-blue-600 text-white'
                        : 'bg-slate-900 border border-slate-800 text-slate-400 hover:text-slate-200'
                    }`}
                  >
                    {c} {c === '1' ? 'Core' : 'Cores'}
                  </button>
                ))}
              </div>
            </div>
          </div>

          <label className="flex items-center gap-3 pt-4 border-t border-slate-800 cursor-pointer">
            <input
              type="checkbox"
              checked={autoStart}
              onChange={(e) => setAutoStart(e.target.checked)}
              className="rounded text-blue-500 focus:ring-blue-500"
            />
            <div>
              <span className="text-sm font-medium text-slate-200">
                Immediately start container upon creation
              </span>
              <p className="text-xs text-slate-400">
                Container will be launched and live interactive shell will be ready instantly.
              </p>
            </div>
          </label>
        </div>

        {/* Submit button */}
        <div className="flex items-center justify-end gap-4">
          <a
            href="#/"
            className="px-5 py-2.5 rounded-xl text-sm text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition"
          >
            Cancel
          </a>
          <button
            type="submit"
            disabled={loading}
            className="flex items-center gap-2 px-6 py-2.5 rounded-xl text-sm font-semibold bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white shadow-xl shadow-blue-500/25 transition active:scale-95 disabled:opacity-50"
          >
            {loading ? (
              <div className="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin" />
            ) : (
              <>
                <Box className="w-4 h-4" />
                <span>Deploy Sandbox Container</span>
              </>
            )}
          </button>
        </div>
      </form>
    </div>
  );
};
