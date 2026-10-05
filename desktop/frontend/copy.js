import {Clipboard} from '/wails/runtime.js';

const icon='<svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.3" aria-hidden="true"><rect x="5.5" y="5.5" width="8" height="8" rx="1.5"/><path d="M10.5 3.5v-1a1 1 0 0 0-1-1h-7a1 1 0 0 0-1 1v7a1 1 0 0 0 1 1h1"/></svg>';
const style=document.createElement('style');
style.textContent=`.copyable{display:inline;white-space:inherit}.copy-button{display:inline-flex;align-items:center;justify-content:center;vertical-align:middle;width:18px;height:18px;padding:0;margin:0;border:0;background:none;color:inherit;visibility:hidden;opacity:.4}.copyable:hover>.copy-button,.copyable:focus-within>.copy-button{visibility:visible}.copy-button:hover,.copy-button:focus-visible{opacity:1;background:none}.copy-button:focus-visible{outline:1px solid currentColor;outline-offset:1px}`;
document.head.append(style);

export function copyable(content,text){
 const wrap=document.createElement('span');
 wrap.className='copyable';
 const button=document.createElement('button');
 button.type='button';button.className='copy-button';button.innerHTML=icon;
 button.title='Copy '+text;button.setAttribute('aria-label','Copy '+text);
 button.onclick=async event=>{
  event.preventDefault();event.stopPropagation();
  try{
   await Clipboard.SetText(text);
   button.textContent='✓';button.title='Copied';
   setTimeout(()=>{button.innerHTML=icon;button.title='Copy '+text},1200);
  }catch{
   button.title='Copy failed — try again';
   document.querySelector('#error').textContent='Could not copy to clipboard. Try again.';
  }
 };
 wrap.append(content,button);
 return wrap;
}
