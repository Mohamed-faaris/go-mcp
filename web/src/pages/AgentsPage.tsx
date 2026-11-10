import React,{useCallback,useEffect,useState} from 'react';
import {apiRequest} from '../api';
import {TaskItem,AgentSession,AgentMessage} from '../types';
import {Button,Card,CardContent,CardHeader,FieldLabel,Select,TextArea,TextInput} from '../components/ui';

export const AgentsPage:React.FC=()=>{
 const [tasks,setTasks]=useState<TaskItem[]>([]);
 const [projectId,setProjectId]=useState('');
 const [agents,setAgents]=useState<AgentSession[]>([]);
 const [messages,setMessages]=useState<AgentMessage[]>([]);
 const [sessionId,setSessionId]=useState('');
 const [label,setLabel]=useState('web-agent');
 const [body,setBody]=useState('');
 const [error,setError]=useState('');
 const loadTasks=useCallback(async()=>{
  try{const r=await apiRequest<{tasks:TaskItem[]}>('/api/v1/tasks');setTasks(r.tasks||[]);if(!projectId&&r.tasks?.length)setProjectId(r.tasks[0].projectId);}
  catch(e:any){setError(e.message||'Unable to load task list');}
 },[projectId]);
 useEffect(()=>{void loadTasks();},[loadTasks]);
 const refresh=useCallback(async()=>{
  if(!projectId)return;
  try{
   const pair=await Promise.all([
    apiRequest<{agents:AgentSession[]}>('/api/v1/projects/'+encodeURIComponent(projectId)+'/agents'),
    apiRequest<{messages:AgentMessage[]}>('/api/v1/projects/'+encodeURIComponent(projectId)+'/messages')
   ]);
   setAgents(pair[0].agents||[]);setMessages(pair[1].messages||[]);
  }catch(e:any){setError(e.message||'Unable to load collaboration data');}
 },[projectId]);
 useEffect(()=>{void refresh();const timer=window.setInterval(()=>void refresh(),5000);return()=>window.clearInterval(timer);},[refresh]);
 const register=async()=>{
  const linked=tasks.find(t=>t.projectId===projectId);if(!linked)return;
  try{
   const r=await apiRequest<{session:AgentSession}>('/api/v1/agent-sessions',{method:'POST',body:JSON.stringify({projectId,taskId:linked.id,agentLabel:label,workstreamId:'web-agent'})});
   setSessionId(r.session.id);await refresh();
  }catch(e:any){setError(e.message||'Could not register agent session');}
 };
 const send=async(e:React.FormEvent)=>{
  e.preventDefault();if(!sessionId||!body.trim())return;
  try{
   await apiRequest('/api/v1/projects/'+encodeURIComponent(projectId)+'/messages',{method:'POST',body:JSON.stringify({senderSessionId:sessionId,taskId:tasks.find(t=>t.projectId===projectId)?.id,channel:'project:'+projectId,type:'progress',body:body.trim()})});
   setBody('');await refresh();
  }catch(e:any){setError(e.message||'Could not send message');}
 };
 const handoff=async()=>{
  if(!sessionId)return;
  try{
   await apiRequest('/api/v1/agent-sessions/'+encodeURIComponent(sessionId)+'/handoffs',{method:'POST',body:JSON.stringify({goal:'Continue project work',summary:'Context saved from the agent collaboration page.',nextActions:['Read repository instructions and verify the latest branch and commit.']})});
   await refresh();
  }catch(e:any){setError(e.message||'Could not create handoff');}
 };
 const bootstrap='Read AGENTS.md, REWORK_PLAN.md, ARCHITECTURE.md, docs/agents/COORDINATION.md and docs/agents/WORKSTREAMS.md. Resume project '+projectId+' using the latest handoff and Git state. Register a new agent session and coordinate through project messages and work claims.';
 return <div className="mx-auto max-w-6xl px-5 py-8">
  <div className="mb-7"><p className="text-xs font-medium uppercase tracking-[0.16em] text-neutral-500">Collaboration</p><h1 className="mt-2 text-3xl font-semibold tracking-tight text-black">Agents & context</h1><p className="mt-1 text-sm text-neutral-500">Persisted sessions, project messages, and handoffs across separate web-agent chats.</p></div>
  {error&&<div className="mb-4 rounded-lg border border-neutral-300 bg-neutral-100 px-4 py-3 text-sm">{error}</div>}
  <div className="mb-5 max-w-xl"><FieldLabel>Project</FieldLabel><Select value={projectId} onChange={e=>setProjectId(e.target.value)}><option value="">Select project</option>{Array.from(new Map(tasks.map(t=>[t.projectId,t])).values()).map(t=><option key={t.projectId} value={t.projectId}>{t.projectPath||t.projectId}</option>)}</Select></div>
  <div className="grid gap-5 lg:grid-cols-[1fr_1.2fr]">
   <Card><CardHeader><h2 className="text-sm font-semibold">Registered agents</h2></CardHeader><CardContent>
    {agents.length===0?<p className="text-sm text-neutral-500">No sessions registered for this project yet.</p>:<div className="space-y-3">{agents.map(a=><div key={a.id} className="rounded-lg border border-neutral-200 p-3"><div className="flex justify-between gap-2"><span className="text-sm font-medium">{a.agentLabel}</span><span className="text-xs text-neutral-500">{a.status}</span></div><p className="mt-1 break-all font-mono text-[11px] text-neutral-500">{a.branch||'No branch'} · {a.commit?.slice(0,8)||'no commit'}</p><p className="mt-1 text-[11px] text-neutral-400">Heartbeat {new Date(a.lastHeartbeatAt).toLocaleString()}</p></div>)}</div>}
    <div className="mt-5 border-t border-neutral-100 pt-4"><FieldLabel>Register this web-agent session</FieldLabel><TextInput value={label} onChange={e=>setLabel(e.target.value)} placeholder="Agent label"/><Button className="mt-2" onClick={()=>void register()} disabled={!projectId||!tasks.some(t=>t.projectId===projectId)}>Register session</Button></div>
    {sessionId&&<div className="mt-4 rounded-lg bg-neutral-50 p-3"><p className="text-xs font-medium">Current session ID</p><p className="mt-1 break-all font-mono text-[11px] text-neutral-600">{sessionId}</p><Button size="sm" variant="secondary" className="mt-2" onClick={()=>void handoff()}>Save context handoff</Button><Button size="sm" variant="ghost" className="mt-2" onClick={()=>void navigator.clipboard.writeText(bootstrap)}>Copy bootstrap prompt</Button></div>}
   </CardContent></Card>
   <Card><CardHeader><h2 className="text-sm font-semibold">Project messages</h2><p className="mt-1 text-xs text-neutral-500">Messages persist in the manager store, not in a chat transcript.</p></CardHeader><CardContent>
    <div className="mb-4 max-h-[360px] space-y-3 overflow-y-auto">{messages.length===0?<p className="text-sm text-neutral-500">No messages yet.</p>:messages.map(m=><div key={m.id} className="rounded-lg border border-neutral-200 p-3"><div className="flex justify-between gap-2"><span className="text-xs font-medium">{m.type} · {m.senderSessionId.slice(0,8)}</span><span className="text-[10px] text-neutral-400">{new Date(m.createdAt).toLocaleTimeString()}</span></div><p className="mt-2 whitespace-pre-wrap text-sm text-neutral-700">{m.body}</p></div>)}</div>
    <form onSubmit={send} className="space-y-2"><FieldLabel>Send project message</FieldLabel><TextArea rows={3} value={body} onChange={e=>setBody(e.target.value)} placeholder="Progress, blocker, decision proposal, or handoff note…"/><Button type="submit" disabled={!sessionId||!body.trim()}>Send message</Button></form>
   </CardContent></Card>
  </div>
 </div>;
};
