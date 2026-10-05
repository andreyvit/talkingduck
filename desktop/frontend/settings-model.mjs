export const shortcutOptions = ['', ...Array.from({length:20},(_,i)=>String(i)), ...'ABCDEFGHIJKLMNOPQRSTUVWXYZ', ...'abcdefghijklmnopqrstuvwxyz'];
export const shortcutLabel = value => value === '' ? 'Hide' : value;
