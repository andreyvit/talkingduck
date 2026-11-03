// A held-Shift search starts at the top on each edit, regardless of selection.
export class LeapSearch {
 constructor(titles, origin) { this.titles=titles; this.origin=origin; this.query=''; this.matches=[]; this.cursor=0; }
 edit(query) {
  this.query=query; this.cursor=0;
  const needle=searchText(query).text;
  this.matches=needle ? this.titles.flatMap((title,index)=>{
   const haystack=searchText(title),at=haystack.text.indexOf(needle);
   if(at<0)return [];
   const start=haystack.starts[at],end=haystack.ends[at+needle.length-1];
   return [{index,start,length:end-start}];
  }) : [];
  return this.match;
 }
 move(delta) { if(this.matches.length)this.cursor=(this.cursor+delta+this.matches.length)%this.matches.length; return this.match; }
 get match() { return this.matches[this.cursor] || null; }
}

// AppKit reports both physical Shift bits on every flagsChanged event.
export function shiftTransition(previous,current){
 if(current===0)return 'finish';
 if(current===3 && previous!==3)return 'next';
 return 'hold';
}

// Keep original UTF-16 offsets for highlighting after whitespace removal/case folding.
function searchText(value) {
 let text='',starts=[],ends=[],offset=0;
 for(const char of value){
  const end=offset+char.length;
  if(!/\s/u.test(char)){
   const folded=char.toLocaleLowerCase();text+=folded;
   for(let i=0;i<folded.length;i++){starts.push(offset);ends.push(end)}
  }
  offset=end;
 }
 return {text,starts,ends};
}
