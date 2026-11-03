import {Events} from '/wails/runtime.js';

// Draw the hover cue in the page. WKWebView can restore its arrow after a click.
const lens=document.createElement('span');
lens.className='image-hover-lens';
lens.hidden=true;
lens.setAttribute('aria-hidden','true');
document.body.append(lens);
function hover(e){
 const img=e.target.closest('.image-frame img,.mermaid-diagram img');
 if(!img || e.pointerType==='touch'){lens.hidden=true;return}
 lens.textContent=img.parentElement.classList.contains('expanded')?'−':'+';
 lens.style.transform='translate('+e.clientX+'px,'+e.clientY+'px)';
 lens.hidden=false;
}
export function installImageControls(article){
 document.addEventListener('pointermove',hover);
 article.addEventListener('pointerover',hover);
 article.addEventListener('pointerout',()=>{lens.hidden=true});
 window.addEventListener('blur',()=>{lens.hidden=true});
 article.addEventListener('contextmenu',()=>{lens.hidden=true});
 article.addEventListener('click',e=>{
  const img=e.target.closest('img');
  if(!img)return;
  if(e.shiftKey){
   e.preventDefault();e.stopImmediatePropagation();
   Events.Emit('image:open',img.dataset.previewKey);
  }else if(img.parentElement.matches('.image-frame,.mermaid-diagram')){
   e.preventDefault();e.stopImmediatePropagation();
   img.parentElement.classList.toggle('expanded');
   hover(e);
  }
 });
}
export function prepareImages(article,generation){
 lens.hidden=true;
 for(const [index,img] of [...article.querySelectorAll('img')].entries()){
  const key=generation+'/'+index;
  img.dataset.previewKey=key;
  img.style.setProperty('--custom-contextmenu-data',key);
  if(img.closest('a,.image-frame,.mermaid-diagram'))continue;
  const frame=document.createElement('span');
  frame.className='image-frame';
  img.replaceWith(frame);frame.append(img);
 }
}
