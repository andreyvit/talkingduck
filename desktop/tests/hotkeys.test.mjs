import test from 'node:test';
import assert from 'node:assert/strict';
import {recordedHotkey,hotkeyLabel} from '../frontend/hotkey-model.mjs';
test('all sixteen modifier combinations including all four at once',()=>{
 const props=['ctrlKey','altKey','shiftKey','metaKey'],names=['Ctrl','Alt','Shift','Super'];
 for(let mask=0;mask<16;mask++){
  const e={code:'KeyT',key:'†'};props.forEach((p,i)=>e[p]=!!(mask&(1<<i)));
  assert.equal(recordedHotkey(e),[...names.filter((_,i)=>mask&(1<<i)),'T'].join('+'));
 }
 assert.equal(hotkeyLabel('Ctrl+Alt+Shift+Super+T'),'⌃⌥⇧⌘T');
});
test('physical punctuation, navigation, functions, repeats and unsupported keys',()=>{
 assert.equal(recordedHotkey({code:'Equal',key:'+',shiftKey:true}),'Shift+=');
 assert.equal(recordedHotkey({code:'PageUp',ctrlKey:true}),'Ctrl+Page Up');
 assert.equal(recordedHotkey({code:'F20'}),'F20');
 assert.equal(recordedHotkey({code:'ShiftLeft'}),null);
 assert.equal(recordedHotkey({code:'KeyT',repeat:true}),null);
 assert.throws(()=>recordedHotkey({code:'F21'}),/supported/);
 assert.throws(()=>recordedHotkey({code:'KeyT',getModifierState:k=>k==='Fn'}),/Fn/);
 assert.equal(hotkeyLabel(''),'Record shortcut');
 assert.equal(hotkeyLabel('Alt+Left'),'⌥←');
});
