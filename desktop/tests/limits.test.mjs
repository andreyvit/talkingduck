import {test} from 'node:test';import assert from 'node:assert/strict';
import {visibleProviders,remaining,resetLabel,pingLabel,pingReading} from '../frontend/limits-model.mjs';
test('Auto tracks recent use, On preserves unavailable slots, Off hides',()=>{
 const snapshot={providers:[{id:'codex',recently_used:true},{id:'claude',recently_used:false},{id:'grok',recently_used:true}]};
 assert.deepEqual(visibleProviders(snapshot,{codex:'auto',claude:'on',grok:'off'}).map(p=>p.id),['codex','claude']);
 assert.deepEqual(visibleProviders(snapshot,{codex:'auto',claude:'auto',grok:'off'}).map(p=>p.id),['codex']);
 assert.equal(remaining({remaining_percent:null}),null);assert.equal(remaining({remaining_percent:0}),0);
});
test('Reset countdown and outage age use elapsed time',()=>{
 const now=Date.UTC(2026,9,21,12);assert.equal(resetLabel(new Date(now+8124000).toISOString(),now),'Reset in 2:15:24');
 assert.equal(resetLabel(new Date(now-1000).toISOString(),now),'Reset pending');
 assert.equal(pingLabel({state:'unreachable',failed_since:new Date(now-3000).toISOString()},now),'3 s no conn');
 assert.equal(pingLabel({state:'reachable',rtt_ms:20.4},now),'20 ms ping');
});

test('Dashboard ping reserves rolling width with a three-digit floor',()=>{
 assert.equal(pingReading({state:'reachable',rtt_ms:8,digits:1}).digits,3);
 assert.equal(pingReading({state:'reachable',rtt_ms:99,digits:4}).digits,4);
 assert.equal(pingReading({state:'reachable',rtt_ms:99,digits:3}).digits,3);
 assert.deepEqual(pingReading({state:'reachable',rtt_ms:99,digits:4}),{label:'99 ms ping',number:'99',unit:' ms ping',digits:4});
});

test('limits timers sleep for static dates and stop after reset',async()=>{
 const {limitsUpdateDelay}=await import('../frontend/limits-model.mjs');
 const now=Date.parse('2026-10-21T07:00:00Z');
 const p={id:'codex',recently_used:true,remaining_percent:80,resets_at:new Date(now+172800000).toISOString()};
 assert.equal(limitsUpdateDelay({providers:[p]}, {},now),86400001);
 p.resets_at=new Date(now+3600000).toISOString();assert.equal(limitsUpdateDelay({providers:[p]}, {},now),1000);
 p.resets_at=new Date(now-1).toISOString();assert.equal(limitsUpdateDelay({providers:[p]}, {},now),null);
 assert.equal(limitsUpdateDelay({ping:{state:'unreachable'}},{ping:'off'},now),null);
 assert.equal(limitsUpdateDelay({ping:{state:'unreachable'}},{ping:'vertical'},now),1000);
 assert.equal(limitsUpdateDelay({providers:[p]},{codex:'off'},now),null);
});
