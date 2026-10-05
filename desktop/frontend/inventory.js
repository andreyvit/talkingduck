import {Events,Clipboard} from '/wails/runtime.js';
import {copyable} from './copy.js';
import {InventoryController} from './controller.mjs';

const $=id=>document.getElementById(id);
let model={columns:[],rows:[]},lastPointer=null,noticeTimer;
const controller=new InventoryController(async action=>{
 try {
  switch(action.kind) {
   case 'open': Events.Emit('inventory:open-'+action.value.kind,action.value.target);break;
   case 'copy': await Clipboard.SetText(action.value);notice('Copied: '+action.label);break;
   case 'refresh': Events.Emit(action.value?'inventory:refresh-folder':'inventory:refresh',action.value);notice('Refresh queued');break;
   case 'settings': Events.Emit('settings:open');break;
  }
 } catch(error) {$('error').textContent=String(error);}
});
function notice(text) {
 clearTimeout(noticeTimer);$('notice').textContent=text;
 noticeTimer=setTimeout(()=>{$('notice').textContent='';},4000);
}
const hints=[['ArrowUp','↑','Up'],['ArrowDown','↓','Down'],['r','r','Refresh'],['R','R','All'],
 ['a','a','Agent'],['t','t','Ticket'],['s','s','Spec'],['f','f','Folder'],['c','c','Copy'],['settings','^S','Settings']];
const copyHints=[['t','ct','Ticket'],['T','cT','Ticket URL'],['s','cs','Spec'],['f','cf','Folder path'],['Escape','Esc','Cancel']];
function invoke(key) {
 const handled=controller.action(key);
 if(!handled)return false;
 paintState();
 if(controller.mode==='keyboard' && ['ArrowUp','ArrowDown'].includes(key))
  $('rows').children[controller.selected]?.scrollIntoView({block:'nearest'});
 return handled;
}
function paintState() {
 for(const [index,tr] of [...$('rows').children].entries()) {
  const keyboard=controller.mode==='keyboard'&&controller.selected===index;
  tr.classList.toggle('keyboard-selected',keyboard);
  tr.classList.toggle('mouse-selected',controller.mode==='mouse'&&controller.hovered===tr.dataset.rowId);
  tr.setAttribute('aria-selected',String(keyboard));
 }
 $('keys').replaceChildren(...(controller.chord?copyHints:hints).map(([key,label,text])=>{
  const button=document.createElement('button');button.className='key-hint';button.type='button';
  const k=document.createElement('kbd');k.textContent=label;button.append(k,' '+text);
  button.onclick=()=>invoke(key);
  return button;
 }));
 renderChooser();
}
function renderChooser() {
 const dialog=$('chooser'),wasOpen=dialog.open;
 if(!controller.choices.length) {if(wasOpen)dialog.close();return;}
 $('choices').replaceChildren(...controller.choices.map((action,index)=>{
  const button=document.createElement('button');button.textContent=action.label;
  button.classList.toggle('chosen',index===controller.choice);
  button.onclick=()=>{controller.choose(index);paintState();};
  return button;
 }));
 if(!wasOpen)dialog.showModal();
 $('choices').children[controller.choice]?.focus({preventScroll:true});
}
$('chooser').addEventListener('cancel',event=>{event.preventDefault();invoke('Escape');});
$('cancel-choice').onclick=()=>invoke('Escape');
document.addEventListener('keydown',event=>{
 if(event.isComposing || event.target.matches('input,textarea,[contenteditable=true]'))return;
 if(event.ctrlKey&&event.key.toLowerCase()==='s') {event.preventDefault();invoke('settings');return;}
 if(event.ctrlKey||event.metaKey||event.altKey)return;
 if(invoke(event.key))event.preventDefault();
});
function mouseRow(event,id) {
 // Layout updates and keyboard scrolling must not switch modes under a still pointer.
 const point=[event.clientX,event.clientY];
 if(event.type==='pointermove'&&lastPointer&&point[0]===lastPointer[0]&&point[1]===lastPointer[1])return;
 lastPointer=point;
 if(controller.mode==='mouse'&&controller.hovered===id)return;
 controller.hover(id);paintState();
}
function render(data) {
 model={columns:data.table?.columns||[],rows:data.table?.rows||[]};controller.replace(model.rows);
 $('headers').replaceChildren(...model.columns.map(col=>{
  const th=document.createElement('th');th.textContent=col.label;th.style.textAlign=col.align||'left';
  th.dataset.column=col.id;return th;
 }));
 $('status').textContent=data.loading?'Refreshing '+(data.refreshingPath||'inventory')+'…':'';
 $('error').textContent=data.error||'';
 $('warnings').textContent=(data.warnings||[]).join('\n');
 $('empty').hidden=model.rows.length>0;
 $('empty').textContent=data.loading?'Collecting worktrees and agent sessions…':data.error?'Refresh failed. Try again.':'No worktrees to display.';
 $('rows').replaceChildren(...model.rows.map(row=>{
  const tr=document.createElement('tr');tr.className=row.state||'';tr.dataset.rowId=row.id;
  tr.addEventListener('pointermove',event=>mouseRow(event,row.id));
  tr.addEventListener('pointerdown',event=>mouseRow(event,row.id));
  row.cells.forEach((cell,column)=>{
   const td=document.createElement('td');td.style.textAlign=model.columns[column].align||'left';
   td.dataset.column=model.columns[column].id;
   cell.lines.forEach(line=>{
    const lineElement=document.createElement('div');lineElement.className='cell-line';
    const items=line.items.filter(item=>!item.hidden);
    if(items.length===1)lineElement.classList.add('single-item');
    items.forEach(item=>{
     const el=document.createElement(item.link?'a':'span');el.textContent=item.text;el.className=item.color||'';
     if(item.bold)el.style.fontWeight='550';el.title=item.tooltip||'';
     if(item.link) {
      el.classList.add('inventory-link');el.href='#';
      el.onclick=event=>{event.preventDefault();controller.perform({kind:'open',value:item.link});};
     }
     lineElement.append(item.copy!==undefined?copyable(el,item.copy):el);
    });
    td.append(lineElement);
   });
   tr.append(td);
  });return tr;
 }));
 paintState();
}
Events.On('inventory:snapshot',event=>{
 try {render(event.data);} catch(error) {$('error').textContent='Could not render inventory: '+error.message;}
});
Events.On('inventory:open-error',event=>{$('error').textContent=event.data;});
Events.On('mock:error',event=>{$('error').textContent=event.data;});
paintState();
Events.Emit('inventory:ready');
