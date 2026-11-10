import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { ArrowUpRight, Plus, RefreshCw, FolderGit2 } from 'lucide-react';
import { apiRequest } from '../api';
import { ProjectItem, TaskItem, RuntimePreset } from '../types';
import { Badge, Button, Card, CardContent, CardHeader, Collapsible, Dialog, FieldLabel, Select, TextArea } from '../components/ui';

export const TaskDashboardPage: React.FC = () => {
  const [tasks,setTasks]=useState<TaskItem[]>([]);
  const [projects,setProjects]=useState<ProjectItem[]>([]);
  const [presets,setPresets]=useState<RuntimePreset[]>([]);
  const [loading,setLoading]=useState(true);
  const [error,setError]=useState('');
  const [dialogOpen,setDialogOpen]=useState(false);
  const [projectPath,setProjectPath]=useState('');
  const [prompt,setPrompt]=useState('');
  const [presetId,setPresetId]=useState('');
  const [keepContainer,setKeepContainer]=useState(false);
  const [submitting,setSubmitting]=useState(false);

  const load=useCallback(async()=>{
    setError('');
    try{
      const [taskRes,projectRes,presetRes]=await Promise.all([
        apiRequest<{tasks:TaskItem[]}>('/api/v1/tasks'),
        apiRequest<{items:ProjectItem[]}>('/api/projects'),
        apiRequest<{presets:RuntimePreset[]}>('/api/v1/presets'),
      ]);
      setTasks(taskRes.tasks||[]);
      setProjects(projectRes.items||[]);
      setPresets(presetRes.presets||[]);
      if(!projectPath&&projectRes.items?.length)setProjectPath(projectRes.items[0].path);
      if(!presetId&&presetRes.presets?.length)setPresetId(presetRes.presets[0].id);
    }catch(e:any){setError(e.message||'Unable to load dashboard data');}
    finally{setLoading(false);}
  },[projectPath,presetId]);

  useEffect(()=>{void load();},[load]);
  useEffect(()=>{
    const timer=window.setInterval(()=>{void apiRequest<{tasks:TaskItem[]}>('/api/v1/tasks').then(v=>setTasks(v.tasks||[])).catch(()=>{});},5000);
    return ()=>window.clearInterval(timer);
  },[]);

  useEffect(()=>{
    if(!projectPath)return;
    apiRequest<{detectedPreset:RuntimePreset;presetDetected:boolean}>('/api/v1/projects/git?path='+encodeURIComponent(projectPath))
      .then(v=>{if(v.presetDetected&&v.detectedPreset?.id)setPresetId(v.detectedPreset.id);}).catch(()=>{});
  },[projectPath]);

  const counts=useMemo(()=>({
    running:tasks.filter(t=>['running','provisioning','validating'].includes(t.status)).length,
    queued:tasks.filter(t=>t.status==='queued').length,
    approval:tasks.filter(t=>t.status==='waiting_approval').length,
    completed:tasks.filter(t=>t.status==='completed').length,
  }),[tasks]);

  const createTask=async(e:React.FormEvent)=>{
    e.preventDefault();if(!projectPath||!prompt.trim())return;
    setSubmitting(true);setError('');
    try{
      const res=await apiRequest<{task:TaskItem}>('/api/v1/tasks',{method:'POST',body:JSON.stringify({projectPath,prompt:prompt.trim(),presetId,keepContainer})});
      setDialogOpen(false);setPrompt('');window.location.hash='#/tasks/'+res.task.id;
    }catch(e:any){setError(e.message||'Task could not be created');}
    finally{setSubmitting(false);}
  };

  const badgeTone=(status:string)=>status==='completed'?'success':status==='failed'||status==='cancelled'?'danger':status==='waiting_approval'?'warning':'default';
  return <div className="mx-auto max-w-6xl px-5 py-8">
    <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
      <div><p className="text-xs font-medium uppercase tracking-[0.16em] text-neutral-500">Workspace</p><h1 className="mt-2 text-3xl font-semibold tracking-tight text-black">Overview</h1><p className="mt-1 text-sm text-neutral-500">Autonomous development tasks, project changes, and validation.</p></div>
      <div className="flex gap-2"><Button variant="secondary" onClick={()=>void load()}><RefreshCw size={14}/>Refresh</Button><Button onClick={()=>setDialogOpen(true)}><Plus size={15}/>New task</Button></div>
    </div>
    {error&&<div className="mb-5 rounded-lg border border-neutral-300 bg-neutral-100 px-4 py-3 text-sm text-neutral-800">{error}</div>}
    <div className="mb-8 grid grid-cols-2 gap-3 md:grid-cols-4">
      {[
        ['Running',counts.running],['Queued',counts.queued],['Waiting approval',counts.approval],['Completed',counts.completed],
      ].map(([label,value])=><Card key={label as string}><CardContent className="p-4"><p className="text-xs text-neutral-500">{label}</p><p className="mt-2 text-2xl font-semibold tabular-nums text-black">{value}</p></CardContent></Card>)}
    </div>
    <Card>
      <CardHeader className="flex flex-row items-center justify-between"><div><h2 className="text-sm font-semibold text-black">Recent tasks</h2><p className="mt-1 text-xs text-neutral-500">Latest activity across your projects</p></div><a href="#/tasks" className="text-xs font-medium text-neutral-600 hover:text-black">All tasks <ArrowUpRight size={13} className="inline"/></a></CardHeader>
      <CardContent className="p-0">
        {loading?<div className="p-8 text-center text-sm text-neutral-500">Loading tasks…</div>:tasks.length===0?<div className="px-6 py-14 text-center"><div className="mx-auto flex h-11 w-11 items-center justify-center rounded-full bg-neutral-100"><FolderGit2 size={19}/></div><h3 className="mt-4 text-sm font-semibold text-black">No tasks yet</h3><p className="mt-1 text-sm text-neutral-500">Choose a project folder and describe the work to start your first agent.</p><Button className="mt-5" onClick={()=>setDialogOpen(true)}><Plus size={14}/>Create task</Button></div>:
        <div className="divide-y divide-neutral-100">{tasks.slice(0,12).map(task=><a key={task.id} href={'#/tasks/'+task.id} className="flex flex-wrap items-center justify-between gap-3 px-5 py-4 transition hover:bg-neutral-50">
          <div className="min-w-0"><div className="flex items-center gap-2"><span className="truncate text-sm font-medium text-black">{task.prompt}</span><Badge tone={badgeTone(task.status)}>{task.status.replace('_',' ')}</Badge></div><div className="mt-1 flex flex-wrap gap-3 text-xs text-neutral-500"><span>{task.presetId}</span>{task.branchName&&<span className="font-mono">{task.branchName}</span>}<span>{new Date(task.createdAt).toLocaleString()}</span></div></div>
          <ArrowUpRight size={15} className="text-neutral-400"/>
        </a>)}</div>}
      </CardContent>
    </Card>
    <Dialog open={dialogOpen} onClose={()=>setDialogOpen(false)} title="Create development task" description="A fresh, isolated container and workspace will be created for this task." wide>
      <form onSubmit={createTask} className="space-y-5">
        <div><FieldLabel>Project folder</FieldLabel><Select value={projectPath} onChange={e=>setProjectPath(e.target.value)} required><option value="">Select a project folder</option>{projects.map(p=><option key={p.path} value={p.path}>{p.name} · {p.type}</option>)}</Select>{projects.length===0&&<p className="mt-1 text-xs text-neutral-500">No project folders found in configured Projects or Scratch directories.</p>}</div>
        <div><FieldLabel>What should the agent do?</FieldLabel><TextArea rows={5} value={prompt} onChange={e=>setPrompt(e.target.value)} placeholder="Example: Add password reset to the auth flow. Inspect existing patterns, implement the feature, and run tests." required/></div>
        <div><FieldLabel>Runtime preset</FieldLabel><Select value={presetId} onChange={e=>setPresetId(e.target.value)}>{presets.map(p=><option key={p.id} value={p.id}>{p.name}</option>)}</Select><p className="mt-1 text-xs text-neutral-500">{presets.find(p=>p.id===presetId)?.description||'Choose the development environment for the task.'}</p></div>
        <Collapsible title="Advanced options" description="Container retention and less common settings">
          <label className="flex items-start gap-2 text-sm text-neutral-700"><input type="checkbox" checked={keepContainer} onChange={e=>setKeepContainer(e.target.checked)} className="mt-0.5 accent-black"/><span>Keep the container after completion for inspection</span></label>
          <p className="mt-2 text-xs text-neutral-500">GitHub credentials are managed by the host. A repository remote is detected automatically. Push actions require approval.</p>
        </Collapsible>
        <div className="flex justify-end gap-2 border-t border-neutral-100 pt-4"><Button type="button" variant="secondary" onClick={()=>setDialogOpen(false)}>Cancel</Button><Button type="submit" disabled={submitting||!projectPath||!prompt.trim()}>{submitting?'Starting…':'Start task'}</Button></div>
      </form>
    </Dialog>
  </div>;
};
