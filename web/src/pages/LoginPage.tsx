import React, { useState } from 'react';
import { Terminal, Shield, Eye, Lock, User, ArrowRight, AlertCircle } from 'lucide-react';
import { apiRequest, setStoredAuth } from '../api';
import { AuthUser } from '../types';

interface LoginPageProps {
  onLoginSuccess: (user: AuthUser) => void;
}

export const LoginPage: React.FC<LoginPageProps> = ({ onLoginSuccess }) => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      const res = await apiRequest<{ username: string; role: 'admin' | 'viewer' }>(
        '/api/auth/login',
        {
          method: 'POST',
          body: JSON.stringify({ username, password }),
        }
      );

      // Create base64 basic auth token
      const token = btoa(`${username}:${password}`);
      const stored = {
        username: res.username,
        role: res.role,
        token,
      };
      setStoredAuth(stored);
      onLoginSuccess(res);
      window.location.hash = '#/';
    } catch (err: any) {
      setError(err.message || 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  const handleQuickLogin = (user: string, pass: string) => {
    setUsername(user);
    setPassword(pass);
    // Auto-submit after setting state
    setTimeout(() => {
      const fakeForm = { preventDefault: () => {} } as React.FormEvent;
      // We directly submit with explicit credentials
      setError(null);
      setLoading(true);
      apiRequest<{ username: string; role: 'admin' | 'viewer' }>('/api/auth/login', {
        method: 'POST',
        body: JSON.stringify({ username: user, password: pass }),
      })
        .then((res) => {
          const token = btoa(`${user}:${pass}`);
          setStoredAuth({ username: res.username, role: res.role, token });
          onLoginSuccess(res);
          window.location.hash = '#/';
        })
        .catch((err) => {
          setError(err.message || 'Quick login failed');
        })
        .finally(() => setLoading(false));
    }, 50);
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-4 relative overflow-hidden bg-[#070a12]">
      {/* Background radial gradient glow */}
      <div className="absolute top-1/4 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[600px] bg-blue-600/10 rounded-full blur-[140px] pointer-events-none" />
      <div className="absolute bottom-10 right-10 w-[400px] h-[400px] bg-indigo-600/10 rounded-full blur-[120px] pointer-events-none" />

      <div className="relative w-full max-w-md">
        {/* Card */}
        <div className="bg-[#0e1422]/90 border border-slate-800/90 rounded-3xl p-8 shadow-2xl backdrop-blur-xl">
          {/* Logo & Header */}
          <div className="text-center mb-8">
            <div className="inline-flex items-center justify-center w-14 h-14 rounded-2xl bg-gradient-to-tr from-blue-600 via-indigo-600 to-cyan-400 p-[1.5px] shadow-xl shadow-blue-500/20 mb-4">
              <div className="w-full h-full bg-[#0b0f19] rounded-[14px] flex items-center justify-center">
                <Terminal className="w-7 h-7 text-blue-400" />
              </div>
            </div>
            <h1 className="text-2xl font-bold text-white tracking-tight font-['Outfit']">
              go-mcp Sandboxes
            </h1>
            <p className="text-xs text-slate-400 mt-1">
              Podman Autonomous Agent Sandbox Manager
            </p>
          </div>

          {/* Error Message */}
          {error && (
            <div className="mb-6 flex items-center gap-2.5 p-3.5 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-300 text-xs animate-in fade-in">
              <AlertCircle className="w-4 h-4 flex-shrink-0" />
              <span>{error}</span>
            </div>
          )}

          {/* Form */}
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1.5">
                Username
              </label>
              <div className="relative">
                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-slate-500">
                  <User className="w-4 h-4" />
                </div>
                <input
                  type="text"
                  required
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder="admin or viewer"
                  className="w-full pl-10 pr-4 py-2.5 bg-slate-900/80 border border-slate-800 rounded-xl text-sm text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500 transition"
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1.5">
                Password
              </label>
              <div className="relative">
                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-slate-500">
                  <Lock className="w-4 h-4" />
                </div>
                <input
                  type="password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••"
                  className="w-full pl-10 pr-4 py-2.5 bg-slate-900/80 border border-slate-800 rounded-xl text-sm text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500 transition"
                />
              </div>
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full mt-2 flex items-center justify-center gap-2 py-2.5 px-4 rounded-xl text-sm font-semibold bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white shadow-lg shadow-blue-500/25 transition active:scale-[0.99] disabled:opacity-50"
            >
              {loading ? (
                <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
              ) : (
                <>
                  <span>Sign In</span>
                  <ArrowRight className="w-4 h-4" />
                </>
              )}
            </button>
          </form>

          {/* Quick Login Helper */}
          <div className="mt-8 pt-6 border-t border-slate-800/80">
            <p className="text-xs text-center text-slate-400 font-medium mb-3">
              Quick One-Click Demo Access
            </p>
            <div className="grid grid-cols-2 gap-2.5">
              <button
                type="button"
                onClick={() => handleQuickLogin('admin', 'admin123')}
                className="flex items-center justify-center gap-2 px-3 py-2 rounded-xl bg-amber-500/10 hover:bg-amber-500/20 border border-amber-500/20 text-amber-300 text-xs font-semibold transition"
              >
                <Shield className="w-3.5 h-3.5" />
                <span>Admin Role</span>
              </button>

              <button
                type="button"
                onClick={() => handleQuickLogin('viewer', 'viewer123')}
                className="flex items-center justify-center gap-2 px-3 py-2 rounded-xl bg-cyan-500/10 hover:bg-cyan-500/20 border border-cyan-500/20 text-cyan-300 text-xs font-semibold transition"
              >
                <Eye className="w-3.5 h-3.5" />
                <span>Viewer Role</span>
              </button>
            </div>
          </div>
        </div>

        {/* Footer info */}
        <p className="text-center text-[11px] text-slate-500 mt-6">
          Antigravity autonomous environment &bull; Podman Libpod v4.0.0
        </p>
      </div>
    </div>
  );
};
