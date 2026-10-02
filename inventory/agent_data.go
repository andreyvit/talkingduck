package inventory

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Object contains evolving private provider fields and the public evidence payload.
// Typed Git records stay separate; missing provider keys never imply false/idle.
type Object = map[string]any

func obj(v any) Object {
	if m, ok := v.(map[string]any); ok && m != nil {
		return m
	}
	return Object{}
}
func str(v any) string { s, _ := v.(string); return s }
func array(v any) []any {
	if a, ok := v.([]any); ok {
		return a
	}
	if a, ok := v.([]string); ok {
		r := make([]any, len(a))
		for i, x := range a {
			r[i] = x
		}
		return r
	}
	if a, ok := v.([]Object); ok {
		r := make([]any, len(a))
		for i, x := range a {
			r[i] = x
		}
		return r
	}
	return nil
}
func number(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	}
	return 0
}
func truth(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case float64:
		return x != 0
	case int:
		return x != 0
	case int64:
		return x != 0
	}
	return true
}
func first(values ...any) any {
	for _, v := range values {
		if truth(v) {
			return v
		}
	}
	return nil
}
func textOr(v any, fallback string) string {
	if s := str(v); s != "" {
		return s
	}
	return fallback
}
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
func timestamp(v any) float64 {
	if s, ok := v.(string); ok {
		for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05"} {
			if t, e := time.Parse(layout, s); e == nil {
				return float64(t.UnixNano()) / 1e9
			}
		}
		return 0
	}
	n := number(v)
	if n > 1e11 {
		n /= 1000
	}
	return n
}
func iso(v any) any {
	if value, ok := v.(string); ok {
		if t, e := time.Parse(time.RFC3339Nano, value); e == nil {
			return t.UTC().Format(time.RFC3339Nano)
		}
	}
	n := timestamp(v)
	if n == 0 {
		return nil
	}
	seconds, fraction := math.Modf(n)
	return time.Unix(int64(seconds), int64(math.Round(fraction*1e6))*1000).UTC().Format(time.RFC3339Nano)
}

func resolved(p string) string {
	p, _ = filepath.Abs(p)
	if r, e := filepath.EvalSymlinks(p); e == nil {
		return r
	}
	return filepath.Clean(p)
}
func scoped(cwd any, roots map[string]bool) bool {
	s := str(cwd)
	return s != "" && (len(roots) == 0 || roots[resolved(s)])
}
func exists(p string) bool   { _, e := os.Stat(p); return e == nil }
func glob(p string) []string { r, _ := filepath.Glob(p); return r }
func readObject(p string) (Object, error) {
	b, e := os.ReadFile(p)
	if e != nil {
		return nil, e
	}
	var m Object
	e = json.Unmarshal(b, &m)
	if e == nil && m == nil {
		e = fmt.Errorf("expected object")
	}
	return m, e
}
func firstRow(p string) Object {
	f, e := os.Open(p)
	if e != nil {
		return Object{}
	}
	defer f.Close()
	var row Object
	_ = json.NewDecoder(io.LimitReader(f, 4*1024*1024)).Decode(&row)
	return obj(row)
}
func runePrefix(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		r = r[:n]
	}
	return string(r)
}
func runeSuffix(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		r = r[len(r)-n:]
	}
	return string(r)
}
func textContent(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	parts := []string{}
	for _, x := range array(v) {
		if s, ok := obj(x)["text"].(string); ok {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "\n")
}
func cleanText(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	parts := []string{}
	for _, x := range array(v) {
		m := obj(x)
		if contains([]string{"text", "input_text", "output_text"}, str(m["type"])) {
			parts = append(parts, str(m["text"]))
		}
	}
	return strings.Join(parts, "\n")
}

var patterns sync.Map

func rx(pattern string) *regexp.Regexp {
	if p, ok := patterns.Load(pattern); ok {
		return p.(*regexp.Regexp)
	}
	p := regexp.MustCompile(pattern)
	patterns.Store(pattern, p)
	return p
}
func matches(pattern, text string) bool { return rx(pattern).MatchString(text) }

const tailBytes = 4 * 1024 * 1024

func parseRows(data []byte, requireNewline bool) ([]Object, int) {
	rows := []Object{}
	bad := 0
	parts := bytes.Split(data, []byte{'\n'})
	for i, line := range parts {
		if i == len(parts)-1 && len(line) == 0 {
			continue
		}
		if requireNewline && i == len(parts)-1 {
			bad++
			continue
		}
		var row Object
		if json.Unmarshal(line, &row) != nil || row == nil {
			bad++
			continue
		}
		rows = append(rows, row)
	}
	return rows, bad
}
func tailRows(path string, budget int) ([]Object, bool, int, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, false, 0, e
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil {
		return nil, false, 0, e
	}
	start := max(int64(0), st.Size()-int64(budget))
	if _, e = f.Seek(start, io.SeekStart); e != nil {
		return nil, false, 0, e
	}
	b, e := io.ReadAll(io.LimitReader(f, int64(budget)))
	if e != nil {
		return nil, false, 0, e
	}
	if start > 0 {
		if i := bytes.IndexByte(b, '\n'); i >= 0 {
			b = b[i+1:]
		} else {
			b = nil
		}
	}
	rows, bad := parseRows(b, true)
	return rows, start > 0, bad, nil
}
func finding(value, source string, at any, confidence, reason string) Object {
	return Object{"value": value, "source": nullable(source), "at": iso(at), "confidence": confidence, "reason": reason}
}
func unknown(reason string) Object {
	if reason == "" {
		reason = "Not observed"
	}
	return finding("unknown", "", nil, "unknown", reason)
}
func newSession(engine, id, cwd, title, source string, updated any, client string) Object {
	if title == "" {
		title = id
	}
	return Object{"engine": engine, "client": client, "id": id, "app_session_id": nil, "parent_id": nil, "cwd": cwd, "title": title, "updated_at": iso(updated), "source": source, "last_message": Object{"at": nil, "role": nil, "excerpt": nil}, "activity": unknown(""), "question": unknown(""), "permission": unknown(""), "process": unknown(""), "spec_association": unknown("Folder match only; explicit spec ownership not established"), "pending_questions": []Object{}, "last_turn_status": nil, "unknown_fields": []string{}, "warnings": []string{}}
}
func warn(s Object, msg string) { a := array(s["warnings"]); s["warnings"] = append(a, msg) }
func tailWarnings(s Object, path string, truncated bool, bad int) {
	if truncated {
		warn(s, "Bounded tail of "+filepath.Base(path)+"; older unmatched requests may be missing")
	}
	if bad > 0 {
		warn(s, fmt.Sprintf("Skipped %d malformed/incomplete records in %s", bad, filepath.Base(path)))
	}
}
func message(s Object, role string, at, content any) {
	text := cleanText(content)
	if strings.TrimSpace(text) == "" {
		return
	}
	if len([]rune(text)) > 400 {
		text = runePrefix(text, 240) + " … " + runeSuffix(text, 160)
	}
	s["last_message"] = Object{"at": iso(at), "role": role, "excerpt": text}
}
func questionCall(name, id string, args, at any) Object {
	parts := strings.Split(name, ".")
	short := parts[len(parts)-1]
	if !contains([]string{"request_user_input", "request_user_input_async", "AskUserQuestion", "ask_user_question"}, short) {
		return nil
	}
	if s, ok := args.(string); ok {
		var decoded any
		_ = json.Unmarshal([]byte(s), &decoded)
		args = decoded
	}
	prompts := []string{}
	for _, v := range array(obj(args)["questions"]) {
		q := obj(v)
		if len(q) > 0 {
			prompts = append(prompts, runePrefix(textOr(q["question"], str(q["title"])), 1000))
		}
	}
	return Object{"call_id": id, "tool": name, "at": iso(at), "prompts": prompts, "blocking": short != "request_user_input_async"}
}
func finishQuestion(s Object, pending map[string]Object, source string, at any, live, terminal, complete bool) {
	qs := []Object{}
	keys := []string{}
	for k := range pending {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	blocking, async := false, false
	for _, k := range keys {
		q := pending[k]
		qs = append(qs, q)
		if truth(q["blocking"]) {
			blocking = true
		} else {
			async = true
		}
	}
	s["pending_questions"] = qs
	if len(qs) > 0 {
		value, confidence, reason := "unknown", "inferred", "Unresolved structured question; current wait is unverified"
		if live {
			value, confidence, reason = "yes", "observed", "Unresolved structured question and live session evidence"
		}
		s["question"] = finding(value, source, at, confidence, reason)
		if async {
			s["question"] = finding("unknown", source, at, "inferred", "Async question request found; immediate tool return does not establish whether it was answered")
		}
		if live && blocking {
			s["activity"] = finding("waiting", source, at, "inferred", "Unresolved blocking question in a live session")
		}
		return
	}
	last := obj(s["last_message"])
	if str(last["role"]) == "assistant" && matches(`(?i)\?|please confirm|let me know|your approval`, str(last["excerpt"])) {
		s["question"] = finding("unknown", source, last["at"], "inferred", "Latest assistant text may ask a question; caller should read it")
	} else if terminal || (complete && live) {
		s["question"] = finding("no", source, at, "inferred", "No unresolved structured question or question cue in the latest message")
	}
}
func selectRecent(sessions []Object, limit int, extra map[string]bool) []Object {
	sort.SliceStable(sessions, func(i, j int) bool {
		a, b := timestamp(sessions[i]["updated_at"]), timestamp(sessions[j]["updated_at"])
		if a != b {
			return a > b
		}
		return str(sessions[i]["id"]) > str(sessions[j]["id"])
	})
	counts := map[string]int{}
	out := []Object{}
	for _, s := range sessions {
		k := str(s["cwd"])
		if limit == 0 || counts[k] < limit || extra[str(s["id"])] {
			out = append(out, s)
		}
		counts[k]++
	}
	return out
}

// URI mode=ro prevents writes; query_only is applied to every pooled connection.
// Do not use immutable: live app databases may have committed data in their WAL.
func openReadDB(path string) (*sql.DB, error) {
	u := url.URL{Scheme: "file", Path: resolved(path)}
	q := u.Query()
	q.Set("mode", "ro")
	q.Add("_pragma", "query_only(1)")
	q.Add("_pragma", "busy_timeout(1000)")
	u.RawQuery = q.Encode()
	db, e := sql.Open("sqlite", u.String())
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if e = db.PingContext(ctx); e != nil {
		db.Close()
		return nil, e
	}
	return db, nil
}
func queryObjects(db *sql.DB, query string, args ...any) ([]Object, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rows, e := db.QueryContext(ctx, query, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	cols, e := rows.Columns()
	if e != nil {
		return nil, e
	}
	result := []Object{}
	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if e = rows.Scan(ptrs...); e != nil {
			return nil, e
		}
		m := Object{}
		for i, c := range cols {
			if b, ok := values[i].([]byte); ok {
				values[i] = string(b)
			}
			m[c] = values[i]
		}
		result = append(result, m)
	}
	return result, rows.Err()
}
func envPath(key, fallback string) string {
	if s := os.Getenv(key); s != "" {
		return s
	}
	return fallback
}
func intString(n any) string { return strconv.Itoa(int(number(n))) }
