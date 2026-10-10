import {test} from 'node:test';
import assert from 'node:assert/strict';
import {AutosaveQueue} from '../frontend/settings-model.mjs';
function clock(){
 let now=0,serial=0;const tasks=new Map();
 return {now:()=>now,setTimer:(fn,delay)=>{tasks.set(++serial,{at:now+delay,fn});return serial},clearTimer:id=>tasks.delete(id),
  advance(to){for(;;){const next=[...tasks].sort((a,b)=>a[1].at-b[1].at)[0];if(!next||next[1].at>to)break;now=next[1].at;tasks.delete(next[0]);next[1].fn();}now=to;}};
}
test('autosave coalesces, waits for completion, and throttles from completion',()=>{
 const c=clock(),runs=[];const q=new AutosaveQueue(()=>runs.push(c.now()),c);
 q.changed();c.advance(200);q.changed();c.advance(499);assert.deepEqual(runs,[]);
 c.advance(500);assert.deepEqual(runs,[500]);q.changed();c.advance(5000);assert.equal(runs.length,1);
 q.finished();c.advance(5499);assert.equal(runs.length,1);c.advance(5500);assert.deepEqual(runs,[500,5500]);
 q.finished();c.advance(10000);assert.equal(runs.length,2);
});
test('continuous editing has a maximum cooldown',()=>{
 const c=clock(),runs=[];const q=new AutosaveQueue(()=>runs.push(c.now()),c);
 for(let t=0;t<1500;t+=200){c.advance(t);q.changed();}
 c.advance(1500);assert.deepEqual(runs,[1500]);
});
test('browser timers are called without a queue receiver',()=>{
 const originalSet=globalThis.setTimeout,originalClear=globalThis.clearTimeout;
 let receiver;
 try {
  globalThis.setTimeout=function(){receiver=this;return 1};
  globalThis.clearTimeout=function(){receiver=this};
  const q=new AutosaveQueue(()=>{});q.changed();assert.equal(receiver,undefined);
  q.changed();assert.equal(receiver,undefined);
 }finally{globalThis.setTimeout=originalSet;globalThis.clearTimeout=originalClear}
});
