package workflow

// PinObserved records the original workflow separately while making setup requests exact.
// Multiple setup actions of the same language are left untouched to avoid misattribution.
func PinObserved(w *Workflow, jobID string, versions map[string]string) {
	jobs, ok := w.Raw["jobs"].(map[string]any)
	if !ok {
		return
	}
	job, ok := jobs[jobID].(map[string]any)
	if !ok {
		return
	}
	steps, ok := job["steps"].([]any)
	if !ok {
		return
	}
	actions := map[string]string{"astral-sh/setup-uv@": "uv-version", "actions/setup-node@": "node-version", "actions/setup-python@": "python-version", "actions/setup-go@": "go-version"}
	for prefix, key := range actions {
		var chosen map[string]any
		count := 0
		for _, value := range steps {
			s, ok := value.(map[string]any)
			if !ok {
				continue
			}
			uses, _ := s["uses"].(string)
			if len(uses) >= len(prefix) && uses[:len(prefix)] == prefix {
				count++
				chosen = s
			}
		}
		if count != 1 || versions[key] == "" {
			continue
		}
		with, ok := chosen["with"].(map[string]any)
		if !ok {
			with = map[string]any{}
			chosen["with"] = with
		}
		inputKey := key
		if key == "uv-version" {
			inputKey = "version"
		}
		with[inputKey] = versions[key]
		delete(with, key+"-file")
		delete(with, "check-latest")
	}
}
