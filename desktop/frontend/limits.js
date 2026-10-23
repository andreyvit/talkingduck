import {viewVisible} from './visibility.js';
import {Events} from '/wails/runtime.js';
import {visibleProviders,remaining,resetLabel,pingReading,providerNames,limitsUpdateDelay} from './limits-model.mjs';
const strip=document.getElementById('limits');let snapshot={},config={},shown=localStorage.getItem('limits-visible')==='true';strip.hidden=!shown;
export function toggleLimits(){shown=!shown;strip.hidden=!shown;localStorage.setItem('limits-visible',String(shown));render();}
function text(tag,value,cls){const el=document.createElement(tag);el.textContent=value;if(cls)el.className=cls;return el;}
let timer;
function schedule(){clearTimeout(timer);timer=undefined;if(!shown||!viewVisible())return;const delay=limitsUpdateDelay(snapshot,config);if(delay!==null)timer=setTimeout(tick,delay);}
function tick(){
 if(!shown||!viewVisible()){schedule();return;}
 for(const p of visibleProviders(snapshot,config)){
  const detail=strip.querySelector('.'+p.id+' .limit-reset');
  if(detail&&remaining(p)!==null)detail.textContent=(p.state==='stale'?'Stale · ':'')+resetLabel(p.resets_at);
 }
 if(config.ping==='vertical'&&snapshot.ping?.state==='unreachable'){
  const reading=pingReading(snapshot.ping),line=strip.querySelector('.ping-reading');
  if(line){const number=line.querySelector('.ping-number');number.textContent=reading.number;number.style.width=reading.digits+'ch';line.querySelector('.ping-unit').textContent=reading.unit;line.title=reading.label;line.setAttribute('aria-label',reading.label);}
 }
 schedule();
}
function render(){schedule();if(!shown||!viewVisible())return;const parts=[];
 if(config.ping==='vertical'){const ping=text('section','', 'limit-ping');const reading=pingReading(snapshot.ping),line=text('div','','ping-reading'),number=text('span',reading.number,'ping-number'),unit=text('span',reading.unit,'ping-unit');number.style.width=reading.digits+'ch';line.setAttribute('aria-label',reading.label);line.title=reading.label;line.append(number,unit);ping.append(line,text('small',snapshot.ping?.target||config.ping_target||'8.8.8.8'));if(snapshot.ping?.state==='unreachable')ping.classList.add('unreachable');parts.push(ping);}
 for(const p of visibleProviders(snapshot,config)){
  const el=text('section','', 'limit-provider '+p.id);el.title=p.error||((p.state==='stale'?'Last reading: ':'Updated: ')+(p.observed_at?new Date(p.observed_at).toLocaleString():'pending'));
  const label=text('span',providerNames[p.id]||p.id,'limit-name');const progress=text('span','','limit-progress');const fill=text('span','','limit-fill');const value=remaining(p);fill.style.width=(value??0)+'%';progress.append(fill);progress.setAttribute('role',value===null?'img':'meter');progress.setAttribute('aria-label',(providerNames[p.id]||p.id)+(value===null?' weekly limit unavailable':' weekly remaining'));progress.setAttribute('aria-valuemin','0');progress.setAttribute('aria-valuemax','100');if(value!==null)progress.setAttribute('aria-valuenow',value);
  const percentage=text('span',value===null?'—':Math.round(value)+'%','limit-value');const detail=p.state==='stale'?'Stale · '+resetLabel(p.resets_at):value!==null?resetLabel(p.resets_at):!p.logged_in?'Not signed in':({loading:'Loading…',auth:'Usage unavailable',unavailable:'Weekly limit unavailable',error:'Unavailable'})[p.state]||'Unavailable';
  el.append(label,progress,percentage,text('small',detail,'limit-reset'));if(p.state==='stale')el.classList.add('stale');parts.push(el);
 }
 if(!parts.length)parts.push(text('small','Monitoring is off. Configure it in Settings.','limits-empty'));strip.replaceChildren(...parts);
}
Events.On('monitor:snapshot',({data})=>{snapshot=data.snapshot;config=data.config;render()});
window.addEventListener('viewvisibility',render);Events.Emit('monitor:ready');
