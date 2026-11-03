import test from 'node:test';
import assert from 'node:assert/strict';
import {LeapSearch,shiftTransition} from '../frontend/markdown-leap.mjs';
test('edits always search from top, arrows cycle only matching headings',()=>{
 const s=new LeapSearch(['Scope','Code overview','Code changes','Other'],{main:100,toc:50});
 assert.equal(s.edit('CO').index,0);
 assert.equal(s.move(1).index,1);
 assert.equal(s.move(-1).index,0);
 assert.equal(s.move(-1).index,2);
 assert.equal(s.edit('CODE').index,1);
 assert.deepEqual(s.match,{index:1,start:0,length:4});
 assert.equal(s.edit('missing'),null);
 assert.deepEqual(s.origin,{main:100,toc:50});
 assert.equal(s.edit(''),null);
 assert.equal(s.edit('code').index,1);
});

test('search ignores spaces but retains original highlight offsets and typed text',()=>{
 const s=new LeapSearch(['Other','UI Changes','UI Check'],{main:123,toc:45});
 assert.deepEqual(s.edit('UICH'),{index:1,start:0,length:5});
 assert.equal(s.move(1).index,2);
 assert.deepEqual(s.edit('UI CH'),{index:1,start:0,length:5});
 assert.equal(s.query,'UI CH');
 assert.equal(s.edit('UI ').index,1);assert.equal(s.query,'UI ');
 assert.equal(s.edit('missing'),null);
 assert.equal(s.edit(''),null);
 assert.deepEqual(s.origin,{main:123,toc:45});
});
test('space-insensitive matches preserve offsets after non-ASCII characters',()=>{
 const s=new LeapSearch(['🙂 UI\tChanges'],{});
 assert.deepEqual(s.edit('uich'),{index:0,start:3,length:5});
});

test('repeated second-Shift taps advance once each, then either release order exits',()=>{
 for(const first of [1,2]){
  for(const last of [1,2]){
   const masks=[first,3,3,first,3,first,3,last,0];
   let previous=0;
   const actions=masks.map(mask=>{
    const action=shiftTransition(previous,mask);previous=mask;return action;
   });
   assert.deepEqual(actions,['hold','next','hold','hold','next','hold','next','hold','finish']);
  }
 }
});
