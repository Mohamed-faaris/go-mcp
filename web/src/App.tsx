import React, { useState, useEffect } from 'react';
import { Navbar } from './components/Navbar';
import { LoginPage } from './pages/LoginPage';
import { ContainersPage } from './pages/ContainersPage';
import { NewContainerPage } from './pages/NewContainerPage';
import { ContainerDetailPage } from './pages/ContainerDetailPage';
import { ProjectsPage } from './pages/ProjectsPage';
import { ImagesPage } from './pages/ImagesPage';
import { DevicesPage } from './pages/DevicesPage';
import { getStoredAuth, setStoredAuth, apiRequest } from './api';
import { AuthUser } from './types';

export const App: React.FC = () => {
  const [currentRoute, setCurrentRoute] = useState(window.location.hash || '#/');
  const [user, setUser] = useState<AuthUser | null>(null);
  const [authChecked, setAuthChecked] = useState(false);

  // Sync hash changes
  useEffect(() => {
    const handleHashChange = () => {
      setCurrentRoute(window.location.hash || '#/');
    };
    window.addEventListener('hashchange', handleHashChange);
    return () => window.removeEventListener('hashchange', handleHashChange);
  }, []);

  // Validate existing auth on initial load
  useEffect(() => {
    const stored = getStoredAuth();
    if (!stored) {
      setAuthChecked(true);
      if (window.location.hash !== '#/login') {
        window.location.hash = '#/login';
      }
      return;
    }

    // Verify token with server
    apiRequest<{ username: string; role: 'admin' | 'viewer' }>('/api/auth/me')
      .then((res) => {
        setUser({ username: res.username, role: res.role });
      })
      .catch(() => {
        setStoredAuth(null);
        window.location.hash = '#/login';
      })
      .finally(() => {
        setAuthChecked(true);
      });
  }, []);

  const handleLogout = () => {
    setStoredAuth(null);
    setUser(null);
    window.location.hash = '#/login';
  };

  const handleLoginSuccess = (authUser: AuthUser) => {
    setUser(authUser);
  };

  if (!authChecked) {
    return (
      <div className="min-h-screen bg-[#070a12] flex items-center justify-center text-slate-500">
        <div className="flex flex-col items-center gap-3">
          <div className="w-8 h-8 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
          <p className="text-xs font-mono">Authenticating session...</p>
        </div>
      </div>
    );
  }

  // If on login route or not authenticated
  if (currentRoute === '#/login' || !user) {
    return <LoginPage onLoginSuccess={handleLoginSuccess} />;
  }

  // Route matching
  const renderContent = () => {
    if (currentRoute === '#/' || currentRoute === '') {
      return <ContainersPage user={user} />;
    }
    if (currentRoute === '#/containers/new') {
      return <NewContainerPage />;
    }
    if (currentRoute.startsWith('#/containers/')) {
      const containerId = currentRoute.replace('#/containers/', '');
      return <ContainerDetailPage containerId={containerId} user={user} />;
    }
    if (currentRoute.startsWith('#/projects')) {
      return <ProjectsPage user={user} />;
    }
    if (currentRoute.startsWith('#/images')) {
      return <ImagesPage user={user} />;
    }
    if (currentRoute.startsWith('#/devices')) {
      return <DevicesPage user={user} />;
    }

    // Default fallback
    return <ContainersPage user={user} />;
  };

  return (
    <div className="min-h-screen bg-[#070a12] text-slate-100 flex flex-col font-['Outfit',sans-serif]">
      <Navbar currentRoute={currentRoute} user={user} onLogout={handleLogout} />
      <main className="flex-1 pb-16">{renderContent()}</main>
      <footer className="border-t border-slate-900 bg-[#05070d] py-6 text-center text-xs text-slate-600">
        <div className="max-w-7xl mx-auto px-4 flex flex-col sm:flex-row items-center justify-between gap-3">
          <p>go-mcp Podman AI Sandbox &bull; Autonomous Computer Control</p>
          <p className="font-mono text-[11px] text-slate-500">
            Libpod v4.0.0 &bull; WebSocket TTY &bull; V4L2 Device Pass-Through
          </p>
        </div>
      </footer>
    </div>
  );
};
