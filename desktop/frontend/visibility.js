import {Events} from '/wails/runtime.js';
let nativeVisible=false;
const key=location.pathname.includes('markdown')?'markdown':'dashboard';
export function viewVisible(){return nativeVisible&&!document.hidden;}
Events.On('energy:visibility',({data})=>{nativeVisible=!!data[key];window.dispatchEvent(new Event('viewvisibility'));});
document.addEventListener('visibilitychange',()=>window.dispatchEvent(new Event('viewvisibility')));

Events.Emit('energy:ready');
