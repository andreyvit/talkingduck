import {test} from 'node:test';
import assert from 'node:assert/strict';
import {PreferenceDraft} from '../frontend/settings-model.mjs';
test('snapshots and old saves preserve unfinished edits',()=>{
 const d=new PreferenceDraft();assert.equal(d.sync({id:'zed'}),true);
 const revision=d.edit({id:'custom',path:'/unfinished'});
 assert.equal(d.sync({id:'zed'}),false);assert.equal(d.value.path,'/unfinished');
 d.edit({id:'custom',path:'/new'});assert.equal(d.result(revision,''),false);assert.equal(d.dirty,true);
 assert.equal(d.result(d.revision,'Invalid path'),true);assert.equal(d.dirty,true);
 assert.equal(d.result(d.revision,''),true);assert.equal(d.dirty,false);
 assert.equal(d.sync({id:'custom',path:'/new'}),true);
});
