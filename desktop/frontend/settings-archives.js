import {Events} from '/wails/runtime.js';
import {archiveName,archiveAge} from './archive-model.mjs';
const list=document.getElementById('archives');
Events.On('settings:archives',({data})=>{
 const rows=(data||[]).map(row=>{
  const item=document.createElement('div');item.className='archive-row';
  const name=document.createElement('span');name.className='archive-name';name.textContent=archiveName(row);
  name.title=row.spec?`${row.folder}/${row.spec}`:row.folder;
  const age=document.createElement('span');age.className='archive-age';age.textContent=archiveAge(row.archivedAt);
  age.title=row.archivedAt?new Date(row.archivedAt).toLocaleString():'';
  const undo=document.createElement('button');undo.textContent='Unarchive';undo.setAttribute('aria-label',`Unarchive ${archiveName(row)}`);
  undo.onclick=()=>Events.Emit('settings:unarchive',{folder:row.folder,spec:row.spec});
  item.append(name,age,undo);return item;
 });
 if(!rows.length){const empty=document.createElement('p');empty.className='archive-empty';empty.textContent='No archived rows.';rows.push(empty)}
 list.replaceChildren(...rows);
});
