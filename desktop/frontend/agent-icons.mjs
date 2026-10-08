// Switch off to restore agent names without changing table data or actions.
export const USE_AGENT_ICONS = true;
const ns='http://www.w3.org/2000/svg';
export function agentIdentity(text){
 const name=text.replace(/!$/,'');
 const id=({Codex:'codex',Claude:'claude',Grok:'grok'})[name];
 return id?{id,name,conflict:text.endsWith('!')}:null;
}
// Full vector counterparts of the menu-bar knot, sunburst, and slashed circle.
export function agentIcon(document,identity){
 const svg=document.createElementNS(ns,'svg');
 svg.setAttribute('viewBox','0 0 24 24');svg.setAttribute('aria-hidden','true');svg.classList.add('agent-icon');
 const add=(tag,attrs)=>{const el=document.createElementNS(ns,tag);for(const [k,v] of Object.entries(attrs))el.setAttribute(k,v);svg.append(el);};
 if(identity.id==='claude'){
  for(let i=0;i<12;i++){const angle=i*Math.PI/6;const radius=i%2?9:10.5;add('path',{d:`M ${12+Math.cos(angle)*3} ${12+Math.sin(angle)*3} L ${12+Math.cos(angle)*radius} ${12+Math.sin(angle)*radius}`,stroke:'currentColor','stroke-width':i%2?1.8:2.3,'stroke-linecap':'square'});}
  add('circle',{cx:12,cy:12,r:3.2,fill:'currentColor'});
 }else if(identity.id==='codex'){
  for(let i=0;i<6;i++)add('path',{d:'M12 5 C8 1 3 5 4 9 L4 13 L12 17 L16 15 L16 7 Z',transform:`rotate(${i*60} 12 12)`,fill:'none',stroke:'currentColor','stroke-width':1.35,'stroke-linejoin':'round'});
 }else{
  add('path',{d:'M18 4 A9 9 0 1 0 20 17',fill:'none',stroke:'currentColor','stroke-width':2.1,'stroke-linecap':'round'});
  add('path',{d:'M4 22 L14 8 L24 1 L15 16 Z',fill:'currentColor'});
 }
 return svg;
}
