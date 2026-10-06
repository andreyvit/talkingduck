export function archiveName(row){
 return row.spec ? row.spec.split('/').at(-1).replace(/\.md$/i,'') : row.folder.split('/').filter(Boolean).at(-1);
}
export function archiveAge(value,now=new Date()){
 if(!value)return '';
 const date=new Date(value);if(Number.isNaN(date.getTime()))return '';
 const day=d=>Date.UTC(d.getFullYear(),d.getMonth(),d.getDate());
 const days=Math.max(0,Math.round((day(now)-day(date))/86400000));
 return days===0?'Today':days===1?'Yesterday':`${days} days ago`;
}
