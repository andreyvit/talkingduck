import test from 'node:test';
import assert from 'node:assert/strict';
import {shortcutOptions,shortcutLabel} from '../frontend/settings-model.mjs';
test('shortcut order: Hide, 0–19, uppercase, lowercase',()=>{
 assert.deepEqual(shortcutOptions.slice(0,21),['',...Array.from({length:20},(_,i)=>String(i))]);
 assert.equal(shortcutOptions[21],'A');assert.equal(shortcutOptions[46],'Z');assert.equal(shortcutOptions[47],'a');assert.equal(shortcutOptions.at(-1),'z');assert.equal(shortcutLabel(''),'Hide');
});
