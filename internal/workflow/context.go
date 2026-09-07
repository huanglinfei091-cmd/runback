package workflow

type ExecutionContext struct {
	Shell            string
	WorkingDirectory string
	Environment      map[string]any
}

// Execution merges workflow/job/step settings without evaluating secret or event expressions.
func Execution(w Workflow, jobID string, index int) ExecutionContext {
	c := ExecutionContext{Shell: "bash", WorkingDirectory: ".", Environment: map[string]any{}}
	merge := func(m map[string]any) {
		if env, ok := m["env"].(map[string]any); ok {
			for k, v := range env {
				c.Environment[k] = v
			}
		}
		if d, ok := m["defaults"].(map[string]any); ok {
			if r, ok := d["run"].(map[string]any); ok {
				if s, ok := r["shell"].(string); ok {
					c.Shell = s
				}
				if s, ok := r["working-directory"].(string); ok {
					c.WorkingDirectory = s
				}
			}
		}
		if s, ok := m["shell"].(string); ok {
			c.Shell = s
		}
		if s, ok := m["working-directory"].(string); ok {
			c.WorkingDirectory = s
		}
	}
	merge(w.Raw)
	jobs, _ := w.Raw["jobs"].(map[string]any)
	j, _ := jobs[jobID].(map[string]any)
	merge(j)
	if steps, ok := j["steps"].([]any); ok && index >= 0 && index < len(steps) {
		if s, ok := steps[index].(map[string]any); ok {
			merge(s)
		}
	}
	return c
}
