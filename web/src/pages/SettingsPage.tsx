import React from 'react';
import {Card,CardContent,CardHeader,Badge} from '../components/ui';
export const SettingsPage:React.FC=()=> <div className="mx-auto max-w-5xl px-5 py-8">
 <div className="mb-7"><p className="text-xs font-medium uppercase tracking-[0.16em] text-neutral-500">System</p><h1 className="mt-2 text-3xl font-semibold tracking-tight text-black">Settings</h1><p className="mt-1 text-sm text-neutral-500">Deployment and security configuration lives on the host.</p></div>
 <div className="grid gap-4 md:grid-cols-2">
  <Card><CardHeader><h2 className="text-sm font-semibold">Container boundary</h2></CardHeader><CardContent><Badge tone="success">Host-managed</Badge><p className="mt-3 text-sm text-neutral-600">The control plane owns Podman access. Workload containers receive a project workspace, not the Podman socket or host root.</p></CardContent></Card>
  <Card><CardHeader><h2 className="text-sm font-semibold">GitHub credentials</h2></CardHeader><CardContent><Badge>Host-side broker</Badge><p className="mt-3 text-sm text-neutral-600">Set GITHUB_APP_ID, GITHUB_INSTALLATION_ID and GITHUB_APP_PRIVATE_KEY_PATH on the manager host. Tokens are scoped to a repository and used only for host-side pushes.</p></CardContent></Card>
  <Card><CardHeader><h2 className="text-sm font-semibold">LLM provider</h2></CardHeader><CardContent><p className="text-sm text-neutral-600">Set OPENAI_BASE_URL, OPENAI_API_KEY and OPENAI_MODEL in the manager environment. The agent runtime remains unavailable until all three are configured.</p></CardContent></Card>
  <Card><CardHeader><h2 className="text-sm font-semibold">External access</h2></CardHeader><CardContent><p className="text-sm text-neutral-600">Route Cloudflare Tunnel to the authenticated control-plane HTTP service. Do not publish the Podman socket or workload ports directly.</p></CardContent></Card>
 </div>
</div>;
