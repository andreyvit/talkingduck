// Match full checkout identities, including multiple task rows in one checkout.
export function rowRefreshing(row,snapshot){
 return !!snapshot.loading && !!snapshot.refreshingPath && row.group===snapshot.refreshingPath;
}
