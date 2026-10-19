import React from 'react';
import { Box, FolderGit2, Layers, HardDrive, Plus, LogOut, Terminal, Shield, Eye } from 'lucide-react';
import { AuthUser } from '../types';

interface NavbarProps {
  currentRoute: string;
  user: AuthUser | null;
  onLogout: () => void;
}

export const Navbar: React.FC<NavbarProps> = ({ currentRoute, user, onLogout }) => {
  const navItems = [
    { label: 'Sandboxes', route: '#/', icon: Box },
    { label: 'Projects', route: '#/projects', icon: FolderGit2 },
    { label: 'Images', route: '#/images', icon: Layers },
    { label: 'Host Devices', route: '#/devices', icon: HardDrive },
  ];

  const isAdmin = user?.role === 'admin';

  return (
    <header className="sticky top-0 z-40 w-full border-b border-slate-800/80 bg-[#0b0f19]/80 backdrop-blur-md">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
        {/* Brand / Logo */}
        <div className="flex items-center gap-6">
          <a href="#/" className="flex items-center gap-3 group">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-tr from-blue-600 via-indigo-600 to-cyan-400 p-[1px] shadow-lg shadow-blue-500/20 group-hover:shadow-blue-500/40 transition">
              <div className="w-full h-full bg-[#0b0f19] rounded-[11px] flex items-center justify-center">
                <Terminal className="w-5 h-5 text-blue-400 group-hover:text-blue-300 transition" />
              </div>
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-bold tracking-tight text-lg text-white font-['Outfit']">
                  go-mcp
                </span>
                <span className="text-[10px] uppercase font-semibold px-1.5 py-0.5 rounded bg-blue-500/10 text-blue-400 border border-blue-500/20">
                  Podman
                </span>
              </div>
              <p className="text-[11px] text-slate-400">Autonomous Agent Sandbox</p>
            </div>
          </a>

          {/* Navigation Links */}
          <nav className="hidden md:flex items-center gap-1 ml-4">
            {navItems.map((item) => {
              const Icon = item.icon;
              const isActive =
                item.route === '#/'
                  ? currentRoute === '#/' || currentRoute === ''
                  : currentRoute.startsWith(item.route);

              return (
                <a
                  key={item.route}
                  href={item.route}
                  className={`flex items-center gap-2 px-3.5 py-2 rounded-lg text-sm font-medium transition ${
                    isActive
                      ? 'bg-blue-600/15 text-blue-400 border border-blue-500/20 shadow-sm'
                      : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/50'
                  }`}
                >
                  <Icon className="w-4 h-4" />
                  {item.label}
                </a>
              );
            })}
          </nav>
        </div>

        {/* Right Controls */}
        <div className="flex items-center gap-3">
          {isAdmin && (
            <a
              href="#/containers/new"
              className="flex items-center gap-2 px-3.5 py-2 rounded-lg text-sm font-semibold bg-blue-600 hover:bg-blue-500 text-white shadow-lg shadow-blue-600/25 transition active:scale-95"
            >
              <Plus className="w-4 h-4" />
              <span className="hidden sm:inline">Spin Up Sandbox</span>
            </a>
          )}

          {user && (
            <div className="flex items-center gap-2 pl-2 border-l border-slate-800">
              <div className="hidden sm:flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-slate-900 border border-slate-800 text-xs">
                {isAdmin ? (
                  <Shield className="w-3.5 h-3.5 text-amber-400" />
                ) : (
                  <Eye className="w-3.5 h-3.5 text-cyan-400" />
                )}
                <span className="font-medium text-slate-300">{user.username}</span>
                <span
                  className={`uppercase text-[9px] font-bold px-1.5 py-0.2 rounded ${
                    isAdmin
                      ? 'bg-amber-500/20 text-amber-300 border border-amber-500/30'
                      : 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/30'
                  }`}
                >
                  {user.role}
                </span>
              </div>

              <button
                onClick={onLogout}
                title="Log Out"
                className="p-2 rounded-lg text-slate-400 hover:text-rose-400 hover:bg-rose-500/10 transition"
              >
                <LogOut className="w-4 h-4" />
              </button>
            </div>
          )}
        </div>
      </div>
    </header>
  );
};
