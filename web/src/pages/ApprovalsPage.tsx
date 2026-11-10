import React,{useCallback,useEffect,useState} from 'react';
import {Check,Clock3,X} from 'lucide-react';
import {apiRequest} from '../api';
import {ApprovalItem} from '../types';
import {Badge,Button,Card,CardContent,CardHeader} from '../components/ui';

export const ApprovalsPage:React.FC=()=>{
 const [items,setItems]=useState<ApprovalItem[]>([]);
 const [loading,setLoading]=useState(true);
 const [error,setError]=useState('');
 const load=useCallback(async()=>{
  try{const r=await apiRequest<{approvals:ApprovalItem[]}>('/api/v1/approvals?status=pending');setItems(r.approvals||[]);}
  catch(e:any){setError(e.message||'Unable to load approval requests');}
  finally{setLoading(false);}
 },[]);
 useEffect(()=>{void load();},[load]);
 const review=async(id:string,decision:'approve'|'deny')=>{
  try{await apiRequest('/api/v1/approvals/'+encodeURIComponent(id)+'/'+decision,{method:'POST'});await load();}
  catch(e:any){setError(e.message||'Approval action failed');}
 };
 return <div className="mx-auto max-w-5xl px-5 py-8">
  <div className="mb-7"><p className="text-xs font-medium uppercase tracking-[0.16em] text-neutral-500">Review queue</p><h1 className="mt-2 text-3xl font-semibold tracking-tight text-black">Approvals</h1><p className="mt-1 text-sm text-neutral-500">High-impact actions requested by task agents.</p></div>
  {error&&<div className="mb-4 rounded-lg border border-neutral-300 bg-neutral-100 px-4 py-3 text-sm">{error}</div>}
  <Card><CardHeader><div className="flex items-center justify-between"><h2 className="text-sm font-semibold">Pending requests</h2><Badge>{items.length} pending</Badge></div></CardHeader>
   <CardContent className="p-0">{loading?<p className="p-6 text-sm text-neutral-500">Loading…</p>:items.length===0?<div className="p-12 text-center"><Clock3 size={20} className="mx-auto text-neutral-400"/><p className="mt-3 text-sm font-medium text-black">No pending approvals</p><p className="mt-1 text-xs text-neutral-500">Requests from agents will appear here.</p></div>:
    <div className="divide-y divide-neutral-100">{items.map(a=><div key={a.id} className="px-5 py-5">
     <div className="flex flex-wrap items-start justify-between gap-4"><div className="min-w-0"><div className="flex items-center gap-2"><Badge tone="warning">Approval required</Badge><span className="text-xs text-neutral-500">{a.toolId}</span></div><h3 className="mt-2 text-sm font-medium text-black">{a.summary}</h3><p className="mt-1 break-all font-mono text-xs text-neutral-500">Resource: {a.resourceType} · {a.resourceId}</p><p className="mt-1 text-xs text-neutral-500">Task {a.taskId} · Requested by {a.requestedBy} · Expires {new Date(a.expiresAt).toLocaleString()}</p></div>
      <div className="flex gap-2"><Button variant="secondary" size="sm" onClick={()=>void review(a.id,'deny')}><X size={13}/>Deny</Button><Button size="sm" onClick={()=>void review(a.id,'approve')}><Check size={13}/>Approve & execute</Button></div>
     </div>
    </div>)}</div>}
   </CardContent>
  </Card>
 </div>;
};
