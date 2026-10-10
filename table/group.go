package table

// GroupRows keeps each checkout's rows together and emits linked worktrees after
// their original checkout. Unmonitored parents are not fabricated as empty rows.
func GroupRows(rows []Row) []Row {
	groups := map[string][]Row{}
	order := []string{}
	parents := map[string]string{}
	for _, r := range rows {
		if _, ok := groups[r.Group]; !ok {
			order = append(order, r.Group)
		}
		groups[r.Group] = append(groups[r.Group], r)
		parents[r.Group] = r.ParentGroup
	}
	result := make([]Row, 0, len(rows))
	seen := map[string]bool{}
	var emit func(string)
	emit = func(path string) {
		if seen[path] {
			return
		}
		seen[path] = true
		result = append(result, groups[path]...)
		for _, child := range order {
			if parents[child] == path {
				emit(child)
			}
		}
	}
	for _, path := range order {
		if len(groups[parents[path]]) == 0 {
			emit(path)
		}
	}
	for _, path := range order {
		emit(path)
	} // Preserve rows even with malformed cyclic input.
	return result
}
