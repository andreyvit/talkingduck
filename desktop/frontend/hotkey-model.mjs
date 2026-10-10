const named = {Equal:'=',Minus:'-',BracketLeft:'[',BracketRight:']',Quote:"'",Semicolon:';',Backslash:'\\',Comma:',',Period:'.',Slash:'/',Backquote:'`',Enter:'Return',Tab:'Tab',Space:'Space',Backspace:'Backspace',Delete:'Delete',Escape:'Escape',Home:'Home',PageUp:'Page Up',End:'End',PageDown:'Page Down',ArrowLeft:'Left',ArrowRight:'Right',ArrowUp:'Up',ArrowDown:'Down'};
const modifiers = [['ctrlKey','Ctrl','⌃'],['altKey','Alt','⌥'],['shiftKey','Shift','⇧'],['metaKey','Super','⌘']];
// Physical codes match the native registry's ANSI key map, including Option
// combinations that produce different characters or dead keys in KeyboardEvent.key.
export function recordedHotkey(event) {
 if (event.isComposing || event.repeat) return null;
 if (/^(Control|Alt|Shift|Meta)/.test(event.code)) return null;
 let key=named[event.code];
 if (/^Key[A-Z]$/.test(event.code)) key=event.code.slice(3);
 if (/^Digit[0-9]$/.test(event.code)) key=event.code.slice(5);
 if (/^F([1-9]|1[0-9]|20)$/.test(event.code)) key=event.code;
 if (!key) throw new Error('This key isn’t supported.');
 if (event.getModifierState?.('Fn') || event.getModifierState?.('AltGraph')) throw new Error('Fn/Globe and Alt Graph aren’t supported.');
 return [...modifiers.filter(([property])=>event[property]).map(([,name])=>name),key].join('+');
}
export function hotkeyLabel(value) {
 if (!value) return 'Record shortcut';
 const keys=value.split('+'),key=keys.pop();
 const labels={Return:'↩',Tab:'⇥',Backspace:'⌫',Delete:'⌦',Escape:'⎋',Left:'←',Right:'→',Up:'↑',Down:'↓',Home:'↖',End:'↘','Page Up':'⇞','Page Down':'⇟'};
 return modifiers.filter(([,name])=>keys.includes(name)).map(([, ,symbol])=>symbol).join('')+(labels[key]||key);
}
