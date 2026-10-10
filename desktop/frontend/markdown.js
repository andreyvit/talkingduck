import {LeapSearch} from './markdown-leap.mjs';
import {Events} from '/wails/runtime.js';
const elements=new Map([...document.querySelectorAll("[id]")].map(e=>[e.id,e]));
const $=id=>elements.get(id),scroller=$("scroller"),article=$("document");
const headingTopGap=28;
let state=null,headings=[],renderToken=0,positionTimer=null,restoreTarget=null;
function currentHeading(){let current=-1;for(let i=0;i<headings.length;i++){if(headings[i].getBoundingClientRect().top-scroller.getBoundingClientRect().top<=headingTopGap+1)current=i;else break}return current}
function position(){const i=currentHeading();return {top:scroller.scrollTop,heading:i<0?'':headings[i].id,offset:i<0?0:scroller.scrollTop-headings[i].offsetTop}}
function send(action,extra={}){Events.Emit('markdown:command',{action,path:state?.path||'',position:position(),heading:currentHeading(),...extra})}
function savePosition(){if(state&&!leap)send('position')}
function highlight(reveal=false){const i=leap?.match?.index??currentHeading();for(const [index,a] of [...$('toc').querySelectorAll('a')].entries()){a.setAttribute('aria-current',String(index===i));if(reveal&&index===i)a.scrollIntoView({block:'nearest'})}}
function restore(p){const h=p.heading&&headings.find(h=>h.id===p.heading);scroller.scrollTop=h?h.offsetTop+p.offset:p.top;highlight(true)}
function jump(id){restoreTarget=null;const h=headings.find(h=>h.id===id);if(h){scroller.scrollTop=h.offsetTop-headingTopGap;highlight(true);savePosition()}}
function cancelRestore(){restoreTarget=null}
scroller.addEventListener('wheel',cancelRestore,{passive:true});scroller.addEventListener('touchstart',cancelRestore,{passive:true});scroller.addEventListener('pointerdown',cancelRestore);
scroller.addEventListener('scroll',()=>{highlight();clearTimeout(positionTimer);positionTimer=setTimeout(savePosition,300)},{passive:true});
window.addEventListener('blur',()=>{finishLeap();savePosition()});
let leap=null;
const heldShifts=new Set();
const leapBar=document.createElement('nav'),leapQuery=document.createElement('span');
leapBar.className='leap-bar';leapBar.hidden=true;leapBar.setAttribute('aria-label','Search actions');
for(const [key,label,action] of [['↑↓','Matches',()=>{leap.move(1);applyLeap()}],['Esc','Reset',()=>{leap.edit('');applyLeap()}],['⌫','Delete',()=>{leap.edit([...leap.query].slice(0,-1).join(''));applyLeap()}]]){
 const button=document.createElement('button'),kbd=document.createElement('kbd');button.className='key-hint';kbd.textContent=key;button.append(kbd,' '+label);button.onclick=()=>{if(leap)action()};leapBar.append(button);
}
leapQuery.className='leap-query';leapBar.prepend(leapQuery);$('keys').after(leapBar);
function renderLeapBar(){
 $('keys').hidden=!!leap;leapBar.hidden=!leap;
 leapQuery.textContent='Type to search: '+(leap?.query||'');
 leapQuery.classList.toggle('no-match',!!leap?.query&&!leap.match);
}
function beginLeap(){
 if(leap)return;
 cancelRestore();clearTimeout(positionTimer);
 leap=new LeapSearch(headings.map(h=>h.textContent),{main:scroller.scrollTop,toc:$('toc').scrollTop});
 renderLeapBar();
}
function paintLeap(){
 for(const [i,a] of [...$('toc').querySelectorAll('a')].entries()){
  const title=headings[i].textContent,m=leap?.matches.find(m=>m.index===i);
  a.replaceChildren();
  if(m){const mark=document.createElement('mark');mark.textContent=title.slice(m.start,m.start+m.length);a.append(title.slice(0,m.start),mark,title.slice(m.start+m.length))}
  else a.textContent=title;
 }
}
function applyLeap(){
 renderLeapBar();paintLeap();
 if(leap.match)jump(headings[leap.match.index].id);
 else {scroller.scrollTop=leap.origin.main;highlight();$('toc').scrollTop=leap.origin.toc}
}
function finishLeap(){
 if(!leap)return;
 if(!leap.match){scroller.scrollTop=leap.origin.main;$('toc').scrollTop=leap.origin.toc}
 leap=null;heldShifts.clear();renderLeapBar();paintLeap();highlight();savePosition();
}
function handleLeap(e){
 if(e.metaKey||e.ctrlKey||e.altKey||$('history').open||e.target.closest('input,textarea,select,[contenteditable=true]'))return false;
 if(e.key==='Shift'){heldShifts.add(e.code);beginLeap();return true}
 if(!e.shiftKey){finishLeap();return false;}
 if(e.key==='Escape'){beginLeap();e.preventDefault();leap.edit('');applyLeap();return true}
 if(e.key==='ArrowUp'||e.key==='ArrowDown'){
  beginLeap();e.preventDefault();leap.move(e.key==='ArrowUp'?-1:1);applyLeap();return true;
 }
 if(e.key==='Backspace'||(e.key.length===1&&!e.isComposing)){
  beginLeap();e.preventDefault();leap.edit(e.key==='Backspace'?[...leap.query].slice(0,-1).join(''):leap.query+e.key);applyLeap();return true;
 }
 return false;
}
document.addEventListener('keyup',e=>{
 if(e.key!=='Shift')return;
 heldShifts.delete(e.code);
 if(!e.shiftKey||!heldShifts.size)finishLeap();
});
const hints=[['Escape','Esc','Close'],[' ','Space','Next page'],['ArrowDown','↑↓','ToC'],['ArrowLeft','←','Back'],['ArrowRight','→','Forward'],['r','r','Refresh'],['c','c','ToC'],['h','h','History'],['Shift','Shift','Hold to search']];
for(const [key,label,title] of hints){const button=document.createElement('button');button.className='key-hint';button.dataset.key=key;const k=document.createElement('kbd');k.textContent=label;button.append(k,' '+title);button.onclick=e=>{if(key==='Shift')return;cancelRestore();send('',{key,shift:e.shiftKey})};$('keys').append(button)}
$('keys').querySelector('[data-key=" "]').title='Space: next page';
document.addEventListener('keydown',e=>{if(handleLeap(e))return;if(e.metaKey||e.ctrlKey||e.altKey||e.target.closest('input,textarea,select,[contenteditable=true]'))return;if(!['Escape',' ','ArrowUp','ArrowDown','ArrowLeft','ArrowRight','r','R','c','C','h','H'].includes(e.key))return;e.preventDefault();cancelRestore();send('',{key:e.key,shift:e.shiftKey})});
article.addEventListener('click',e=>{const a=e.target.closest('a');if(!a)return;e.preventDefault();const href=a.getAttribute('href')||'';if(href.startsWith('#')){try{jump(decodeURIComponent(href.slice(1)))}catch{}}else send('link',{link:href})});
Events.On('markdown:snapshot',({data})=>{
 if(state&&data.generation<state.generation)return;
 finishLeap();
 const same=state?.generation>0&&state?.path===data.path,keep=same?position():data.position;
 const changed=!same||state?.document.html!==data.document.html;
 clearTimeout(positionTimer);
 state=data;$('error').textContent=[data.error,...(data.document.warnings||[])].filter(Boolean).join('\n');
 $('toc').hidden=data.hideTOC;
 article.style.setProperty('--viewer-font',data.appearance.font==='github'?'var(--github-font)':JSON.stringify(data.appearance.font)+', var(--github-font)');article.style.setProperty('--viewer-size',data.appearance.size+'px');
 $('keys').querySelector('[data-key="ArrowLeft"]').disabled=!data.back;$('keys').querySelector('[data-key="ArrowRight"]').disabled=!data.forward;
 if(changed){article.innerHTML=data.document.html||'';headings=[...article.querySelectorAll('h1,h2,h3,h4,h5,h6')];$('toc').replaceChildren();for(const h of data.document.headings||[]){const a=document.createElement('a');a.href='#'+h.id;a.textContent=h.title;a.style.paddingLeft=(8+(h.level-1)*12)+'px';a.onclick=e=>{e.preventDefault();jump(h.id)};$('toc').append(a)}if(!headings.length){const empty=document.createElement('span');empty.className='empty';empty.textContent='No headings';$('toc').append(empty)}}
 const token=++renderToken;restoreTarget=keep||{top:0};
 const apply=()=>{if(token===renderToken&&restoreTarget)restore(restoreTarget)};
 requestAnimationFrame(apply);
 // Images and font swaps can change heading offsets after the first layout.
 for(const img of article.querySelectorAll('img'))if(!img.complete){img.addEventListener('load',apply,{once:true});img.addEventListener('error',apply,{once:true})}
 document.fonts.ready.then(apply);
});
Events.On('markdown:effect',({data})=>{cancelRestore();switch(data.action){case 'heading':jump(data.id);break;case 'toc':{const p=position();$('toc').hidden=data.hidden;restore(p);break}case 'page-up':scroller.scrollTop-=scroller.clientHeight*.9;break;case 'page-down':scroller.scrollTop+=scroller.clientHeight*.9;break}});
Events.On('markdown:error',({data})=>{$('error').textContent=data});
Events.Emit('markdown:ready');
let recent=[],filtered=[],historyIndex=0;
function renderHistory(){const query=$('history-filter').value.toLowerCase();filtered=recent.filter(v=>v.path.toLowerCase().includes(query));historyIndex=Math.min(historyIndex,Math.max(0,filtered.length-1));$('history-list').replaceChildren();filtered.forEach((visit,i)=>{const b=document.createElement('button');b.setAttribute('aria-selected',String(i===historyIndex));b.textContent=visit.path.split('/').at(-1);const small=document.createElement('small');small.textContent=visit.path;b.append(small);b.onclick=()=>openRecent(visit.path);$('history-list').append(b)});$('history-empty').hidden=filtered.length>0;$('history-empty').textContent=recent.length?'No matching files.':'No recent Markdown files. Open a spec from the dashboard with S.'}
function openRecent(path){$('history').close();send('recent',{link:path})}
$('history-filter').oninput=()=>{historyIndex=0;renderHistory()};$('history-close').onclick=()=>$('history').close();
$('history').addEventListener('keydown',e=>{e.stopPropagation();if(e.key==='ArrowDown'||e.key==='ArrowUp'){e.preventDefault();historyIndex=Math.max(0,Math.min(filtered.length-1,historyIndex+(e.key==='ArrowDown'?1:-1)));renderHistory();$('history-list').children[historyIndex]?.scrollIntoView({block:'nearest'})}else if(e.key==='Enter'&&filtered[historyIndex]){e.preventDefault();openRecent(filtered[historyIndex].path)}});
Events.On('markdown:history',({data})=>{recent=data||[];historyIndex=0;$('history-filter').value='';renderHistory();if(!$('history').open)$('history').showModal();$('history-filter').focus()});
