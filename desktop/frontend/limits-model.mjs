export const providerNames={codex:'Codex',claude:'Claude',grok:'Grok'};
export function visibleProviders(snapshot,config){return(snapshot?.providers||[]).filter(p=>config?.[p.id]==='on'||(config?.[p.id]!=='off'&&p.recently_used));}
export function remaining(p){return Number.isFinite(p.remaining_percent)?Math.max(0,Math.min(100,p.remaining_percent)):null;}
export function resetLabel(value,now=Date.now()){
 const end=Date.parse(value);if(!Number.isFinite(end))return '';
 const seconds=Math.ceil((end-now)/1000);if(seconds<=0)return 'Reset pending';
 if(seconds<86400)return 'Reset in '+Math.floor(seconds/3600)+':'+String(Math.floor(seconds/60)%60).padStart(2,'0')+':'+String(seconds%60).padStart(2,'0');
 return 'Reset '+new Intl.DateTimeFormat(undefined,{weekday:'short',hour:'2-digit',minute:'2-digit',hour12:false}).format(new Date(end));
}
export function pingLabel(p,now=Date.now()){
 if(p?.state==='reachable'&&Number.isFinite(p.rtt_ms))return Math.round(p.rtt_ms)+' ms ping';
 if(p?.state==='unreachable'&&p.failed_since)return Math.max(0,Math.floor((now-Date.parse(p.failed_since))/1000))+' s no conn';
 return p?.state==='off'?'Ping off':p?.state==='error'?'Ping unavailable':'Checking ping…';
}

export function pingReading(p,now=Date.now()){
 const label=pingLabel(p,now),match=/^(\d+)(.*)$/.exec(label);
 return {label,number:match?.[1]??'—',unit:match?.[2]??' ping',digits:Math.max(3,p?.digits||1,match?.[1]?.length||0)};
}
