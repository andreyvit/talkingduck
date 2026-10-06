export const shortcutOptions = ['', ...Array.from({length:20},(_,i)=>String(i)), ...'ABCDEFGHIJKLMNOPQRSTUVWXYZ', ...'abcdefghijklmnopqrstuvwxyz'];
export const shortcutLabel = value => value === '' ? 'Hide' : value;

// Background snapshots must not replace a user's unfinished edit or selection.
export class PreferenceDraft {
 constructor(){this.revision=0;this.dirty=false;this.value=null;}
 sync(value){if(this.dirty)return false;this.value=structuredClone(value);return true;}
 edit(value){this.value=structuredClone(value);this.dirty=true;return ++this.revision;}
 result(revision,error){if(revision!==this.revision)return false;if(!error)this.dirty=false;return true;}
}

// One in-flight save per form. Continuous typing cannot postpone a save forever.
export class AutosaveQueue {
 constructor(save,{now=Date.now,setTimer=setTimeout,clearTimer=clearTimeout,quiet=300,maxWait=1500,throttle=500}={}){
  Object.assign(this,{save,now,setTimer,clearTimer,quiet,maxWait,throttle});
  this.first=null;this.last=0;this.lastFinished=-Infinity;this.running=false;this.timer=null;
 }
 changed(){const now=this.now();if(this.first===null)this.first=now;this.last=now;this.schedule();}
 schedule(){
  if(this.timer!==null){this.clearTimer(this.timer);this.timer=null;}
  if(this.running||this.first===null)return;
  const deadline=Math.max(Math.min(this.last+this.quiet,this.first+this.maxWait),this.lastFinished+this.throttle);
  this.timer=this.setTimer(()=>{this.timer=null;this.first=null;this.running=true;this.save();},Math.max(0,deadline-this.now()));
 }
 finished(){this.running=false;this.lastFinished=this.now();this.schedule();}
}
