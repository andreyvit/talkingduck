import {Events} from '/wails/runtime.js';

const get=id=>document.getElementById(id);
const port=get('viewport'), img=get('picture'), label=get('zoom');
let scale=1, fitting=true, drag=null;

function resize(next, x, y){
 if(!img.naturalWidth)return;
 const bounds=port.getBoundingClientRect();
 x??=bounds.left+bounds.width/2;
 y??=bounds.top+bounds.height/2;
 const before=img.getBoundingClientRect();
 const u=(x-before.left)/before.width, v=(y-before.top)/before.height;
 scale=next;
 img.style.width=img.naturalWidth*scale+'px';
 const after=img.getBoundingClientRect();
 port.scrollBy(after.left+u*after.width-x,after.top+v*after.height-y);
 label.value=Math.round(scale*100)+'%';
}
function fit(){
 fitting=true;
 if(img.naturalWidth)resize(Math.min(port.clientWidth/img.naturalWidth,port.clientHeight/img.naturalHeight,1));
}
function zoom(next,x,y){fitting=false;resize(Math.min(16,Math.max(.01,next)),x,y)}
window.clearImage=()=>{img.removeAttribute('src');label.value=''};
window.showImage=({src,alt})=>{fitting=true;img.alt=alt;img.src=src;if(img.complete)fit()};
img.onload=fit;
img.onerror=()=>{label.value='Image unavailable'};
new ResizeObserver(()=>{if(fitting&&port.clientWidth&&port.clientHeight)fit()}).observe(port);
get('fit').onclick=fit;
get('actual').onclick=()=>zoom(1);
get('in').onclick=()=>zoom(scale*1.25);
get('out').onclick=()=>zoom(scale/1.25);
get('close').onclick=()=>Events.Emit('image:close');
document.addEventListener('keydown',e=>{if(e.key==='Escape'){e.preventDefault();get('close').click()}});
port.addEventListener('wheel',e=>{e.preventDefault();zoom(scale*Math.exp(-Math.max(-200,Math.min(200,e.deltaY))*.007),e.clientX,e.clientY)},{passive:false});
port.onpointerdown=e=>{if(e.button!==0)return;port.setPointerCapture(e.pointerId);drag={x:e.clientX,y:e.clientY,left:port.scrollLeft,top:port.scrollTop};port.classList.add('dragging')};
port.onpointermove=e=>{if(drag)port.scrollTo(drag.left+drag.x-e.clientX,drag.top+drag.y-e.clientY)};
port.onlostpointercapture=()=>{drag=null;port.classList.remove('dragging')};
Events.Emit('image:ready');
