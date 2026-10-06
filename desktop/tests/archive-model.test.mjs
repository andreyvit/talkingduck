import {test} from 'node:test';
import assert from 'node:assert/strict';
import {archiveName,archiveAge} from '../frontend/archive-model.mjs';
test('archive names include summary rows and dates use local calendar days',()=>{
 assert.equal(archiveName({folder:'/work/back2',spec:'.spec/fix.md'}),'fix');
 assert.equal(archiveName({folder:'/work/back2',spec:''}),'back2');
 const now=new Date(2026,9,6,0,1);
 assert.equal(archiveAge(null,now),'');
 assert.equal(archiveAge('invalid',now),'');
 assert.equal(archiveAge(new Date(2026,9,6).toISOString(),now),'Today');
 assert.equal(archiveAge(new Date(2026,9,5,23,59).toISOString(),now),'Yesterday');
 assert.equal(archiveAge(new Date(2026,9,4).toISOString(),now),'2 days ago');
});
