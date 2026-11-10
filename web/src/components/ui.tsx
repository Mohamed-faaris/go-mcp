import React, { useEffect } from 'react';

type ButtonProps = React.ButtonHTMLAttributes<HTMLButtonElement> & { variant?: 'primary'|'secondary'|'ghost'|'danger'; size?: 'sm'|'md' };
export const Button: React.FC<ButtonProps> = ({variant='primary',size='md',className='',...props}) => {
  const styles = {
    primary: 'bg-black text-white hover:bg-neutral-800 border border-black',
    secondary: 'bg-white text-neutral-900 border border-neutral-300 hover:bg-neutral-100',
    ghost: 'bg-transparent text-neutral-700 border border-transparent hover:bg-neutral-100',
    danger: 'bg-white text-neutral-900 border border-neutral-300 hover:border-black hover:bg-neutral-100',
  };
  const sizeClass = size==='sm'?'px-3 py-1.5 text-xs':'px-4 py-2 text-sm';
  return <button className={'inline-flex items-center justify-center gap-2 rounded-md font-medium transition disabled:opacity-40 disabled:cursor-not-allowed '+sizeClass+' '+styles[variant]+' '+className} {...props}/>;
};

export const Card: React.FC<React.HTMLAttributes<HTMLDivElement>> = ({className='',...props}) =>
  <div className={'rounded-xl border border-neutral-200 bg-white shadow-sm '+className} {...props}/>;
export const CardHeader: React.FC<React.HTMLAttributes<HTMLDivElement>> = ({className='',...props}) =>
  <div className={'border-b border-neutral-100 px-5 py-4 '+className} {...props}/>;
export const CardContent: React.FC<React.HTMLAttributes<HTMLDivElement>> = ({className='',...props}) =>
  <div className={'p-5 '+className} {...props}/>;

export const Badge: React.FC<{children:React.ReactNode; tone?:'default'|'success'|'warning'|'danger'}> = ({children,tone='default'}) => {
  const styles={default:'bg-neutral-100 text-neutral-700',success:'bg-neutral-900 text-white',warning:'bg-neutral-200 text-neutral-900',danger:'bg-white border border-neutral-400 text-neutral-900'};
  return <span className={'inline-flex items-center rounded-full px-2.5 py-1 text-[11px] font-medium '+styles[tone]}>{children}</span>;
};

export const Dialog: React.FC<{open:boolean;title:string;description?:string;onClose:()=>void;children:React.ReactNode;wide?:boolean}> = ({open,title,description,onClose,children,wide=false}) => {
  useEffect(()=>{
    if(!open)return;
    const listener=(event:KeyboardEvent)=>{if(event.key==='Escape')onClose();};
    window.addEventListener('keydown',listener);
    return ()=>window.removeEventListener('keydown',listener);
  },[open,onClose]);
  if(!open)return null;
  return <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" role="presentation" onMouseDown={(e)=>{if(e.target===e.currentTarget)onClose();}}>
    <section role="dialog" aria-modal="true" aria-labelledby="dialog-title" className={'max-h-[90vh] w-full overflow-y-auto rounded-xl border border-neutral-200 bg-white shadow-2xl '+(wide?'max-w-2xl':'max-w-lg')}>
      <header className="flex items-start justify-between border-b border-neutral-200 px-6 py-4">
        <div><h2 id="dialog-title" className="text-base font-semibold text-black">{title}</h2>{description&&<p className="mt-1 text-sm text-neutral-500">{description}</p>}</div>
        <button onClick={onClose} aria-label="Close dialog" className="rounded-md px-2 py-1 text-neutral-500 hover:bg-neutral-100">×</button>
      </header>
      <div className="p-6">{children}</div>
    </section>
  </div>;
};

export const Collapsible: React.FC<{title:string;description?:string;defaultOpen?:boolean;children:React.ReactNode}> = ({title,description,defaultOpen=false,children}) => {
  const [open,setOpen]=React.useState(defaultOpen);
  return <section className="rounded-lg border border-neutral-200">
    <button type="button" aria-expanded={open} onClick={()=>setOpen(v=>!v)} className="flex w-full items-center justify-between gap-4 px-4 py-3 text-left">
      <span><span className="block text-sm font-medium text-neutral-900">{title}</span>{description&&<span className="mt-0.5 block text-xs text-neutral-500">{description}</span>}</span>
      <span className="text-neutral-500">{open?'−':'+'}</span>
    </button>
    {open&&<div className="border-t border-neutral-200 p-4">{children}</div>}
  </section>;
};

export const FieldLabel: React.FC<React.LabelHTMLAttributes<HTMLLabelElement>> = ({className='',...props}) =>
  <label className={'mb-1.5 block text-xs font-medium text-neutral-700 '+className} {...props}/>;
export const TextInput: React.FC<React.InputHTMLAttributes<HTMLInputElement>> = ({className='',...props}) =>
  <input className={'w-full rounded-md border border-neutral-300 bg-white px-3 py-2 text-sm text-neutral-900 outline-none placeholder:text-neutral-400 focus:border-black focus:ring-1 focus:ring-black '+className} {...props}/>;
export const TextArea: React.FC<React.TextareaHTMLAttributes<HTMLTextAreaElement>> = ({className='',...props}) =>
  <textarea className={'w-full rounded-md border border-neutral-300 bg-white px-3 py-2 text-sm text-neutral-900 outline-none placeholder:text-neutral-400 focus:border-black focus:ring-1 focus:ring-black '+className} {...props}/>;
export const Select: React.FC<React.SelectHTMLAttributes<HTMLSelectElement>> = ({className='',...props}) =>
  <select className={'w-full rounded-md border border-neutral-300 bg-white px-3 py-2 text-sm text-neutral-900 outline-none focus:border-black focus:ring-1 focus:ring-black '+className} {...props}/>;
