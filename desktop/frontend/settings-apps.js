import {Events} from '/wails/runtime.js';
import {PreferenceDraft,AutosaveQueue} from './settings-model.mjs';
const forms=new Map();
const client=crypto.randomUUID();
function options(select,items){select.replaceChildren(...items.map(item=>{const o=document.createElement('option');o.value=item.id;o.textContent=item.name;return o}));}
function row(parent,label,control){
 const wrapper=document.createElement('div');wrapper.className='form-row';
 const caption=document.createElement('label');caption.htmlFor=control.id;caption.textContent=label;
 wrapper.append(caption,control);parent.append(wrapper);return wrapper;
}
function input(form,key,label,placeholder=''){
 const el=document.createElement('input');el.type='text';el.id=form.kind+'-'+key;el.placeholder=placeholder;el.autocomplete='off';el.spellcheck=false;
 form.fields[key]=el;form.rows[key]=row(form.element,label,el);return el;
}
function read(form){
 const f=form.fields;
 if(form.kind!=='tracker')return {id:f.id.value,path:f.path.value,args:f.args.value};
 return {id:f.id.value,url:f.url.value,custom:{name:'Custom',ticketURL:f.ticketURL.value,rootURL:f.rootURL.value,urlRegExp:f.urlRegExp.value}};
}
function visible(form){
 const f=form.fields,custom=f.id.value==='custom';
 if(form.kind!=='tracker'){
  form.rows.path.hidden=form.rows.args.hidden=!custom;form.help.hidden=!custom;
 }else{
  const profile=form.catalog.find(p=>p.id===f.id.value);
  for(const key of ['ticketURL','rootURL','urlRegExp'])form.rows[key].hidden=!custom;
  const needsURL=custom?!!f.urlRegExp.value:!!profile?.urlRegExp;
  form.rows.url.hidden=!needsURL;
  f.url.placeholder=profile?.rootURL?.replaceAll('$ACCOUNT_SLUG','your-workspace')||'https://issues.example.com';
  form.help.textContent=custom?'Use $TICKET for the ticket ID. Optional $ACCOUNT_SLUG comes from URL pattern capture group 1.':'Paste a workspace, project, or ticket URL.';
 }
}
function save(form){
 const revision=form.draft.revision;
 Events.Emit('settings:application',{client,kind:form.kind,revision,selection:read(form)});
}
function edited(form){form.draft.edit(read(form));form.status.textContent='';form.queue.changed();}
function build(kind,catalog){
 const form={kind,catalog,draft:new PreferenceDraft(),element:document.getElementById(kind+'-form'),fields:{},rows:{}};
 form.queue=new AutosaveQueue(()=>save(form));
 const select=document.createElement('select');select.id=kind+'-id';options(select,catalog);form.fields.id=select;
 row(form.element,kind==='editor'?'Editor':kind==='git_client'?'Git client':'Ticket tracker',select);
 if(kind==='tracker'){
  input(form,'url','Workspace URL');input(form,'ticketURL','Ticket URL','https://issues.example.com/issue/$TICKET');
  input(form,'rootURL','Root URL','https://issues.example.com');input(form,'urlRegExp','URL pattern','^https://issues.example.com/([^/]+).*$');
 }else{
  const path=input(form,'path','Application','/Applications/Example.app');
  const wrapper=document.createElement('div');wrapper.className='path-control';path.replaceWith(wrapper);wrapper.append(path);
  const browse=document.createElement('button');browse.type='button';browse.textContent='Choose…';browse.onclick=()=>Events.Emit('settings:choose-application',kind);wrapper.append(browse);
  input(form,'args','Arguments','"$PATH"');
 }
 form.help=document.createElement('p');form.help.className='form-help';form.help.textContent='Use $PATH for the selected folder. Quote arguments containing spaces.';form.element.append(form.help);
 form.status=document.createElement('p');form.status.className='form-status';form.status.setAttribute('role','status');form.element.append(form.status);
 form.element.oninput=()=>{edited(form);visible(form)};
 select.onchange=()=>{
  if(kind!=='tracker'&&select.value==='custom'&&!form.fields.args.value)form.fields.args.value='"$PATH"';
  edited(form);visible(form);
 };
 form.element.onsubmit=event=>event.preventDefault();
 forms.set(kind,form);return form;
}
function write(form,value){
 form.fields.id.value=value.id;
 if(form.kind==='tracker'){
  form.fields.url.value=value.url||'';
  for(const key of ['ticketURL','rootURL','urlRegExp'])form.fields[key].value=value.custom?.[key]||'';
 }else{form.fields.path.value=value.path||'';form.fields.args.value=value.args||'';}
 visible(form);
}
export function updateApplications(data){
 if(!data)return;
 for(const [kind,catalog] of [['editor',data.editors],['git_client',data.gitClients],['tracker',data.trackers]]){
  const form=forms.get(kind)||build(kind,catalog);
  if(form.draft.sync(data.values[kind]))write(form,data.values[kind]);
 }
}
Events.On('settings:application-result',({data})=>{
 if(data.client!==client)return;
 const form=forms.get(data.kind);if(!form)return;
 form.queue.finished();
 if(!form.draft.result(data.revision,data.error))return;
 form.status.textContent=data.error||'';form.status.classList.toggle('error',!!data.error);
});
Events.On('settings:application-path',({data})=>{
 const form=forms.get(data.kind);if(!form)return;
 form.fields.path.value=data.path;edited(form);
});
