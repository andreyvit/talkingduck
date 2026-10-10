import {Events} from '/wails/runtime.js';
import {shortcutOptions,shortcutLabel} from './settings-model.mjs';
import {updateApplications} from './settings-apps.js';
import './settings-archives.js';
import './settings-hotkeys.js';
import {copyable} from './copy.js';
const $=id=>document.getElementById(id);
let folderSnapshot=[],folderSignature='',addressValue=null;
function select(values,value,label,onchange){
 const el=document.createElement('select');el.setAttribute('aria-label',label);
 for(const [v,text] of values){const option=document.createElement('option');option.value=v;option.textContent=text;el.append(option)}
 el.value=value;el.onchange=()=>{const value=el.value;$('error').textContent='';onchange(value)};return el;
}
Events.On('settings:snapshot',({data})=>{
 updateApplications(data.applications);
 if(addressValue!==data.address){
  addressValue=data.address;
  const field=document.createElement('input');field.id='address';field.readOnly=true;field.value=data.address;field.placeholder='Starting server…';field.onclick=()=>field.select();
  const content=copyable(field,data.address);content.classList.add('address-copy');$('address-control').replaceChildren(content);
 }
 $('mcp-error').textContent=data.error;
 $('path').textContent=data.path;
 if(data.settingsError)$('error').textContent=data.settingsError;
 folderSnapshot=data.folders||[];renderFolders();
});
function renderFolders(){
 const signature=JSON.stringify(folderSnapshot);
 if(signature===folderSignature||$('folders').contains(document.activeElement))return;
 folderSignature=signature;
 const rows=[];
 for(const folder of folderSnapshot){
  const row=document.createElement('tr'),path=document.createElement('td'),visibility=document.createElement('td'),shortcut=document.createElement('td');
  path.textContent=folder.path.split('/').filter(Boolean).at(-1)||folder.path;
  const detail=document.createElement('small');detail.textContent=folder.path;path.append(detail);
  const status=document.createElement('small');status.textContent=[folder.status==='missing'?'Missing':'',folder.error||''].filter(Boolean).join(' · ');path.append(status);
  visibility.className='use-cell';
  const use=document.createElement('input');use.type='checkbox';use.checked=folder.visible;use.setAttribute('aria-label','Use '+folder.path);
  use.onchange=()=>{$('error').textContent='';Events.Emit('settings:visibility',{path:folder.path,visible:use.checked})};visibility.append(use);
  shortcut.append(select(shortcutOptions.map(value=>[value,shortcutLabel(value)]),folder.menubar_item_shortcut,'Menu bar shortcut '+folder.path,value=>Events.Emit('settings:shortcut',{path:folder.path,shortcut:value})));
  row.append(visibility,path,shortcut);rows.push(row);
 }
 $('folders').replaceChildren(...rows);
 if(!rows.length){const row=document.createElement('tr'),cell=document.createElement('td');cell.colSpan=3;cell.textContent='No folders yet. Add one to start monitoring.';row.append(cell);$('folders').append(row)}
}
$('folders').addEventListener('focusout',()=>setTimeout(renderFolders,0));
Events.On('settings:error',({data})=>{$('error').textContent=data;Events.Emit('settings:ready')});
$('reveal').onclick=()=>Events.Emit('settings:reveal-file');
$('add').onclick=()=>Events.Emit('settings:add-folder');
Events.Emit('settings:ready');
