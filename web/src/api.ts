import { AuthUser } from './types';

const AUTH_KEY = 'gomcp_auth';

export interface StoredAuth {
  username: string;
  role: 'admin' | 'viewer';
  token: string; // base64(username:password)
}

export function getStoredAuth(): StoredAuth | null {
  try {
    const raw = localStorage.getItem(AUTH_KEY);
    if (!raw) return null;
    return JSON.parse(raw);
  } catch {
    return null;
  }
}

export function setStoredAuth(auth: StoredAuth | null) {
  if (auth) {
    localStorage.setItem(AUTH_KEY, JSON.stringify(auth));
  } else {
    localStorage.removeItem(AUTH_KEY);
  }
}

export async function apiRequest<T = any>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const auth = getStoredAuth();
  const headers = new Headers(options.headers || {});

  if (auth && auth.token) {
    headers.set('Authorization', `Basic ${auth.token}`);
  }

  if (options.body && !(options.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  const response = await fetch(path, {
    ...options,
    headers,
  });

  if (response.status === 401) {
    setStoredAuth(null);
    window.location.hash = '#/login';
    throw new Error('Authentication required or session expired.');
  }

  const text = await response.text();
  let data: any = {};
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = { raw: text };
    }
  }

  if (!response.ok) {
    throw new Error(data.error || `HTTP error ${response.status}: ${response.statusText}`);
  }

  return data as T;
}

export function getTerminalWsUrl(containerId: string): string {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const host = window.location.host;
  const auth = getStoredAuth();
  const token = auth ? encodeURIComponent(auth.token) : '';
  return `${protocol}//${host}/api/containers/${encodeURIComponent(containerId)}/terminal?token=${token}`;
}
