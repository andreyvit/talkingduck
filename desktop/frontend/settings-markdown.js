import {Events} from '/wails/runtime.js';
import {PreferenceDraft,AutosaveQueue} from './settings-model.mjs';
const font=document.getElementById('markdown-font'),size=document.getElementById('markdown-size'),error=document.getElementById('markdown-appearance-error');
const draft=new PreferenceDraft(),client=crypto.randomUUID();let fontsSet=false;
const queue=new AutosaveQueue(()=>Events.Emit('settings:application',{client,kind:'markdown_appearance',revision:draft.revision,selection:draft.value}));
function edit(){draft.edit({font:font.value,size:Number(size.value)});error.textContent='';queue.changed()}
document.getElementById("markdown-appearance-form").onsubmit=e=>e.preventDefault();
font.onchange=edit;size.oninput=edit;
export function updateMarkdownAppearance(data){
 if(!data)return;
 if(!fontsSet){fontsSet=true;for(const name of data.fonts||[]){const o=document.createElement('option');o.value=name;o.textContent=name;font.append(o)}}
 if(draft.sync(data.appearance)){if(![...font.options].some(o=>o.value===data.appearance.font)){const o=document.createElement('option');o.value=data.appearance.font;o.textContent=data.appearance.font;font.append(o)}font.value=data.appearance.font;size.value=data.appearance.size}
}
Events.On('settings:application-result',({data})=>{if(data.client!==client||data.kind!=='markdown_appearance')return;queue.finished();if(draft.result(data.revision,data.error))error.textContent=data.error||''});
