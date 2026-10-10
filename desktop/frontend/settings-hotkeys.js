import {Events} from '/wails/runtime.js';
import {recordedHotkey,hotkeyLabel} from './hotkey-model.mjs';
let keys=['Ctrl+T','',''],recording=-1,ready=false,pending=false;
const error=document.getElementById('hotkey-error');
const controls=[0,1,2].map(index=>{
 const button=document.getElementById('hotkey-'+index),clear=document.getElementById('hotkey-clear-'+index);
 button.onclick=()=>{
  if(pending)return;
  recording=index;ready=false;error.textContent='';render();
  Events.Emit('settings:hotkey',{kind:'record'});
 };
 clear.onclick=()=>{const next=[...keys];next[index]='';save(next)};
 return {button,clear};
});
function render(){
 controls.forEach(({button,clear},index)=>{
  button.textContent=recording===index?'Press shortcut…':hotkeyLabel(keys[index]);
  button.classList.toggle('recording',recording===index);
  button.setAttribute('aria-label',(['Global hotkey: ','Additional hotkey: ','Markdown preview: '][index])+button.textContent);
  button.setAttribute('aria-pressed',String(recording===index));
  button.disabled=pending;clear.disabled=pending;clear.hidden=!keys[index];
 });
}
function cancel(){
 if(recording<0)return;
 recording=-1;ready=false;render();Events.Emit('settings:hotkey',{kind:'cancel'});
}
function save(next){
 recording=-1;ready=false;pending=true;error.textContent='';render();
 Events.Emit('settings:hotkey',{kind:'save',keys:next});
}
Events.On('settings:hotkeys',({data})=>{
 if(data.recording){if(recording>=0)ready=true;return}
 keys=data.keys;pending=false;
 if(data.error){error.textContent=data.error;recording=-1}
 render();
});
document.addEventListener('keydown',event=>{
 if(recording<0)return;
 event.preventDefault();event.stopImmediatePropagation();
 if(event.code==='Escape'&&!event.ctrlKey&&!event.altKey&&!event.shiftKey&&!event.metaKey){cancel();return}
 if(!ready)return;
 try {const value=recordedHotkey(event);if(value){const next=[...keys];next[recording]=value;save(next)}}
 catch(e){error.textContent=e.message}
},true);
document.addEventListener('pointerdown',event=>{
 if(recording>=0&&!controls[recording].button.contains(event.target))cancel();
},true);
window.addEventListener('blur',cancel);
render();
