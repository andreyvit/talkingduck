package inventory

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
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
		for _, tag := range []string{"rules", "user_info", "environment_context", "system-reminder", "memory-context"} {
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

// Compacted/reused chats can name historical tasks alongside the active one.
// Accept a narrow current-task label attached to a spec path, not recency alone.
func activeSpecReferences(rows []conversationEntry, specs []Object) map[string]Object {
	var latest map[string]Object
	for _, row := range rows {
		current := map[string]Object{}
		for _, line := range strings.Split(row.text, "\n") {
			for _, spec := range specs {
				path := str(spec["path"])
				base := filepath.Base(path)
				if !specReference(line, base) {
					continue
				}
				i := strings.Index(line, base)
				before, after := line[:i], line[i+len(base):]
				// Remove the directory prefix before checking a "Current task:" label.
				if start := strings.LastIndex(before, ".spec/"); start >= 0 {
					before = before[:start]
				}
				before = strings.TrimRight(before, " `*_[(:=-")
				after = strings.TrimLeft(after, " `*_]):=-")
				label := `(?:active|current) (?:zoo )?(?:task|spec)(?: file)?`
				prefix := matches(`(?i)`+label+`$`, before) && !matches(`(?i)\b(?:not|previous|previously|formerly|old|no longer)\b`, before)
				suffix := matches(`(?i)^(?:is (?:the )?)?`+label+`\b`, after)
				status := matches("(?i)^(?:frontmatter )?status:\\s*(?:executing|implementing)\\b", strings.ReplaceAll(after, "`", ""))
				if prefix || suffix || status {
					current[path] = Object{"kind": "explicit current task declaration", "at": row.at, "speaker": row.role, "excerpt": runePrefix(strings.TrimSpace(line), 500)}
				}
			}
		}
		if len(current) > 0 {
			latest = current
		} else if len(latest) > 0 {
			// Later discussion of another spec can signal another task switch.
			// Without a new declaration, fall back to the ordinary ambiguity rules.
			for _, spec := range specs {
				path := str(spec["path"])
				if latest[path] == nil && specReference(row.text, filepath.Base(path)) {
					latest = nil
					break
				}
			}
		}
	}
	return latest
}

func associate(session Object, specs []Object) string {
	rows, bounded := transcriptEvidence(session)
	session["zoo_skill_activity"] = skillActivity(rows)
	receipts := zooReceipts(rows)
	session["zoo_receipts"] = receipts
	receiptPath, receiptRole := receiptOwnership(receipts, specs)
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
	if receiptRole != "" && !truth(session["parent_id"]) {
		role, reason = receiptRole, "Explicit Zoo ownership receipt"
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
	// A generated title may omit the ticket, and planning may never name the spec.
	// Use user requests only, after stronger evidence; keep shared tickets ambiguous.
	if len(hits) == 0 {
		for _, row := range rows {
			if row.role != "user" {
				continue
			}
			ids := tickets(row.text)
			for _, s := range specs {
				matched := []string{}
				for _, id := range array(s["tickets"]) {
					if contains(ids, str(id)) {
						matched = append(matched, str(id))
					}
				}
				if len(matched) > 0 {
					hits[str(s["path"])] = Object{"kind": "unique ticket in user message", "at": row.at, "speaker": "user", "tickets": matched}
				}
			}
		}
	}
	// Current-task declarations supersede older names/tickets in a reused chat.
	// Multiple declarations in the same message remain ambiguous.
	if active := activeSpecReferences(rows, specs); len(active) > 0 {
		hits = active
	}
	if receiptPath != "" {
		hits = map[string]Object{receiptPath: {"kind": "explicit Zoo ownership receipt"}}
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
	value, why := "unknown", "No explicit spec or unique title/user-message ticket match"
	if len(hits) > 0 {
		value, why = "ambiguous", "Multiple specs match session references or tickets"
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
			finishTask(s)
		}
	}
	return report
}

func finishTask(s Object) {
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
		return
	}
	p := primaries[0]
	s["primary_session"] = Object{"engine": p["engine"], "id": p["id"], "app_session_id": p["app_session_id"], "client": p["client"]}
	activity := str(obj(p["activity"])["value"])
	skill := obj(p["zoo_skill_activity"])
	if str(w["stage_confidence"]) != "explicit" && len(skill) > 0 && timestamp(p["updated_at"]) >= timestamp(s["last_activity"]) {
		w["stage"], w["skill_evidence"], w["progress"] = skill["stage"], skill, "?"
		if str(skill["state"]) == "completed" {
			w["progress"] = "✓"
		}
	}
	applyZooReceipts(w, p, str(s["path"]))
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
