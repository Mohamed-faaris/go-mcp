import React,{useCallback,useEffect,useState} from 'react';
import {Search,ArrowUpRight} from 'lucide-react';
import {apiRequest} from '../api';
import {TaskItem} from '../types';
import {Badge,Card,CardContent,CardHeader,TextInput} from '../components/ui';

export const TasksPage:React.FC=()=>{
 const [tasks,setTasks]=useState<TaskItem[]>([]);const [loading,setLoading]=useState(true);const [query,setQuery]=useState('');const [status,setStatus]=useState('all');const [error,setError]=useState('');
 const load=useCallback(async()=>{try{const r=await apiRequest<{tasks:TaskItem[]}>('/api/v1/tasks');setTasks(r.tasks||[]);}catch(e:any){setError(e.message||'Unable to load tasks');}finally{setLoading(false);}},[]);
 useEffect(()=>{void load();const timer=window.setInterval(()=>void load(),6000);return()=>window.clearInterval(timer);},[load]);
 const visible=tasks.filter(t=>(status==='all'||t.status===status)&&(t.prompt+' '+t.projectPath+' '+t.presetId).toLowerCase().includes(query.toLowerCase()));
 return <div className="mx-auto max-w-6xl px-5 py-8">
  <div className="mb-7"><p className="text-xs font-medium uppercase tracking-[0.16em] text-neutral-500">Workspace</p><h1 className="mt-2 text-3xl font-semibold tracking-tight text-black">Tasks</h1><p className="mt-1 text-sm text-neutral-500">Execution history and active agent work.</p></div>
  {error&&<div className="mb-4 rounded-lg border border-neutral-300 bg-neutral-100 px-4 py-3 text-sm">{error}</div>}
  <div className="mb-4 flex flex-wrap gap-2"><div className="flex min-w-[240px] flex-1 items-center gap-2 rounded-md border border-neutral-300 px-3"><Search size={15} className="text-neutral-400"/><input value={query} onChange={e=>setQuery(e.target.value)} placeholder="Search tasks…" className="h-10 w-full bg-transparent text-sm outline-none"/></div><select value={status} onChange={e=>setStatus(e.target.value)} className="rounded-md border border-neutral-300 bg-white px-3 text-sm"><option value="all">All statuses</option>{['queued','provisioning','running','waiting_approval','validating','completed','failed','cancelled'].map(s=><option key={s} value={s}>{s.replace('_',' ')}</option>)}</select></div>
  <Card><CardHeader><h2 className="text-sm font-semibold">All tasks</h2><p className="mt-1 text-xs text-neutral-500">{visible.length} results</p></CardHeader><CardContent className="p-0">{loading?<p className="p-6 text-sm text-neutral-500">Loading…</p>:visible.length===0?<p className="p-10 text-center text-sm text-neutral-500">No tasks match this filter.</p>:<div className="divide-y divide-neutral-100">{visible.map(t=><a href={'#/tasks/'+t.id} key={t.id} className="flex flex-wrap items-center justify-between gap-3 px-5 py-4 hover:bg-neutral-50"><div className="min-w-0"><div className="flex flex-wrap items-center gap-2"><span className="truncate text-sm font-medium text-black">{t.prompt}</span><Badge>{t.status.replace('_',' ')}</Badge></div><p className="mt-1 truncate text-xs text-neutral-500">{t.projectPath} · {t.presetId} · {new Date(t.createdAt).toLocaleString()}</p>{t.branchName&&<p className="mt-1 font-mono text-[11px] text-neutral-400">{t.branchName}</p>}</div><ArrowUpRight size={15} className="text-neutral-400"/></a>)}</div>}</CardContent></Card>
 </div>;
};
