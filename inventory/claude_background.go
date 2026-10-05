package inventory

import "strings"

// Claude may finish its chat turn while background Bash/Agent tasks still run.
// Completion notifications can be consumed mid-turn and appear only in the
// queue journal, without a corresponding user message.
func claudeBackground(st *transcriptState, row Object) {
	result := obj(row["toolUseResult"])
	id := str(result["backgroundTaskId"])
	if str(result["status"]) == "async_launched" {
		id = str(result["agentId"])
	}
	if id != "" {
		st.background[id] = row["timestamp"]
	}
	content := obj(row["message"])["content"]
	if str(row["type"]) == "queue-operation" {
		content = row["content"]
	} else if str(row["type"]) != "user" {
		return
	}
	for _, block := range rx(`(?s)<task-notification>(.*?)</task-notification>`).FindAllStringSubmatch(cleanText(content), -1) {
		id := rx(`<task-id>([^<]+)</task-id>`).FindStringSubmatch(block[1])
		status := rx(`<status>([^<]+)</status>`).FindStringSubmatch(block[1])
		if len(id) == 2 && len(status) == 2 && contains([]string{"completed", "failed", "killed", "stopped", "cancelled"}, strings.TrimSpace(status[1])) {
			delete(st.background, strings.TrimSpace(id[1]))
		}
	}
}
