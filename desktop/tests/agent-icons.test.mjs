import {test} from 'node:test';
import assert from 'node:assert/strict';
import {agentIdentity,agentIcon} from '../frontend/agent-icons.mjs';
test('agent icons retain names, conflicts, and distinct full vector motifs',()=>{
 const document={createElementNS(_ns,tag){return {tag,attrs:{},children:[],classList:{add(){}},setAttribute(k,v){this.attrs[k]=v},append(child){this.children.push(child)}}}};
 const shapes=['Codex','Claude','Grok'].map(name=>JSON.stringify(agentIcon(document,agentIdentity(name))));
 assert.equal(new Set(shapes).size,3);
 assert.deepEqual(agentIdentity('Claude!'),{id:'claude',name:'Claude',conflict:true});
 assert.equal(agentIdentity('?'),null);assert.equal(agentIdentity('--'),null);
});
