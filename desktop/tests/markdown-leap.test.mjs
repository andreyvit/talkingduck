import test from 'node:test';
import assert from 'node:assert/strict';
import {LeapSearch} from '../frontend/markdown-leap.mjs';
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
