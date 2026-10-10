// UI state and actions are independent of DOM rendering and the native bridge.
export class InventoryController {
 constructor(run) {
  this.run=run;this.rows=[];this.selected=0;this.hovered=null;this.mode='mouse';
  this.chord='';this.chordRow=null;this.choices=[];this.choice=0;this.choiceAction=null;
 }
 get effectiveRow() {
  return this.mode==='keyboard'?this.rows[this.selected]:this.rows.find(row=>row.id===this.hovered);
 }
 replace(rows) {
  rows=rows||[];
  const id=this.rows[this.selected]?.id;
  this.rows=rows;
  const index=rows.findIndex(row=>row.id===id);
  this.selected=index>=0?index:Math.max(0,Math.min(this.selected,rows.length-1));
  if(!rows.some(row=>row.id===this.hovered))this.hovered=null;
  if(this.chord && !rows.some(row=>row.id===this.chordRow))this.cancel();
  if(this.choices.length) {
   const row=rows.find(row=>row.id===this.choiceAction.row);
   if(JSON.stringify(this.matches(row,this.choiceAction.key))!==JSON.stringify(this.choices))this.cancel();
  }
 }
 cancel() {this.chord='';this.chordRow=null;this.choices=[];this.choiceAction=null;}
 hover(id,{active=true,clicked=false}={}) {
  if(this.choices.length)return;
  if(!clicked&&(!active||(this.mode==='keyboard'&&this.rows[this.selected]?.id===id))) {
   this.hovered=id;return;
  }
  if(this.mode!=='mouse'||this.hovered!==id)this.cancel();
  this.mode='mouse';this.hovered=id;
 }
 activate(hoveredID=null) {
  if(hoveredID!==null)this.hover(hoveredID);
 }
 matches(row,key) {
  if(!row)return [];
  const copying=key.startsWith('c'),suffix=copying?key.slice(1):key;
  const pathCopy=copying&&suffix==='S';
  const chord=key==='S'||pathCopy?'s':suffix==='T'&&copying?'t':suffix;
  return [...(row.actions||[]),...row.cells.flatMap(cell=>cell.lines.flatMap(line=>line.items))].filter(item=>
   item.chord===chord && (copying?(suffix==='T'?item.link?.kind==='ticket':pathCopy?item.link?.kind==='spec':item.copy!==undefined):!!item.link)
  ).map(item=>({label:item.text,kind:copying?(suffix==='T'?'copy-ticket-url':'copy'):'open',value:copying?
   (suffix==='T'||pathCopy?item.link.target:item.copy):key==='S'?{...item.link,kind:'reveal-spec'}:item.link}));
 }
 selectRow(id) {
  const index=this.rows.findIndex(row=>row.id===id);
  if(index<0)return;
  this.selected=index;this.mode='keyboard';
 }
 perform(action,rowID) {
  if(rowID!==undefined)this.selectRow(rowID);
  this.run(action);
 }
 choose(index) {
  const action=this.choices[index];
  if(!action)return;
  const rowID=this.choiceAction.row;
  this.cancel();this.perform(action,rowID);
 }
 action(key) {
  const row=this.effectiveRow;
  if(key==='Escape') {this.cancel();this.mode='mouse';return true;}
  if(key==='settings') {this.perform({kind:'settings'});return true;}
  if(this.choices.length) {
   if(key==='ArrowUp')this.choice=Math.max(0,this.choice-1);
   else if(key==='ArrowDown')this.choice=Math.min(this.choices.length-1,this.choice+1);
   else if(key==='Enter')this.choose(this.choice);
   else return false;
   return true;
  }
  if(key==='ArrowUp'||key==='ArrowDown') {
   this.cancel();this.mode='keyboard';
   this.selected=Math.max(0,Math.min(this.rows.length-1,this.selected+(key==='ArrowDown'?1:-1)));
   return true;
  }
  if(!this.chord&&(key==='c'||key==='C'||key==='A')) {
   if(!row)return true;
   this.chord=key==='A'?'A':'c';this.chordRow=row.id;return true;
  }
  let target=row;
  if(this.chord) {
   target=this.rows.find(row=>row.id===this.chordRow);
   key=this.chord+(this.chord==='A'?key.toLowerCase():key);this.chord='';this.chordRow=null;
  }
  if(key==='R') {this.perform({kind:'refresh',value:''});return true;}
  if(key==='X') {
   if(target?.archive)this.perform({kind:'archive',value:{folder:target.group,spec:target.archive.target}},target.id);
   return true;
  }
  if(key==='r') {if(target)this.perform({kind:'refresh',value:target.group},target.id);return true;}
  if(!['a','t','s','S','f','e','g','ct','cT','cs','cS','cf','Ax','Ac','Ag'].includes(key))return false;
  const matches=this.matches(target,key);
  if(matches.length===1)this.perform(matches[0],target.id);
  else if(matches.length>1) {
   this.choices=matches;this.choice=0;this.choiceAction={row:target.id,key};
  }
  if(!matches.length&&key.startsWith('A'))this.run({kind:'notice',value:'Action unavailable for this row'});
  return true;
 }
}
