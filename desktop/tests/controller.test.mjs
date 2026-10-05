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
 c.hover('c');c.action('f');assert.equal(calls[0].value.target,'/repo/c');assert.equal(c.selected,0);
 c.action('ArrowDown');assert.equal(c.mode,'keyboard');assert.equal(c.effectiveRow.id,'b');
 c.action('Escape');assert.equal(c.mode,'mouse');assert.equal(c.selected,1);assert.equal(c.effectiveRow.id,'c');
 c.action('ArrowDown');assert.equal(c.selected,2);
 c.hover('a');assert.equal(c.selected,2);assert.equal(c.effectiveRow.id,'a');
 c.replace([row('c'),row('a')]);assert.equal(c.selected,0);assert.equal(c.effectiveRow.id,'a');
});
test('open/copy commands and refresh use effective folder, uppercase copy URL supported',()=>{
 const {c,calls}=fixture();c.hover('a');
 for(const key of ['a','s','f','t'])c.action(key);
 for(const keys of ['ct','cT','CT','cs','cf'])for(const key of keys)c.action(key);
 assert.deepEqual(calls.slice(0,4).map(a=>a.value.kind),['agent','spec','folder','ticket']);
 assert.deepEqual(calls.slice(4).map(a=>a.value),['DEV-1','https://linear.app/bubblehouse/issue/DEV-1','https://linear.app/bubblehouse/issue/DEV-1','Spec','/repo/a']);
 c.action('r');c.action('R');c.action('settings');
 assert.deepEqual(calls.slice(-3),[{kind:'refresh',value:'/repo/a'},{kind:'refresh',value:''},{kind:'settings'}]);
 assert.equal(c.action('Q'),false);assert.equal(c.action('q'),false);assert.equal(c.action('o'),false);
});
test('chooser survives unchanged snapshots, supports keys, cancels when targets change',()=>{
 const {c,calls}=fixture();c.hover('b');c.action('c');c.action('T');
 assert.equal(c.choices.length,2);
 c.replace([row('a'),row('b',['DEV-2','DEV-3'])]);assert.equal(c.choices.length,2);
 c.action('ArrowDown');c.action('Enter');
 assert.equal(calls[0].value,'https://linear.app/bubblehouse/issue/DEV-3');
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
