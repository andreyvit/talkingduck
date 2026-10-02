package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/term"
	"golang.org/x/text/width"
)

func infoFile(path string) bool {
	if !strings.HasSuffix(strings.ToLower(path), ".md") {
		return false
	}
	for _, p := range strings.Split(filepath.ToSlash(path), "/") {
		if contains([]string{".spec", ".proposal", ".proposals", ".tasks"}, p) {
			return true
		}
	}
	return false
}
func gitState(repo Object) Object {
	tracked, untracked, info := map[string]bool{}, map[string]bool{}, map[string]bool{}
	all := map[string]bool{}
	for _, v := range array(repo["local_changes"]) {
		c := obj(v)
		p := str(c["path"])
		if str(c["status"]) == "rename/copy source" {
			continue
		}
		all[p] = true
		if str(c["status"]) != "??" {
			tracked[p] = true
		}
		if infoFile(p) {
			info[p] = true
		} else if str(c["status"]) == "??" {
			untracked[p] = true
		}
	}
	var ahead, behind any
	if b, e := git(str(repo["path"]), "rev-list", "--left-right", "--count", "@{upstream}...HEAD"); e == nil {
		f := strings.Fields(b)
		if len(f) == 2 {
			n, e1 := strconv.Atoi(f[0])
			m, e2 := strconv.Atoi(f[1])
			if e1 == nil && e2 == nil {
				behind, ahead = n, m
			}
		}
	}
	allInfo, allMarkdown := true, true
	for p := range all {
		allInfo = allInfo && infoFile(p)
		allMarkdown = allMarkdown && strings.HasSuffix(strings.ToLower(p), ".md")
	}
	cleanliness := "DIRTY"
	if allInfo {
		cleanliness = "CLEAN"
	} else if allMarkdown {
		cleanliness = "MARKDN"
	}
	return Object{"tracked": len(tracked), "untracked": len(untracked), "informational": len(info), "ahead": ahead, "behind": behind, "cleanliness": cleanliness, "warnings": []string{}, "scope": "Current staged, unstaged, and untracked paths only; remote counts use local upstream ref"}
}
func transcriptEvidence(session Object) ([]conversationEntry, bool) {
	path := str(session["transcript_path"])
	f, e := os.Open(path)
	if e != nil {
		return nil, false
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil {
		return nil, false
	}
	bounded := st.Size() > tailBytes
	var data []byte
	if bounded {
		head, _ := io.ReadAll(io.LimitReader(f, 256*1024))
		if i := bytes.LastIndexByte(head, '\n'); i >= 0 {
			head = head[:i+1]
		} else {
			head = nil
		}
		if _, e = f.Seek(st.Size()-tailBytes, io.SeekStart); e != nil {
			return nil, true
		}
		tail, _ := io.ReadAll(io.LimitReader(f, tailBytes))
		if i := bytes.IndexByte(tail, '\n'); i >= 0 {
			tail = tail[i+1:]
		} else {
			tail = nil
		}
		data = append(head, tail...)
	} else {
		data, _ = io.ReadAll(io.LimitReader(f, tailBytes))
	}
	records, _ := parseRows(data, false)
	rows := []conversationEntry{}
	for _, r := range records {
		typ := str(r["type"])
		role, text := "", ""
		if str(session["engine"]) == "codex" {
			p := obj(r["payload"])
			if typ == "event_msg" && contains([]string{"user_message", "agent_message"}, str(p["type"])) {
				role = "assistant"
				if str(p["type"]) == "user_message" {
					role = "user"
				}
				text = textContent(p["message"])
			} else if typ == "response_item" && str(p["type"]) == "message" && contains([]string{"user", "assistant"}, str(p["role"])) {
				role, text = str(p["role"]), textContent(p["content"])
			} else {
				continue
			}
		} else if typ == "user" || typ == "assistant" {
			role = typ
			m := r
			if value, ok := r["message"]; ok {
				m = obj(value)
			}
			text = textContent(m["content"])
		} else {
			continue
		}
		for _, tag := range []string{"rules", "user_info", "environment_context", "system-reminder"} {
			text = rx(`(?s)<`+tag+`>.*?</`+tag+`>`).ReplaceAllString(text, "")
		}
		if strings.HasPrefix(text, "# AGENTS.md instructions") || strings.HasPrefix(text, "<environment_context>") {
			continue
		}
		if strings.TrimSpace(text) != "" {
			rows = append(rows, conversationEntry{role, text, r["timestamp"]})
		}
	}
	return rows, bounded
}
func specReference(text, base string) bool {
	offset := 0
	for {
		index := strings.Index(text[offset:], base)
		if index < 0 {
			return false
		}
		start := offset + index
		end := start + len(base)
		word := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' }
		left, right := true, true
		if start > 0 {
			r, _ := utf8.DecodeLastRuneInString(text[:start])
			left = !word(r)
		}
		if end < len(text) {
			r, _ := utf8.DecodeRuneInString(text[end:])
			right = !word(r) && r != '.'
		}
		if left && right {
			return true
		}
		offset = end
		if offset >= len(text) {
			return false
		}
	}
}
func intersects(a []string, b any) bool {
	for _, v := range array(b) {
		if contains(a, str(v)) {
			return true
		}
	}
	return false
}
func associate(session Object, specs []Object) string {
	rows, bounded := transcriptEvidence(session)
	session["zoo_skill_activity"] = skillActivity(rows)
	prompts := []string{}
	for _, row := range rows {
		if row.role == "user" && len(prompts) < 3 {
			prompts = append(prompts, row.text)
		}
	}
	firstPrompt := strings.Join(prompts, "\n")
	review := matches(`(?i)findings only|do not (?:modify|edit) files|do not implement|read.only review`, firstPrompt) && matches(`(?i)review|zoo-ensure-safe-deploy`, firstPrompt)
	role, reason := "unknown", "Client role unavailable"
	switch {
	case truth(session["parent_id"]):
		role, reason = "secondary", "Explicit parent session"
	case str(session["client"]) == "headless" && review:
		role, reason = "secondary", "Headless session with review-only prompt"
	case str(session["client"]) == "headless":
		reason = "Headless does not establish whether this is a primary task"
	case contains([]string{"desktop", "cli"}, str(session["client"])):
		role, reason = "primary", "Top-level interactive session; inferred from client metadata"
	}
	session["task_role"] = Object{"value": role, "reason": reason, "confidence": "inferred"}
	if bounded {
		warn(session, "Association uses transcript head and tail; middle omitted")
	}
	hits := map[string]Object{}
	for _, s := range specs {
		path := str(s["path"])
		for _, row := range rows {
			if specReference(row.text, filepath.Base(path)) {
				hits[path] = Object{"kind": "explicit spec reference", "at": row.at, "speaker": row.role}
			}
		}
	}
	titleTickets := tickets(str(session["title"]))
	if len(hits) == 0 {
		for _, s := range specs {
			if intersects(titleTickets, s["tickets"]) {
				hits[str(s["path"])] = Object{"kind": "unique ticket in session title", "at": session["updated_at"]}
			}
		}
	}
	if len(hits) > 1 {
		supported := []string{}
		for _, s := range specs {
			p := str(s["path"])
			if hits[p] != nil && intersects(titleTickets, s["tickets"]) {
				supported = append(supported, p)
			}
		}
		if len(supported) == 1 {
			path := supported[0]
			ev := hits[path]
			ev["corroboration"] = "Unique ticket in session title"
			hits = map[string]Object{path: ev}
		}
	}
	if len(hits) == 1 {
		for path, evidence := range hits {
			evidence["value"], evidence["confidence"], evidence["source"] = path, "inferred", first(session["transcript_path"], session["source"])
			session["spec_association"] = evidence
			fields := []string{}
			for _, v := range array(session["unknown_fields"]) {
				if str(v) != "spec_association" {
					fields = append(fields, str(v))
				}
			}
			session["unknown_fields"] = fields
			return path
		}
	}
	candidates := []string{}
	for p := range hits {
		candidates = append(candidates, p)
	}
	sort.Strings(candidates)
	value, why := "unknown", "No explicit spec or unique title ticket match"
	if len(hits) > 0 {
		value, why = "ambiguous", "Multiple spec references"
	}
	session["spec_association"] = Object{"value": value, "candidates": candidates, "reason": why}
	return ""
}
func objects(v any) []Object {
	out := []Object{}
	for _, x := range array(v) {
		out = append(out, obj(x))
	}
	return out
}
func enrich(report Object) Object {
	report["schema_version"] = 2
	for _, repo := range objects(report["repositories"]) {
		if truth(repo["error"]) {
			repo["git"] = Object{"error": repo["error"]}
		} else {
			repo["git"] = gitState(repo)
		}
		specs := objects(repo["specs"])
		byPath := map[string]Object{}
		for _, s := range specs {
			b, _ := os.ReadFile(filepath.Join(str(repo["path"]), str(s["path"])))
			s["workflow"] = taskState(s, string(b))
			s["sessions"] = []Object{}
			s["primary_session"] = nil
			s["display_title"] = rx(`(?i)^\d{8}-(?:(?:dev-)?\d+-)*`).ReplaceAllString(str(s["title"]), "")
			byPath[str(s["path"])] = s
		}
		unassigned := []Object{}
		for _, session := range objects(repo["agents"]) {
			path := associate(session, specs)
			if owner := byPath[path]; owner != nil {
				owner["sessions"] = append(objects(owner["sessions"]), session)
			} else {
				unassigned = append(unassigned, session)
			}
		}
		delete(repo, "agents")
		repo["unassigned_sessions"] = unassigned
		for _, s := range specs {
			primaries := []Object{}
			active := 0
			for _, session := range objects(s["sessions"]) {
				if str(obj(session["task_role"])["value"]) == "primary" {
					primaries = append(primaries, session)
					if contains([]string{"running", "waiting"}, str(obj(session["activity"])["value"])) {
						active++
					}
				}
			}
			sort.Slice(primaries, func(i, j int) bool {
				a, b := timestamp(primaries[i]["updated_at"]), timestamp(primaries[j]["updated_at"])
				if a != b {
					return a > b
				}
				return str(primaries[i]["id"]) > str(primaries[j]["id"])
			})
			s["primary_conflict"] = active > 1
			w := obj(s["workflow"])
			w["interaction"], w["questions"] = "?", nil
			if len(primaries) == 0 {
				continue
			}
			p := primaries[0]
			s["primary_session"] = Object{"engine": p["engine"], "id": p["id"]}
			activity := str(obj(p["activity"])["value"])
			skill := obj(p["zoo_skill_activity"])
			if len(skill) > 0 && timestamp(p["updated_at"]) >= timestamp(s["last_activity"]) {
				w["stage"], w["skill_evidence"], w["progress"] = skill["stage"], skill, "?"
				if str(skill["state"]) == "completed" {
					w["progress"] = "✓"
				}
			}
			if activity == "running" {
				w["progress"] = "▶"
			}
			q := str(obj(p["question"])["value"])
			if activity == "waiting" || q == "yes" {
				w["interaction"] = "!"
			} else if q == "no" {
				w["interaction"] = ""
			}
			pending := objects(p["pending_questions"])
			if len(pending) > 0 {
				n := 0
				for _, q := range pending {
					n += max(1, len(array(q["prompts"])))
				}
				w["questions"] = n
			} else if q == "no" {
				w["questions"] = 0
			}
		}
	}
	return report
}
func cellWidth(s string) int {
	n := 0
	for _, r := range s {
		if unicode.Is(unicode.Mn, r) || r == '\ufe0e' || r == '\ufe0f' {
			continue
		}
		kind := width.LookupRune(r).Kind()
		if kind == width.EastAsianWide || kind == width.EastAsianFullwidth {
			n += 2
		} else {
			n++
		}
	}
	return n
}
func clean(s string) string { return strings.Join(strings.Fields(s), " ") }
func wrap(text string, size int) []string {
	result := []string{}
	var current strings.Builder
	n := 0
	for _, r := range clean(text) {
		w := cellWidth(string(r))
		if n > 0 && n+w > size {
			result = append(result, current.String())
			current.Reset()
			n = 0
		}
		current.WriteRune(r)
		n += w
	}
	return append(result, current.String())
}
func display(v any) string {
	if v == nil {
		return "?"
	}
	return fmt.Sprint(v)
}
func render(report Object, size int) string {
	rows := [][]string{}
	for _, repo := range objects(report["repositories"]) {
		name := filepath.Base(str(repo["path"]))
		g := obj(repo["git"])
		if truth(g["error"]) {
			rows = append(rows, []string{name, "--", "--", "0", "--", "--", "?", "?", "?/?", "ERROR", "", "", "⇣? ⇡?", str(g["error"])})
			continue
		}
		tracked := "±" + display(g["tracked"])
		untracked := "?" + display(g["untracked"])
		info := "ℹ︎" + display(g["informational"])
		remote := "⇣" + display(g["behind"]) + " ⇡" + display(g["ahead"])
		specs := objects(repo["specs"])
		if len(specs) == 0 {
			rows = append(rows, []string{name, "--", "--", "0", "--", "0/0", "", "", "0/0", tracked, untracked, info, remote, str(g["cleanliness"])})
			continue
		}
		for _, s := range specs {
			w, p := obj(s["workflow"]), obj(s["primary_session"])
			agent := map[string]string{"codex": "Codex", "claude": "Claude", "grok": "Grok"}[str(p["engine"])]
			if agent == "" {
				agent = "?"
			}
			if truth(s["primary_conflict"]) {
				agent += "!"
			}
			ids := []string{}
			for _, id := range array(s["tickets"]) {
				ids = append(ids, str(id))
			}
			ticket := strings.Join(ids, "\n")
			if ticket == "" {
				ticket = "--"
			}
			rows = append(rows, []string{name, agent, ticket, display(w["round"]), str(w["stage"]), display(w["subtasks_done"]) + "/" + display(w["subtasks_total"]), str(w["progress"]), str(w["interaction"]), display(w["questions"]) + "/" + display(w["suggestions"]), tracked, untracked, info, remote, str(s["display_title"])})
		}
	}
	widths := []int{6, 6, 8, 1, 4, 5, 1, 1, 3, 2, 2, 2, 5}
	for _, row := range rows {
		for i, value := range row[:len(widths)] {
			for _, line := range strings.Split(value, "\n") {
				widths[i] = max(widths[i], cellWidth(clean(line)))
			}
		}
	}
	total := len(widths)
	for _, w := range widths {
		total += w
	}
	lastWidth := max(8, size-total)
	out := []string{}
	for _, row := range rows {
		cells := [][]string{}
		height := 0
		for i, value := range row {
			var lines []string
			if i == len(widths) {
				lines = wrap(value, lastWidth)
			} else {
				for _, line := range strings.Split(value, "\n") {
					lines = append(lines, clean(line))
				}
			}
			cells = append(cells, lines)
			height = max(height, len(lines))
		}
		for line := 0; line < height; line++ {
			values := []string{}
			for i, cell := range cells {
				value := ""
				if line < len(cell) {
					value = cell[line]
				}
				if i < len(widths) {
					value += strings.Repeat(" ", max(0, widths[i]-cellWidth(value)))
				}
				values = append(values, value)
			}
			out = append(out, strings.TrimRight(strings.Join(values, " "), " "))
		}
	}
	return strings.Join(out, "\n")
}
func tableWidth(requested int) int {
	if requested > 0 {
		return requested
	}
	if n, e := strconv.Atoi(os.Getenv("COLUMNS")); e == nil && n > 0 {
		return n
	}
	if w, _, e := term.GetSize(int(os.Stdout.Fd())); e == nil && w > 0 {
		return w
	}
	return 120
}
func printOverview(report Report, asJSON bool, size int) error {
	b, e := json.Marshal(report)
	if e != nil {
		return e
	}
	var data Object
	if e = json.Unmarshal(b, &data); e != nil {
		return e
	}
	enrich(data)
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}
	fmt.Println(render(data, tableWidth(size)))
	if report.AgentError != "" {
		fmt.Fprintln(os.Stderr, report.AgentError)
	}
	for _, s := range objects(data["agent_sources"]) {
		if str(s["status"]) != "available" {
			fmt.Fprintf(os.Stderr, "%s: %s coverage; see JSON source warnings\n", s["engine"], s["status"])
		}
	}
	for _, r := range objects(data["repositories"]) {
		warnings := append(array(r["warnings"]), array(obj(r["git"])["warnings"])...)
		for _, w := range warnings {
			fmt.Fprintf(os.Stderr, "%s: %s\n", filepath.Base(str(r["path"])), w)
		}
	}
	return nil
}
