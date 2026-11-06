import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {runInNewContext} from 'node:vm';

class Element {
 constructor(){this.children=[];this.dataset={};}
 append(...nodes){for(const node of nodes){node.remove();node.parent=this;this.children.push(node)}}
 remove(){if(this.parent){this.parent.children.splice(this.parent.children.indexOf(this),1);this.parent=null}}
 contains(node){return this===node||this.children.some(child=>child.contains(node))}
 setAttribute(){}
 get firstChild(){return this.children[0]||null}
 get nextSibling(){return this.parent?.children[this.parent.children.indexOf(this)+1]||null}
 insertBefore(node,next){node.remove();node.parent=this;this.children.splice(next?this.children.indexOf(next):this.children.length,0,node)}
}
const source=readFileSync(new URL('../frontend/settings.js',import.meta.url),'utf8');
const render=source.slice(source.indexOf('function renderFolders()'),source.indexOf("$('folders').addEventListener"));
test('new folders appear in path order while a changed row keeps focus until blur',()=>{
 const table=new Element(),document={activeElement:null,createElement:()=>new Element()};
 const folder=path=>({path,visible:false,menubar_item_shortcut:''});
 const context={document,folderSnapshot:[folder('/repo/back2')],$:()=>table,select:()=>new Element(),shortcutOptions:[],shortcutLabel:x=>x};
 runInNewContext(render+';renderFolders()',context);
 const original=table.firstChild,checkbox=original.children[0].firstChild;
 document.activeElement=checkbox;
 context.folderSnapshot=[folder('/repo/back12'),{...folder('/repo/back2'),visible:true},folder('/other/repo'),folder('/repo/back3')];
 runInNewContext('renderFolders()',context);
 assert.deepEqual(table.children.map(row=>row.dataset.path),['/other/repo','/repo/back2','/repo/back3','/repo/back12']);
 assert.equal(table.children[1],original);
 assert.equal(table.contains(checkbox),true);
 assert.equal(checkbox.checked,false);
 document.activeElement=null;
 runInNewContext('renderFolders()',context);
 assert.equal(table.children[1].children[0].firstChild.checked,true);
 context.folderSnapshot=[];
 runInNewContext('renderFolders()',context);
 assert.equal(table.children.length,1);
 assert.match(table.firstChild.firstChild.textContent,/No folders yet/);
});
