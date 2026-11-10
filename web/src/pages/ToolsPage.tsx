import React,{useCallback,useEffect,useState} from 'react';
import {Search,Settings2,ShieldAlert} from 'lucide-react';
import {apiRequest} from '../api';
import {ToolDefinition} from '../types';
import {Badge,Button,Card,CardContent,CardHeader,Dialog,FieldLabel,Select} from '../components/ui';

export const ToolsPage:React.FC=()=>{
 const [tools,setTools]=useState<ToolDefinition[]>([]);
 const [loading,setLoading]=useState(true);
 const [query,setQuery]=useState('');
 const [editing,setEditing]=useState<ToolDefinition|null>(null);
 const [policy,setPolicy]=useState('automatic');
 const [enabled,setEnabled]=useState(true);
 const [saving,setSaving]=useState(false);
 const [error,setError]=useState('');
 const load=useCallback(async()=>{
  try{const r=await apiRequest<{tools:ToolDefinition[]}>('/api/v1/tools');setTools(r.tools||[]);}
  catch(e:any){setError(e.message||'Unable to load tool catalog');}
  finally{setLoading(false);}
 },[]);
 useEffect(()=>{void load();},[load]);
 const openEdit=(t:ToolDefinition)=>{setEditing(t);setPolicy(t.policy);setEnabled(t.enabled);};
 const save=async()=>{
  if(!editing)return;
  setSaving(true);
  try{await apiRequest('/api/v1/tools/'+encodeURIComponent(editing.id),{method:'PATCH',body:JSON.stringify({policy,enabled})});setEditing(null);await load();}
  catch(e:any){setError(e.message||'Could not update policy');}
  finally{setSaving(false);}
 };
 const filtered=tools.filter(t=>(t.name+' '+t.description+' '+t.category).toLowerCase().includes(query.toLowerCase()));
 return <div className="mx-auto max-w-6xl px-5 py-8">
  <div className="mb-7"><p className="text-xs font-medium uppercase tracking-[0.16em] text-neutral-500">Administration</p><h1 className="mt-2 text-3xl font-semibold tracking-tight text-black">Tools & permissions</h1><p className="mt-1 text-sm text-neutral-500">Configure what agents can invoke. Server-side policy is authoritative.</p></div>
  {error&&<div className="mb-4 rounded-lg border border-neutral-300 bg-neutral-100 px-4 py-3 text-sm">{error}</div>}
  <div className="mb-4 flex items-center gap-2 rounded-md border border-neutral-300 bg-white px-3"><Search size={15} className="text-neutral-400"/><input value={query} onChange={e=>setQuery(e.target.value)} placeholder="Search tools…" className="h-10 flex-1 bg-transparent text-sm outline-none"/></div>
  <Card><CardHeader><h2 className="text-sm font-semibold">Tool catalog</h2><p className="mt-1 text-xs text-neutral-500">{tools.length} registered tools</p></CardHeader>
   <CardContent className="p-0">{loading?<p className="p-6 text-sm text-neutral-500">Loading…</p>:<div className="divide-y divide-neutral-100">
    {filtered.map(t=><div key={t.id} className="flex flex-wrap items-center justify-between gap-4 px-5 py-4">
      <div className="min-w-0 flex-1"><div className="flex flex-wrap items-center gap-2"><span className="text-sm font-medium text-black">{t.name}</span><Badge>{t.category}</Badge>{t.hardDenied&&<Badge tone="danger">Hard denied</Badge>}</div><p className="mt-1 text-xs text-neutral-500">{t.description}</p><p className="mt-1 font-mono text-[11px] text-neutral-400">Scope: {t.scope} · Risk: {t.risk}</p></div>
      <div className="flex items-center gap-3"><Badge tone={t.policy==='approval_required'?'warning':t.policy==='disabled'||t.policy==='hard_denied'?'danger':'default'}>{t.policy.replace('_',' ')}</Badge><Button size="sm" variant="secondary" disabled={t.hardDenied} onClick={()=>openEdit(t)}><Settings2 size={13}/>Configure</Button></div>
    </div>)}
    {filtered.length===0&&<p className="p-6 text-sm text-neutral-500">No matching tools.</p>}
   </div>}</CardContent>
  </Card>
  <div className="mt-4 flex items-start gap-2 text-xs text-neutral-500"><ShieldAlert size={15} className="mt-0.5 shrink-0"/>Hard-denied host filesystem and Podman socket access cannot be enabled or approved. Approval-required actions execute through the host manager after review.</div>
  <Dialog open={!!editing} onClose={()=>setEditing(null)} title={editing?'Configure '+editing.name:'Configure tool'} description="Policy is enforced by the backend for subsequent agent tool calls.">
   {editing&&<div className="space-y-4"><div><FieldLabel>Tool policy</FieldLabel><Select value={policy} onChange={e=>setPolicy(e.target.value)}><option value="automatic">Automatic</option><option value="approval_required">Approval required</option><option value="disabled">Disabled</option></Select></div><label className="flex items-center gap-2 text-sm text-neutral-700"><input type="checkbox" checked={enabled} onChange={e=>setEnabled(e.target.checked)} className="accent-black"/>Tool enabled</label><div className="flex justify-end gap-2 pt-3"><Button variant="secondary" onClick={()=>setEditing(null)}>Cancel</Button><Button disabled={saving} onClick={()=>void save()}>{saving?'Saving…':'Save policy'}</Button></div></div>}
  </Dialog>
 </div>;
};
