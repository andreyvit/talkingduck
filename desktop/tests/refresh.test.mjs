import {test} from 'node:test';
import assert from 'node:assert/strict';
import {rowRefreshing} from '../frontend/refresh-model.mjs';
test('refresh follows full checkout identity, not a repeated folder name or agent presence',()=>{
 const state={loading:true,refreshingPath:'/one/back9'};
 assert.equal(rowRefreshing({group:'/one/back9',cells:[]},state),true);
 assert.equal(rowRefreshing({group:'/two/back9'},state),false);
 assert.equal(rowRefreshing({group:'/one/back9'},{...state,loading:false}),false);
 assert.equal(rowRefreshing({group:''},{loading:true,refreshingPath:''}),false);
});
