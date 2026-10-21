import {test} from 'node:test';import assert from 'node:assert/strict';
import {visibleProviders,remaining,resetLabel,pingLabel} from '../frontend/limits-model.mjs';
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
