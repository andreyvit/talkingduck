import {test} from 'node:test';
import assert from 'node:assert/strict';
import {InventoryController} from '../frontend/controller.mjs';
const row=(id,tickets=['DEV-1'])=>({id,group:'/repo/'+id,cells:[{lines:[
 {items:[{text:id,chord:'f',hidden:true,copy:'/repo/'+id,link:{kind:'folder',target:'/repo/'+id}}]},
 ...tickets.map(text=>({items:[{text,chord:'t',copy:text,link:{kind:'ticket',target:text}}]})),
 {items:[{text:'Spec',chord:'s',copy:'Spec',link:{kind:'spec',target:'/spec/'+id}}]},
 {items:[{text:'Agent',chord:'a',link:{kind:'agent',target:'agent:'+id}}]},
]}]});
function fixture(){const calls=[];const c=new InventoryController(a=>calls.push(a));c.replace([row('a'),row('b',['DEV-2','DEV-3']),row('c')]);return {c,calls};}
test('mouse and keyboard keep independent rows; Esc hides rather than deselects',()=>{
 const {c,calls}=fixture();
 assert.equal(c.mode,'mouse');assert.equal(c.effectiveRow,undefined);
 c.hover('c');assert.equal(c.selected,0);
 c.action('ArrowDown');assert.equal(c.mode,'keyboard');assert.equal(c.effectiveRow.id,'b');
 c.action('Escape');assert.equal(c.mode,'mouse');assert.equal(c.selected,1);assert.equal(c.effectiveRow.id,'c');
 c.action('ArrowDown');assert.equal(c.selected,2);
 c.hover('a');assert.equal(c.selected,2);assert.equal(c.effectiveRow.id,'a');
 c.replace([row('c'),row('a')]);assert.equal(c.selected,0);assert.equal(c.effectiveRow.id,'a');
});
test('open/copy commands and refresh use effective folder, uppercase copy URL supported',()=>{
 const {c,calls}=fixture();c.hover('a');
 for(const key of ['a','s','f','t'])c.action(key);
 for(const keys of ['ct','cT','CT','cs','cS','cf'])for(const key of keys)c.action(key);
 assert.deepEqual(calls.slice(0,4).map(a=>a.value.kind),['agent','spec','folder','ticket']);
 assert.deepEqual(calls.slice(4).map(a=>a.value),['DEV-1','DEV-1','DEV-1','Spec','/spec/a','/repo/a']);
 c.action('r');c.action('R');c.action('settings');
 assert.deepEqual(calls.slice(-3),[{kind:'refresh',value:'/repo/a'},{kind:'refresh',value:''},{kind:'settings'}]);
 assert.equal(c.action('Q'),false);assert.equal(c.action('q'),false);assert.equal(c.action('o'),false);
});
test('chooser survives unchanged snapshots, supports keys, cancels when targets change',()=>{
 const {c,calls}=fixture();c.hover('b');c.action('c');c.action('T');
 assert.equal(c.choices.length,2);
 c.replace([row('a'),row('b',['DEV-2','DEV-3'])]);assert.equal(c.choices.length,2);
 c.action('ArrowDown');c.action('Enter');
 assert.equal(calls[0].value,'DEV-3');
 c.action('t');c.replace([row('b',['DEV-9'])]);assert.equal(c.choices.length,0);
});
test('hover changes cancel pending copy and missing rows cannot receive actions',()=>{
 const {c,calls}=fixture();c.hover('a');c.action('c');c.hover('b');
 assert.equal(c.chord,'');
 c.action('Escape');c.replace([]);c.action('r');c.action('f');c.action('c');
 assert.equal(calls.length,0);assert.equal(c.selected,0);
});
test('empty startup snapshot does not poison later updates',()=>{
 const c=new InventoryController(()=>{});
 c.replace(null);assert.equal(c.rows.length,0);
 c.replace([row('a')]);c.action('ArrowDown');assert.equal(c.effectiveRow.id,'a');
});
test('Shift-X archives the effective row; lowercase x does not',()=>{
 const {c,calls}=fixture();
 for(const r of c.rows)r.archive={kind:'archive',target:'.spec/'+r.id+'.md'};
 c.hover('c');assert.equal(c.action('x'),false);c.action('X');
 assert.deepEqual(calls[0],{kind:'archive',value:{folder:'/repo/c',spec:'.spec/c.md'}});
 c.action('ArrowUp');c.action('X');
 assert.equal(calls[1].value.spec,'.spec/b.md');
});
test('editor and Git actions use the effective worktree without adding cells',()=>{
 const {c,calls}=fixture();
 for(const row of c.rows)row.actions=['editor','git'].map((kind,i)=>({text:kind,chord:i?'g':'e',link:{kind,target:row.group}}));
 c.hover('c');c.action('e');c.action('g');
 assert.deepEqual(calls.map(a=>a.value),[{kind:'editor',target:'/repo/c'},{kind:'git',target:'/repo/c'}]);
 c.action('ArrowUp');c.action('e');assert.equal(calls.at(-1).value.target,'/repo/b');
});
test('ticket URL copying resolves current preferences in Go',()=>{
 const {c,calls}=fixture();c.hover('a');c.action('c');c.action('T');
 assert.deepEqual(calls,[{kind:'copy-ticket-url',value:'DEV-1',label:'DEV-1'}]);
});

test('Shift-S reveals the effective spec without changing the regular open action',()=>{
 const actions=[];const c=new InventoryController(a=>actions.push(a));
 c.replace([{id:'one',group:'/work',cells:[{lines:[{items:[{text:'Spec',chord:'s',link:{kind:'spec',target:'/work/.spec/task.md'}}]}]}]}]);
 c.hover('one');c.action('S');c.action('s');
 assert.equal(actions[0].value.kind,'reveal-spec');assert.equal(actions[1].value.kind,'spec');
 assert.equal(actions[0].value.target,'/work/.spec/task.md');
});

test('row actions pin the hovered row before handing off focus',()=>{
 const {c,calls}=fixture();
 c.hover('c');c.action('t');
 assert.equal(c.mode,'keyboard');assert.equal(c.selected,2);
 c.hover('a',{active:false});
 assert.equal(c.hovered,'a');assert.equal(c.mode,'keyboard');
 c.action('s');assert.equal(calls.at(-1).value.target,'/spec/c');
 c.hover('c');assert.equal(c.mode,'keyboard');
 c.hover('a');assert.equal(c.mode,'mouse');assert.equal(c.selected,2);
});
test('mouse activation resets mode, direct actions and chooser retain their row',()=>{
 const {c}=fixture();c.hover('c');c.action('s');
 c.hover('c',{active:false,clicked:true});assert.equal(c.mode,'mouse');
 c.perform({kind:'open',value:{kind:'spec',target:'/spec/a'}},'a');
 assert.equal(c.mode,'keyboard');assert.equal(c.selected,0);
 c.hover('b');c.action('t');assert.equal(c.mode,'mouse');
 c.choose(1);assert.equal(c.mode,'keyboard');assert.equal(c.selected,1);
 c.replace([row('b'),row('c')]);assert.equal(c.selected,0);
});

test('activation follows the row under the pointer, without requiring a click',()=>{
 const {c}=fixture();c.hover('c');c.action('s');
 c.activate();assert.equal(c.mode,'keyboard');assert.equal(c.selected,2);
 c.activate('c');assert.equal(c.mode,'keyboard');
 c.hover('a',{active:false});assert.equal(c.mode,'keyboard');
 c.activate('a');assert.equal(c.mode,'mouse');assert.equal(c.effectiveRow.id,'a');
 assert.equal(c.selected,2);
});
