// Inventory reads local Git evidence. It never fetches or changes a checkout.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type AgentSession struct {
	LastMessage struct {
		At *string `json:"at"`
	} `json:"last_message"`
	PendingQuestions []struct {
		Prompts []string `json:"prompts"`
	} `json:"pending_questions"`
	Engine   string       `json:"engine"`
	ID       string       `json:"id"`
	Cwd      string       `json:"cwd"`
	Title    string       `json:"title"`
	Activity AgentFinding `json:"activity"`
	Question AgentFinding `json:"question"`
}
type AgentFinding struct {
	Value string `json:"value"`
}
type AgentReport struct {
	Sessions    []json.RawMessage `json:"sessions"`
	Sources     []json.RawMessage `json:"sources"`
	Limitations []string          `json:"limitations"`
	ObservedAt  string            `json:"observed_at"`
	Error       string            `json:"error,omitempty"`
}

type Commit struct {
	Hash    string    `json:"hash"`
	Subject string    `json:"subject"`
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
	Paths   []string  `json:"paths"`
}
type CommitCandidate struct {
	Spec         string   `json:"spec"`
	MatchedPaths []string `json:"matched_paths"`
}
type CommitAssociationIssue struct {
	Hash       string            `json:"hash"`
	Subject    string            `json:"subject"`
	Reason     string            `json:"reason"`
	Candidates []CommitCandidate `json:"candidates"`
}
type Evidence struct {
	Kind     string `json:"kind"`
	Source   string `json:"source"`
	Inferred bool   `json:"inferred"`
}
type Change struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

type Spec struct {
	GitStatus        string     `json:"git_status,omitempty"`
	MentionedTickets []string   `json:"mentioned_tickets"`
	Path             string     `json:"path"`
	Title            string     `json:"title"`
	Tickets          []string   `json:"tickets"`
	Status           string     `json:"status"`
	Stage            string     `json:"stage"`
	Dirty            bool       `json:"dirty"`
	Deleted          bool       `json:"deleted"`
	LastActivity     time.Time  `json:"last_activity"`
	ActivitySource   string     `json:"activity_source"`
	Commits          []string   `json:"commits"`
	DirtyFiles       []string   `json:"dirty_files"`
	Evidence         []Evidence `json:"evidence"`
	body             string
	rank             int
	commitOrder      int
}
type Repo struct {
	CommitAssociationIssues []CommitAssociationIssue `json:"commit_association_issues"`
	Agents                  []json.RawMessage        `json:"agents"`
	LocalChanges            []Change                 `json:"local_changes"`
	Path                    string                   `json:"path"`
	Branch                  string                   `json:"branch"`
	Head                    string                   `json:"head"`
	AgentState              string                   `json:"agent_state"`
	Specs                   []*Spec                  `json:"specs"`
	Commits                 []Commit                 `json:"unpushed_commits"`
	UnassignedCommits       []string                 `json:"unassigned_commits"`
	UnassignedFiles         []string                 `json:"unassigned_files"`
	AuxiliaryFiles          []string                 `json:"auxiliary_files"`
	Warnings                []string                 `json:"warnings"`
	Error                   string                   `json:"error,omitempty"`
}

// Commits listed in the spec report are stronger evidence than shared code paths.
// Resolve old hashes by subject after rebases; retain an explicit inference label.
func reportedCommits(dir, body string, cache map[string]string) (hashes, subjects []string) {
	level := 0
	span := regexp.MustCompile("`([^`]+)`")
	hashRE := regexp.MustCompile(`^[0-9a-f]{7,40}$`)
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "#") {
			n := len(line) - len(strings.TrimLeft(line, "#"))
			heading := strings.TrimSpace(line[n:])
			if level > 0 && n <= level {
				level = 0
			}
			if strings.EqualFold(heading, "Commits") {
				level = n
			}
			continue
		}
		if level == 0 {
			continue
		}
		parts := span.FindAllStringSubmatch(line, -1)
		if len(parts) == 0 || !hashRE.MatchString(parts[0][1]) {
			continue
		}
		hash := parts[0][1]
		hashes = append(hashes, hash)
		if len(parts) > 1 && !hashRE.MatchString(parts[1][1]) {
			subjects = append(subjects, parts[1][1])
		}
		value, ok := cache[hash]
		if !ok {
			value, _ = git(dir, "show", "-s", "--format=%s", hash+"^{commit}")
			value = strings.TrimSpace(value)
			cache[hash] = value
		}
		if value != "" {
			subjects = append(subjects, value)
		}
	}
	return unique(hashes), unique(subjects)
}

type Report struct {
	AgentSources     []json.RawMessage `json:"agent_sources"`
	AgentLimitations []string          `json:"agent_limitations"`
	AgentError       string            `json:"agent_error,omitempty"`
	ObservedAt       time.Time         `json:"observed_at"`
	Boundary         string            `json:"boundary"`
	Repositories     []Repo            `json:"repositories"`
}

func git(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	b, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(b)))
	}
	return string(b), nil
}
func lines(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return strings.Split(strings.TrimSpace(s), "\n")
}
func specPath(p string) bool {
	name := strings.ToLower(filepath.Base(p))
	return strings.HasPrefix(p, ".spec/") && strings.HasSuffix(p, ".md") && name != "agents.md" && name != "claude.md" && name != "readme.md"
}
func auxiliary(p string) bool {
	return p == ".ai-mistakes.md" || p == "_ai/MISTAKES.md" || strings.HasPrefix(p, ".proposals/") || strings.HasPrefix(p, ".tasks/")
}

var ticketRE = regexp.MustCompile(`\b[A-Z][A-Z0-9]+-[0-9]+\b`)
var statusRE = regexp.MustCompile(`(?m)^status:\s*([^\r\n]+)`)

func tickets(s string) []string { return unique(ticketRE.FindAllString(strings.ToUpper(s), -1)) }
func unique(xs []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}
func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
func stage(status string) string {
	s := strings.ToLower(status)
	if s == "done" || s == "complete" || s == "completed" {
		return "done"
	}
	if strings.Contains(s, "defer") || strings.Contains(s, "paused") || strings.Contains(s, "blocked") || strings.Contains(s, "waiting for") {
		return "parked"
	}
	if strings.Contains(s, "pending suggestions") || strings.Contains(s, "awaiting decision") {
		return "needs decision"
	}
	if strings.Contains(s, "review") {
		return "review"
	}
	if strings.Contains(s, "subtask") || strings.Contains(s, "execut") || strings.Contains(s, "implement") {
		return "execution"
	}
	if strings.Contains(s, "plan") {
		return "planning"
	}
	return "unknown"
}
func parseSpec(path, body string) *Spec {
	s := &Spec{commitOrder: 1 << 30, Path: path, Title: strings.TrimSuffix(filepath.Base(path), ".md"), Status: "unknown", body: body, Tickets: []string{}, Commits: []string{}, DirtyFiles: []string{}, Evidence: []Evidence{}}
	if strings.HasPrefix(body, "---\n") {
		if end := strings.Index(body[4:], "\n---"); end >= 0 {
			if m := statusRE.FindStringSubmatch(body[4 : 4+end]); m != nil {
				s.Status = strings.Trim(m[1], ` "'`)
			}
		}
	}
	// Only the request and filename identify tickets; later sections often cite unrelated work.
	request := ""
	inRequest := false
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "# ") {
			s.Title = strings.TrimPrefix(line, "# ")
			break
		}
	}
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "## ") {
			if inRequest {
				break
			}
			h := strings.ToLower(strings.TrimSpace(line[3:]))
			inRequest = h == "request" || h == "user request" || h == "user input"
			continue
		}
		if inRequest {
			request += line + "\n"
		}
	}
	// Filename + first request paragraph + explicit additions avoid background tickets.
	primary := filepath.Base(path) + "\n" + strings.Split(strings.TrimSpace(request), "\n\n")[0]
	for _, line := range strings.Split(request, "\n") {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "also include ") || strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "ticket:") || strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "tickets:") {
			primary += "\n" + line
		}
	}
	s.Tickets = tickets(primary)
	s.MentionedTickets = tickets(request)
	s.Evidence = append(s.Evidence, Evidence{"tickets inferred from filename and request", path, true})
	s.Stage = stage(s.Status)
	return s
}

// Porcelain -z puts a rename destination first, then its source. Preserve both.
func dirtyPaths(raw string) []string {
	fields := strings.Split(raw, "\x00")
	out := []string{}
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		if len(f) < 4 {
			continue
		}
		out = append(out, f[3:])
		if strings.ContainsAny(f[:2], "RC") && i+1 < len(fields) {
			i++
			out = append(out, fields[i])
		}
	}
	return unique(out)
}
func touch(s *Spec, t time.Time, source string) {
	if t.After(s.LastActivity) {
		s.LastActivity = t
		s.ActivitySource = source
	}
}
func overlap(s *Spec, paths []string) []string {
	out := []string{}
	for _, p := range paths {
		if specPath(p) || auxiliary(p) {
			continue
		}
		match := strings.Contains(s.body, p)
		for parent := filepath.ToSlash(filepath.Dir(p)); parent != "." && strings.Contains(parent, "/"); parent = filepath.ToSlash(filepath.Dir(parent)) {
			match = match || strings.Contains(s.body, "`"+parent+"`") || strings.Contains(s.body, "`"+parent+"/`")
		}
		if match {
			out = append(out, p)
		}
	}
	return out
}
func scan(dir string, now time.Time) (r Repo) {
	r = Repo{Path: dir, AgentState: "unknown (see individual sessions)", Specs: []*Spec{}, Commits: []Commit{}, UnassignedCommits: []string{}, UnassignedFiles: []string{}, AuxiliaryFiles: []string{}, Warnings: []string{}}
	head, err := git(dir, "rev-parse", "--verify", "HEAD")
	if err != nil {
		r.Error = err.Error()
		return
	}
	r.Head = strings.TrimSpace(head)
	branch, _ := git(dir, "symbolic-ref", "--quiet", "--short", "HEAD")
	r.Branch = strings.TrimSpace(branch)
	if r.Branch == "" {
		r.Branch = "(detached)"
	}
	raw, err := git(dir, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		r.Error = err.Error()
		return
	}
	dirty := dirtyPaths(raw)
	r.LocalChanges = []Change{}
	fields := strings.Split(raw, "\x00")
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		if len(f) < 4 {
			continue
		}
		r.LocalChanges = append(r.LocalChanges, Change{f[3:], f[:2]})
		if strings.ContainsAny(f[:2], "RC") && i+1 < len(fields) {
			i++
			r.LocalChanges = append(r.LocalChanges, Change{fields[i], "rename/copy source"})
		}
	}
	refs, err := git(dir, "for-each-ref", "--format=%(refname)", "refs/remotes/")
	if err != nil {
		r.Error = err.Error()
		return
	}
	if strings.TrimSpace(refs) == "" {
		r.Warnings = append(r.Warnings, "No remote-tracking refs: unpushed history is unknown; only dirty/untracked specs collected.")
	} else {
		// Compare against every known remote branch, not just upstream. Pushed feature work is history too.
		log, err := git(dir, "log", "--topo-order", "--format=%H%x00%cI%x00%s%x00%b%x1e", "HEAD", "--not", "--remotes")
		if err != nil {
			r.Error = err.Error()
			return
		}
		for _, line := range strings.Split(log, "\x1e") {
			v := strings.SplitN(strings.TrimSpace(line), "\x00", 4)
			if len(v) != 4 {
				continue
			}
			t, _ := time.Parse(time.RFC3339, v[1])
			names, e := git(dir, "diff-tree", "--root", "--no-commit-id", "--name-only", "-r", "-m", "--no-renames", "-z", v[0])
			if e != nil {
				r.Error = e.Error()
				return
			}
			r.Commits = append(r.Commits, Commit{Hash: v[0], Subject: v[2], Message: v[2] + "\n\n" + v[3], Time: t, Paths: unique(strings.FieldsFunc(names, func(c rune) bool { return c == 0 }))})
		}
	}
	all := map[string]*Spec{}
	active := map[string]*Spec{}
	// Read tracked and untracked files only, never ignored private spec files.
	files, err := git(dir, "ls-files", "--cached", "--others", "--exclude-standard", "-z", "--", ".spec/")
	if err != nil {
		r.Error = err.Error()
		return
	}
	for _, p := range unique(strings.Split(files, "\x00")) {
		if !specPath(p) {
			continue
		}
		b, e := os.ReadFile(filepath.Join(dir, p))
		if e == nil {
			all[p] = parseSpec(p, string(b))
		}
	}
	get := func(p string) *Spec {
		if s := all[p]; s != nil {
			return s
		}
		body, _ := git(dir, "show", "HEAD:"+p)
		s := parseSpec(p, body)
		s.Deleted = true
		all[p] = s
		return s
	}
	for _, p := range dirty {
		if specPath(p) {
			s := get(p)
			s.Dirty = true
			for _, c := range r.LocalChanges {
				if c.Path == p {
					s.GitStatus = c.Status
				}
			}
			s.Evidence = append(s.Evidence, Evidence{"spec changed", p, false})
			if st, e := os.Stat(filepath.Join(dir, p)); e == nil {
				touch(s, st.ModTime(), "spec filesystem mtime (approximate)")
			}
			active[p] = s
		}
	}
	for _, c := range r.Commits {
		for _, p := range c.Paths {
			if specPath(p) {
				s := get(p)
				active[p] = s
			}
		}
	}
	// Explicit commit references can discover an unchanged spec. Ticket-only links require one match.
	for _, c := range r.Commits {
		ct := tickets(c.Message)
		for _, s := range all {
			if strings.Contains(c.Message, s.Path) {
				active[s.Path] = s
			}
		}
		for _, t := range ct {
			matches := []*Spec{}
			for _, s := range all {
				if contains(s.Tickets, t) {
					matches = append(matches, s)
				}
			}
			if len(matches) == 1 {
				active[matches[0].Path] = matches[0]
			}
		}
	}
	// A pushed spec can still describe current local code edits. Require a unique
	// path match, and prefer already-selected specs before searching historical ones.
	for _, p := range dirty {
		if specPath(p) || auxiliary(p) {
			continue
		}
		existing := false
		for _, s := range active {
			if len(overlap(s, []string{p})) > 0 {
				existing = true
			}
		}
		if existing {
			continue
		}
		matches := []*Spec{}
		for _, s := range all {
			if len(overlap(s, []string{p})) > 0 {
				matches = append(matches, s)
			}
		}
		if len(matches) == 1 {
			active[matches[0].Path] = matches[0]
		}
	}
	// Attach each commit by strongest evidence, retaining multiple direct owners.
	hashRefs, subjectRefs := map[string][]string{}, map[string][]string{}
	cache, subjectCounts := map[string]string{}, map[string]int{}
	for _, c := range r.Commits {
		subjectCounts[c.Subject]++
	}
	for path, s := range active {
		hashRefs[path], subjectRefs[path] = reportedCommits(dir, s.body, cache)
	}
	for ci, c := range r.Commits {
		owners := []*Spec{}
		ticketOwners := []*Spec{}
		commitTickets := tickets(c.Message)
		for _, s := range active {
			for _, ticket := range commitTickets {
				if contains(s.Tickets, ticket) {
					ticketOwners = append(ticketOwners, s)
					break
				}
			}
		}
		kind := "spec in commit"
		inferred := false
		for _, s := range active {
			if contains(c.Paths, s.Path) || strings.Contains(c.Message, s.Path) {
				owners = append(owners, s)
			}
		}
		if len(owners) == 0 && len(ticketOwners) == 1 {
			owners = ticketOwners
			kind, inferred = "unique spec ticket in full commit message", true
		}
		if len(owners) == 0 {
			kind = "commit hash listed in spec report"
			for path, s := range active {
				for _, hash := range hashRefs[path] {
					if strings.HasPrefix(c.Hash, hash) {
						owners = append(owners, s)
						break
					}
				}
			}
		}
		if len(owners) == 0 && subjectCounts[c.Subject] == 1 {
			kind, inferred = "unique commit subject from spec's historical commit list (rebase-tolerant)", true
			for path, s := range active {
				if contains(subjectRefs[path], c.Subject) {
					owners = append(owners, s)
				}
			}
		}
		if len(owners) == 0 {
			kind = "ticket in commit message"
			inferred = true
			owners = ticketOwners
		}
		if len(owners) == 0 {
			kind = "file or directory path mentioned in spec"
			inferred = true
			for _, s := range active {
				if len(overlap(s, c.Paths)) > 0 {
					owners = append(owners, s)
				}
			}
		}
		if inferred && len(owners) > 1 {
			issue := CommitAssociationIssue{Hash: c.Hash, Subject: c.Subject, Reason: kind + " matches multiple specs; shared context does not establish task ownership", Candidates: []CommitCandidate{}}
			for _, s := range owners {
				issue.Candidates = append(issue.Candidates, CommitCandidate{s.Path, overlap(s, c.Paths)})
			}
			sort.Slice(issue.Candidates, func(i, j int) bool { return issue.Candidates[i].Spec < issue.Candidates[j].Spec })
			r.CommitAssociationIssues = append(r.CommitAssociationIssues, issue)
			owners = nil
		}
		if len(owners) == 0 {
			r.UnassignedCommits = append(r.UnassignedCommits, c.Hash)
			continue
		}
		for _, s := range owners {
			s.Commits = append(s.Commits, c.Hash)
			if ci < s.commitOrder {
				s.commitOrder = ci
			}
			s.Evidence = append(s.Evidence, Evidence{kind, c.Hash, inferred})
			for _, ticket := range commitTickets {
				if contains(s.Tickets, ticket) {
					s.Evidence = append(s.Evidence, Evidence{"commit message and spec share ticket " + ticket, c.Hash, true})
				}
			}
			touch(s, c.Time, "commit "+c.Hash[:10])
		}
	}
	for _, p := range dirty {
		if specPath(p) {
			continue
		}
		if auxiliary(p) {
			r.AuxiliaryFiles = append(r.AuxiliaryFiles, p)
			continue
		}
		owners := []*Spec{}
		for _, s := range active {
			matches := len(overlap(s, []string{p})) > 0
			for _, c := range r.Commits {
				if contains(s.Commits, c.Hash) && contains(c.Paths, p) {
					matches = true
				}
			}
			if matches {
				owners = append(owners, s)
			}
		}
		if len(owners) != 1 {
			r.UnassignedFiles = append(r.UnassignedFiles, p)
			continue
		}
		s := owners[0]
		s.DirtyFiles = append(s.DirtyFiles, p)
		s.Evidence = append(s.Evidence, Evidence{"dirty path matches spec or associated commit", p, true})
		if st, e := os.Stat(filepath.Join(dir, p)); e == nil {
			touch(s, st.ModTime(), "dirty file mtime (approximate)")
		}
	}
	for _, s := range active {
		switch {
		case s.Stage == "done":
			s.rank = 4
		case s.Stage == "parked":
			s.rank = 3
		case s.LastActivity.After(now.Add(-7 * 24 * time.Hour)):
			s.rank = 0
		case len(s.Commits) > 0 || len(s.DirtyFiles) > 0 || s.Stage == "execution":
			s.rank = 1
		default:
			s.rank = 2
		}
		r.Specs = append(r.Specs, s)
	}
	sort.Slice(r.Specs, func(i, j int) bool {
		a, b := r.Specs[i], r.Specs[j]
		if a.rank != b.rank {
			return a.rank < b.rank
		}
		if !a.Dirty && !b.Dirty && len(a.DirtyFiles) == 0 && len(b.DirtyFiles) == 0 && len(a.Commits) > 0 && len(b.Commits) > 0 && a.commitOrder != b.commitOrder {
			return a.commitOrder < b.commitOrder
		}
		if !a.LastActivity.Equal(b.LastActivity) {
			return a.LastActivity.After(b.LastActivity)
		}
		return a.Path < b.Path
	})
	// Detect writes during the scan rather than promise an atomic snapshot.
	end, _ := git(dir, "rev-parse", "HEAD")
	status, _ := git(dir, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if strings.TrimSpace(end) != r.Head || status != raw {
		r.Warnings = append(r.Warnings, "Checkout changed during scan; rerun for a fresh observation.")
	}
	return
}
func printReport(report Report) {
	fmt.Printf("Local work inventory — %s\n%s\n", report.ObservedAt.Format(time.RFC3339), report.Boundary)
	if report.AgentError != "" {
		fmt.Println("Agent collection:", report.AgentError)
	}
	for _, raw := range report.AgentSources {
		var source struct {
			Engine   string   `json:"engine"`
			Status   string   `json:"status"`
			Warnings []string `json:"warnings"`
		}
		if json.Unmarshal(raw, &source) == nil {
			if source.Status != "available" {
				fmt.Printf("Agent source %s: %s\n", source.Engine, source.Status)
			}
			for _, w := range source.Warnings {
				fmt.Printf("  %s: %s\n", source.Engine, w)
			}
		}
	}
	for _, r := range report.Repositories {
		fmt.Printf("\n%s [%s] — %d unpushed commits\n  %s\n", filepath.Base(r.Path), r.Branch, len(r.Commits), r.Path)
		if r.Error != "" {
			fmt.Println("  ERROR:", r.Error)
			continue
		}
		for _, raw := range r.Agents {
			var a AgentSession
			if json.Unmarshal(raw, &a) == nil {
				fmt.Printf("  Agent %s | %s | activity: %s | question: %s\n", a.Engine, a.Title, a.Activity.Value, a.Question.Value)
				if a.LastMessage.At != nil {
					fmt.Println("    Last text message:", *a.LastMessage.At)
				}
				for _, q := range a.PendingQuestions {
					for _, prompt := range q.Prompts {
						fmt.Println("    Question:", prompt)
					}
				}
			}
		}
		if len(r.Specs) == 0 {
			fmt.Println("  No specs with local work found.")
		}
		for _, s := range r.Specs {
			ts := strings.Join(s.Tickets, ", ")
			if ts == "" {
				ts = "no ticket found"
			}
			activity := "unknown"
			if !s.LastActivity.IsZero() {
				activity = s.LastActivity.Local().Format("2006-01-02 15:04")
			}
			fmt.Printf("  %s | %s | %s\n    %s\n    status: %s\n    %d commits; spec dirty: %t; %d associated dirty files; last: %s\n", ts, s.Stage, s.Title, s.Path, s.Status, len(s.Commits), s.Dirty, len(s.DirtyFiles), activity)
			if s.GitStatus != "" {
				fmt.Printf("    Git status: %q (?? = untracked)\n", s.GitStatus)
			}
			if s.Deleted {
				fmt.Println("    spec deleted; metadata from HEAD when available")
			}
			inferred := false
			for _, e := range s.Evidence {
				inferred = inferred || (e.Inferred && e.Kind != "tickets inferred from filename and request")
			}
			if inferred {
				fmt.Println("    includes inferred file/ticket associations; see -json for evidence")
			}
		}
		if len(r.UnassignedCommits) > 0 {
			fmt.Printf("  Unassigned commits: %d\n", len(r.UnassignedCommits))
			for _, c := range r.Commits {
				if contains(r.UnassignedCommits, c.Hash) {
					fmt.Printf("    %s %s\n", c.Hash[:10], c.Subject)
				}
			}
		}
		for _, p := range r.UnassignedFiles {
			fmt.Println("  Unassigned file:", p)
		}
		for _, p := range r.AuxiliaryFiles {
			fmt.Println("  Auxiliary file (no activity boost):", p)
		}
		for _, w := range r.Warnings {
			fmt.Println("  WARNING:", w)
		}
	}
}
func main() {
	home, _ := os.UserHomeDir()
	root := flag.String("root", filepath.Join(home, "Developer", "prairie"), "parent of backend and back2 through back10")
	noAgents := flag.Bool("no-agents", false, "skip local agent session collection")
	agentLimit := flag.Int("agent-limit", 0, "recent sessions per provider and folder; 0 = all")
	asJSON := flag.Bool("json", false, "emit structured observations and association evidence")
	details := flag.Bool("details", false, "legacy detailed worktree/session listing")
	width := flag.Int("width", 0, "table width; 0 uses COLUMNS or 120")
	agentsOnly := flag.Bool("agents-only", false, "emit raw agent observations as JSON without scanning Git")
	agentHome := flag.String("home", home, "home directory for agents-only data discovery")
	flag.Parse()
	if *agentLimit < 0 {
		fmt.Fprintln(os.Stderr, "-agent-limit must be nonnegative")
		os.Exit(2)
	}
	paths := flag.Args()
	if *agentsOnly {
		data := collectAgentData(*agentHome, paths, *agentLimit, float64(time.Now().UnixNano())/1e9, probeProcess)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(data); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if len(paths) == 0 {
		paths = append(paths, filepath.Join(*root, "backend"))
		for i := 2; i <= 10; i++ {
			paths = append(paths, filepath.Join(*root, fmt.Sprintf("back%d", i)))
		}
	}
	report := Report{ObservedAt: time.Now().UTC(), Boundary: "Unpushed = HEAD commits absent from local remote-tracking refs. No fetch; remote knowledge may be stale. Agent states include uncertainty; see JSON evidence.", Repositories: []Repo{}}
	failed := false
	for _, p := range paths {
		p, e := filepath.Abs(p)
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
		r := scan(p, report.ObservedAt)
		failed = failed || r.Error != ""
		report.Repositories = append(report.Repositories, r)
	}
	if !*noAgents {
		roots := []string{}
		for _, r := range report.Repositories {
			roots = append(roots, r.Path)
		}
		agents := collectAgents(roots, *agentLimit)
		report.AgentSources = agents.Sources
		report.AgentLimitations = agents.Limitations
		report.AgentError = agents.Error
		for _, raw := range agents.Sessions {
			var a AgentSession
			if json.Unmarshal(raw, &a) != nil {
				continue
			}
			cwd := filepath.Clean(a.Cwd)
			if resolved, e := filepath.EvalSymlinks(cwd); e == nil {
				cwd = resolved
			}
			for i := range report.Repositories {
				root := report.Repositories[i].Path
				if resolved, e := filepath.EvalSymlinks(root); e == nil {
					root = resolved
				}
				if root == cwd {
					report.Repositories[i].Agents = append(report.Repositories[i].Agents, raw)
				}
			}
		}
	}
	if !*details {
		if err := printOverview(report, *asJSON, *width); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		printReport(report)
	}
	if failed {
		os.Exit(1)
	}
}
