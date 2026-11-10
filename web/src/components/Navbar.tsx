import React from 'react';
import {Activity,FolderGit2,Box,Users,ShieldCheck,Wrench,Settings,LogOut,Plus} from 'lucide-react';
import {AuthUser} from '../types';

interface NavbarProps {currentRoute:string;user:AuthUser|null;onLogout:()=>void}
export const Navbar:React.FC<NavbarProps>=({currentRoute,user,onLogout})=>{
 const items=[
  {label:'Overview',route:'#/',icon:Activity},
  {label:'Tasks',route:'#/tasks',icon:FolderGit2},
  {label:'Projects',route:'#/projects',icon:FolderGit2},
  {label:'Containers',route:'#/containers',icon:Box},
  {label:'Agents',route:'#/agents',icon:Users},
  {label:'Approvals',route:'#/approvals',icon:ShieldCheck},
  {label:'Tools',route:'#/tools',icon:Wrench},
  {label:'Settings',route:'#/settings',icon:Settings},
 ];
 return <header className="sticky top-0 z-40 border-b border-neutral-200 bg-white/95 backdrop-blur">
  <div className="mx-auto flex h-16 max-w-7xl items-center justify-between gap-4 px-5">
   <a href="#/" className="flex shrink-0 items-center gap-2"><div className="flex h-8 w-8 items-center justify-center rounded-lg bg-black text-white"><span className="text-sm font-semibold">g</span></div><span className="text-sm font-semibold tracking-tight text-black">go-mcp</span></a>
   <nav className="hidden items-center gap-1 overflow-x-auto lg:flex">{items.map(item=>{const Icon=item.icon;const active=item.route==='#/'?currentRoute==='#/'||currentRoute==='':currentRoute.startsWith(item.route);return <a key={item.route} href={item.route} className={'inline-flex items-center gap-2 whitespace-nowrap rounded-md px-3 py-2 text-xs font-medium transition '+(active?'bg-black text-white':'text-neutral-600 hover:bg-neutral-100 hover:text-black')}><Icon size={14}/>{item.label}</a>;})}</nav>
   <div className="flex shrink-0 items-center gap-2"><a href="#/" onClick={e=>{e.preventDefault();window.dispatchEvent(new HashChangeEvent('hashchange'));}} className="hidden rounded-md bg-black px-3 py-2 text-xs font-medium text-white hover:bg-neutral-800 sm:inline-flex"><Plus size={14} className="mr-1"/>New task</a>{user&&<span className="hidden max-w-28 truncate border-l border-neutral-200 pl-3 text-xs text-neutral-500 md:inline">{user.username}</span>}<button onClick={onLogout} title="Sign out" className="rounded-md p-2 text-neutral-500 hover:bg-neutral-100 hover:text-black"><LogOut size={15}/></button></div>
  </div>
  <nav className="flex gap-1 overflow-x-auto border-t border-neutral-100 px-4 py-2 lg:hidden">{items.map(item=>{const active=item.route==='#/'?currentRoute==='#/':currentRoute.startsWith(item.route);return <a key={item.route} href={item.route} className={'whitespace-nowrap rounded-md px-3 py-1.5 text-xs '+(active?'bg-black text-white':'text-neutral-600 hover:bg-neutral-100')}>{item.label}</a>;})}</nav>
 </header>;
};
