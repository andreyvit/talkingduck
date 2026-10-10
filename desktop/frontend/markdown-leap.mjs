// A held-Shift search starts at the top on each edit, regardless of selection.
export class LeapSearch {
 constructor(titles, origin) { this.titles=titles; this.origin=origin; this.query=''; this.matches=[]; this.cursor=0; }
 edit(query) {
  this.query=query; this.cursor=0;
  this.matches=query ? this.titles.flatMap((title,index)=>{
   const start=title.toLocaleLowerCase().indexOf(query.toLocaleLowerCase());
   return start<0 ? [] : [{index,start,length:query.length}];
  }) : [];
  return this.match;
 }
 move(delta) { if(this.matches.length)this.cursor=(this.cursor+delta+this.matches.length)%this.matches.length; return this.match; }
 get match() { return this.matches[this.cursor] || null; }
}
