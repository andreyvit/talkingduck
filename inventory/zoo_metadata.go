package inventory

import (
	"strconv"
	"strings"
)

// Deliberately a restricted frontmatter reader, not a general YAML parser.
// Unsupported or duplicate values fall back to legacy inference.
func zooFrontmatter(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	m := rx(`\A---\n([\s\S]*?)\n---(?:\n|$)`).FindStringSubmatch(text)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}
func zooField(text, key string) (string, bool) {
	ms := rx(`(?m)^`+key+`:[ \t]*([^\n]*)$`).FindAllStringSubmatch(zooFrontmatter(text), -1)
	if len(ms) != 1 {
		return "", false
	}
	v := strings.TrimSpace(strings.SplitN(ms[0][1], " #", 2)[0])
	return strings.Trim(v, `"'`), true
}
func zooTickets(text string) ([]string, bool) {
	v, ok := zooField(text, "tickets")
	if !ok {
		return nil, false
	}
	if strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") {
		v = strings.TrimSpace(v[1 : len(v)-1])
		if v == "" {
			return []string{}, true
		}
		out := []string{}
		for _, item := range strings.Split(v, ",") {
			item = strings.Trim(strings.TrimSpace(item), `"'`)
			if !matches(`^[A-Z][A-Z0-9]+-[0-9]+$`, item) {
				return nil, false
			}
			out = append(out, item)
		}
		return unique(out), true
	}
	if v != "" {
		return nil, false
	}
	m := rx(`(?m)^tickets:[ \t]*\n((?:[ \t]+-[ \t]+[^\n]+\n?)+)`).FindStringSubmatch(zooFrontmatter(text))
	if len(m) != 2 {
		return nil, false
	}
	out := []string{}
	for _, line := range strings.Split(strings.TrimSpace(m[1]), "\n") {
		item := strings.Trim(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-")), `"'`)
		if !matches(`^[A-Z][A-Z0-9]+-[0-9]+$`, item) {
			return nil, false
		}
		out = append(out, item)
	}
	return unique(out), true
}
func applyZooMetadata(w Object, text string) {
	if v, ok := zooField(text, "revision_round"); ok && matches(`^[0-9]+$`, v) {
		if n, e := strconv.Atoi(v); e == nil {
			w["round"], w["round_confidence"], w["round_evidence"] = n, "explicit", []string{"frontmatter revision_round"}
			w["squashed_this_round"] = false
		}
	}
	if v, ok := zooField(text, "stage"); ok && contains([]string{"INV", "HL", "LL", "EX", "FIN"}, v) {
		w["stage"], w["stage_confidence"] = v, "explicit"
	}
	if v, ok := zooField(text, "stage_complete"); ok && (v == "true" || v == "false") {
		w["stage_complete"], w["progress"] = v == "true", "?"
		if v == "true" {
			w["progress"] = "✓"
		}
	}
	if str(w["round_confidence"]) == "explicit" || str(w["stage_confidence"]) == "explicit" {
		w["reason"] = "Valid Zoo frontmatter overrides prose inference; activity remains independently observed"
	}
}
