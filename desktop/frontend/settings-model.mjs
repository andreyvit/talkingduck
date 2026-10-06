export const shortcutOptions = ['', ...Array.from({length:20},(_,i)=>String(i)), ...'ABCDEFGHIJKLMNOPQRSTUVWXYZ', ...'abcdefghijklmnopqrstuvwxyz'];
export const shortcutLabel = value => value === '' ? 'Hide' : value;

// Background snapshots must not replace a user's unfinished edit or selection.
export class PreferenceDraft {
 constructor(){this.revision=0;this.dirty=false;this.value=null;}
 sync(value){if(this.dirty)return false;this.value=structuredClone(value);return true;}
 edit(value){this.value=structuredClone(value);this.dirty=true;return ++this.revision;}
 result(revision,error){if(revision!==this.revision)return false;if(!error)this.dirty=false;return true;}
}
