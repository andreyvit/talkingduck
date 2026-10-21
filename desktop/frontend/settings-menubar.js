import {Events} from '/wails/runtime.js';
import {PreferenceDraft,AutosaveQueue} from './settings-model.mjs';
const form=document.getElementById('menubar-form');
const fields=[['ping','Ping',[['vertical','Vertical text'],['off','Off']]],['ping_target','Ping server',[['8.8.8.8','Google · 8.8.8.8'],['1.1.1.1','Cloudflare · 1.1.1.1'],['9.9.9.9','Quad9 · 9.9.9.9'],['custom','Custom…']]],...['codex','claude','grok'].map(id=>[id,id[0].toUpperCase()+id.slice(1)+' weekly limit',[['on','On'],['auto','Auto'],['off','Off']]]),['projects','Projects',[['on','Projects with menu bar shortcut'],['off','Off']]]];
const controls={};const draft=new PreferenceDraft(),client=crypto.randomUUID();
const queue=new AutosaveQueue(()=>{if(!draft.value?.ping_target){queue.finished();return;}Events.Emit('settings:menubar',{client,revision:draft.revision,value:draft.value});});
function row(id,label,control){const r=document.createElement('div');r.className='form-row';const l=document.createElement('label');l.htmlFor=control.id='menubar-'+id;l.textContent=label;r.append(l,control);form.append(r);return r;}
for(const [id,label,options]of fields){const select=document.createElement('select');for(const[value,text]of options){const opt=document.createElement('option');opt.value=value;opt.textContent=text;select.append(opt)}controls[id]=select;row(id,label,select);select.onchange=()=>{customRow.hidden=controls.ping_target.value!=='custom';save()};}
const custom=document.createElement('input');custom.type='text';custom.placeholder='IP address or hostname';const customRow=row('custom','Custom server',custom);customRow.hidden=true;
const help=document.createElement('p');help.className='form-help';help.textContent='Auto shows providers used in the last 14 days. Ping Off hides it everywhere and stops probes.';form.append(help);
const error=document.createElement('p');error.className='form-status error';error.setAttribute('role','status');form.append(error);
custom.oninput=save;
form.onsubmit=e=>{e.preventDefault();save()};
function save(){const value=Object.fromEntries(Object.entries(controls).map(([k,v])=>[k,v.value]));if(value.ping_target==='custom'){value.ping_target=custom.value.trim();if(!value.ping_target){draft.edit(value);error.textContent='Enter a ping server.';return;}}error.textContent='';draft.edit(value);queue.changed();}
Events.On('settings:menubar-result',({data})=>{if(data.client!==client)return;queue.finished();if(draft.result(data.revision,data.error))error.textContent=data.error||'';});
Events.On('settings:snapshot',({data})=>{if(!data.menubar||draft.dirty||form.contains(document.activeElement))return;for(const[id]of fields){const value=data.menubar[id];if(id==='ping_target'){const preset=['8.8.8.8','1.1.1.1','9.9.9.9'].includes(value);controls[id].value=preset?value:'custom';custom.value=preset?'':value;customRow.hidden=preset;}else controls[id].value=value;}});
