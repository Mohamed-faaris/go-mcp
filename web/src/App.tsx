import React,{useEffect,useState} from 'react';
import {Navbar} from './components/Navbar';
import {LoginPage} from './pages/LoginPage';
import {TaskDashboardPage} from './pages/TaskDashboardPage';
import {TasksPage} from './pages/TasksPage';
import {TaskDetailPage} from './pages/TaskDetailPage';
import {ProjectsPage} from './pages/ProjectsPage';
import {ContainersPage} from './pages/ContainersPage';
import {ContainerDetailPage} from './pages/ContainerDetailPage';
import {ToolsPage} from './pages/ToolsPage';
import {ApprovalsPage} from './pages/ApprovalsPage';
import {AgentsPage} from './pages/AgentsPage';
import {SettingsPage} from './pages/SettingsPage';
import {DevicesPage} from './pages/DevicesPage';
import {ImagesPage} from './pages/ImagesPage';
import {getStoredAuth,setStoredAuth,apiRequest} from './api';
import {AuthUser} from './types';

export const App:React.FC=()=>{
 const [currentRoute,setCurrentRoute]=useState(window.location.hash||'#/');const [user,setUser]=useState<AuthUser|null>(null);const [authChecked,setAuthChecked]=useState(false);
 useEffect(()=>{const onHash=()=>setCurrentRoute(window.location.hash||'#/');window.addEventListener('hashchange',onHash);return()=>window.removeEventListener('hashchange',onHash);},[]);
 useEffect(()=>{
  const stored=getStoredAuth();
  if(!stored){setAuthChecked(true);if(window.location.hash!=='#/login')window.location.hash='#/login';return;}
  apiRequest<{username:string;role:'admin'|'viewer'}>('/api/auth/me')
   .then(v=>setUser({username:v.username,role:v.role}))
   .catch(()=>{setStoredAuth(null);window.location.hash='#/login';})
   .finally(()=>setAuthChecked(true));
 },[]);
 const logout=()=>{setStoredAuth(null);setUser(null);window.location.hash='#/login';};
 const onLogin=(v:AuthUser)=>setUser(v);
 if(!authChecked)return <div className="flex min-h-screen items-center justify-center bg-white text-sm text-neutral-500">Authenticating…</div>;
 if(currentRoute==='#/login'||!user)return <LoginPage onLoginSuccess={onLogin}/>;
 let content:React.ReactNode;
 if(currentRoute==='#/'||currentRoute==='')content=<TaskDashboardPage/>;
 else if(currentRoute==='#/tasks')content=<TasksPage/>;
 else if(currentRoute.startsWith('#/tasks/'))content=<TaskDetailPage taskId={currentRoute.slice('#/tasks/'.length)}/>;
 else if(currentRoute.startsWith('#/projects'))content=<ProjectsPage user={user}/>;
 else if(currentRoute==='#/containers')content=<ContainersPage user={user}/>;
 else if(currentRoute.startsWith('#/containers/'))content=<ContainerDetailPage containerId={currentRoute.slice('#/containers/'.length)} user={user}/>;
 else if(currentRoute.startsWith('#/tools'))content=<ToolsPage/>;
 else if(currentRoute.startsWith('#/approvals'))content=<ApprovalsPage/>;
 else if(currentRoute.startsWith('#/agents'))content=<AgentsPage/>;
 else if(currentRoute.startsWith('#/settings'))content=<SettingsPage/>;
 else if(currentRoute.startsWith('#/images'))content=<ImagesPage user={user}/>;
 else if(currentRoute.startsWith('#/devices'))content=<DevicesPage user={user}/>;
 else content=<TaskDashboardPage/>;
 return <div className="min-h-screen bg-neutral-50 text-neutral-950"><Navbar currentRoute={currentRoute} user={user} onLogout={logout}/><main className="min-h-[calc(100vh-64px)]">{content}</main><footer className="border-t border-neutral-200 px-5 py-5 text-center text-[11px] text-neutral-400">go-mcp · Container-native development agents</footer></div>;
};
