// Package workflow resolves static matrices without pretending to evaluate all Actions expressions.
package workflow

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Workflow struct {
	Name string         `yaml:"name"`
	Jobs map[string]Job `yaml:"jobs"`
	Raw  map[string]any
}
type Job struct {
	Name     string `yaml:"name"`
	RunsOn   any    `yaml:"runs-on"`
	Strategy struct {
		Matrix any `yaml:"matrix"`
	} `yaml:"strategy"`
	Steps       []Step `yaml:"steps"`
	Uses        string `yaml:"uses"`
	Needs       any    `yaml:"needs"`
	Container   any    `yaml:"container"`
	Services    any    `yaml:"services"`
	Environment any    `yaml:"environment"`
}
type Step struct {
	Name string         `yaml:"name"`
	Run  string         `yaml:"run"`
	Uses string         `yaml:"uses"`
	With map[string]any `yaml:"with"`
}

func Parse(b []byte) (Workflow, error) {
	var w Workflow
	if e := yaml.Unmarshal(b, &w); e != nil {
		return w, e
	}
	if len(w.Jobs) == 0 {
		return w, fmt.Errorf("workflow has no jobs")
	}
	if e := yaml.Unmarshal(b, &w.Raw); e != nil {
		return w, e
	}
	return w, nil
}
func clone(m map[string]any) map[string]any {
	c := map[string]any{}
	for k, v := range m {
		c[k] = v
	}
	return c
}
func subset(a, b map[string]any) bool {
	for k, v := range a {
		if fmt.Sprint(b[k]) != fmt.Sprint(v) {
			return false
		}
	}
	return true
}
func Matrix(raw any) ([]map[string]any, error) {
	if raw == nil {
		return []map[string]any{{}}, nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("dynamic matrix cannot be reconstructed from job names")
	}
	var keys []string
	for k := range m {
		if k != "include" && k != "exclude" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	rows := []map[string]any{{}}
	for _, k := range keys {
		values, ok := m[k].([]any)
		if !ok {
			return nil, fmt.Errorf("matrix axis %s is not a static list", k)
		}
		var next []map[string]any
		for _, row := range rows {
			for _, v := range values {
				if _, ok := v.(map[string]any); ok {
					return nil, fmt.Errorf("object matrix axes are outside V1")
				}
				c := clone(row)
				c[k] = v
				next = append(next, c)
				if len(next) > 256 {
					return nil, fmt.Errorf("matrix exceeds 256 combinations")
				}
			}
		}
		rows = next
	}
	if v, exists := m["exclude"]; exists {
		ex, ok := v.([]any)
		if !ok {
			return nil, fmt.Errorf("invalid matrix exclude")
		}
		for _, v := range ex {
			e, ok := v.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid matrix exclude entry")
			}
			var keep []map[string]any
			for _, r := range rows {
				if !subset(e, r) {
					keep = append(keep, r)
				}
			}
			rows = keep
		}
	}
	original := make([]map[string]any, len(rows))
	for i, r := range rows {
		original[i] = clone(r)
	}
	if len(keys) == 0 {
		rows = nil
		original = nil
	}
	if v, exists := m["include"]; exists {
		inc, ok := v.([]any)
		if !ok {
			return nil, fmt.Errorf("invalid matrix include")
		}
		for _, v := range inc {
			entry, ok := v.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid matrix include entry")
			}
			merged := false
			for i, base := range original {
				compatible := true
				for _, k := range keys {
					if value, ok := entry[k]; ok && fmt.Sprint(base[k]) != fmt.Sprint(value) {
						compatible = false
					}
				}
				if compatible {
					for k, v := range entry {
						rows[i][k] = v
					}
					merged = true
				}
			}
			if !merged {
				rows = append(rows, clone(entry))
			}
		}
	}
	if len(rows) > 256 {
		return nil, fmt.Errorf("matrix exceeds 256 combinations")
	}
	return rows, nil
}

var expression = regexp.MustCompile(`\$\{\{\s*(.*?)\s*\}\}`)
var keyExpr = regexp.MustCompile(`^matrix\.([A-Za-z0-9_-]+)$`)

// Render supports matrix.key and || literals in names and runner labels only.
// Commands and env expressions remain intact for act to evaluate.
func Render(s string, m map[string]any) (string, bool) {
	ok := true
	out := expression.ReplaceAllStringFunc(s, func(token string) string {
		inner := expression.FindStringSubmatch(token)[1]
		parts := strings.Split(inner, "||")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			var v string
			if k := keyExpr.FindStringSubmatch(p); k != nil {
				if x, exists := m[k[1]]; exists {
					v = fmt.Sprint(x)
				}
			} else if len(p) >= 2 && p[0] == '\'' && p[len(p)-1] == '\'' {
				v = p[1 : len(p)-1]
			} else {
				ok = false
				return token
			}
			if v != "" && v != "false" {
				return v
			}
		}
		return ""
	})
	return out, ok
}

type Selection struct {
	ID     string
	Job    Job
	Matrix map[string]any
	Runner string
}

func runner(j Job, m map[string]any) (string, bool) {
	switch v := j.RunsOn.(type) {
	case string:
		return Render(v, m)
	case []any:
		if len(v) == 1 {
			return Render(fmt.Sprint(v[0]), m)
		}
	}
	return "", false
}
func Select(w Workflow, remoteName string) (Selection, error) {
	var matches []Selection
	var unresolved []string
	for id, j := range w.Jobs {
		rows, e := Matrix(j.Strategy.Matrix)
		if e != nil {
			unresolved = append(unresolved, id+": "+e.Error())
			continue
		}
		for _, m := range rows {
			name := j.Name
			if name == "" {
				name = id
				if len(m) > 0 {
					var keys []string
					for k := range m {
						keys = append(keys, k)
					}
					sort.Strings(keys)
					var values []string
					for _, k := range keys {
						values = append(values, fmt.Sprint(m[k]))
					}
					name += " (" + strings.Join(values, ", ") + ")"
				}
			}
			name, ok := Render(name, m)
			if !ok {
				continue
			}
			matched := name == remoteName
			// GitHub's default matrix name follows YAML axis order, not sorted Go map order.
			if !matched && j.Name == "" && len(m) > 0 {
				prefix := id + " ("
				if strings.HasPrefix(remoteName, prefix) && strings.HasSuffix(remoteName, ")") {
					parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(remoteName, prefix), ")"), ", ")
					var values []string
					for _, v := range m {
						values = append(values, fmt.Sprint(v))
					}
					sort.Strings(parts)
					sort.Strings(values)
					matched = strings.Join(parts, "\x00") == strings.Join(values, "\x00")
				}
			}
			if matched {
				r, _ := runner(j, m)
				matches = append(matches, Selection{id, j, m, r})
			}
		}
	}
	if len(matches) != 1 {
		return Selection{}, fmt.Errorf("job/matrix mapping for %q has %d candidates (requires exactly one); %s", remoteName, len(matches), strings.Join(unresolved, "; "))
	}
	return matches[0], nil
}
func FailedCommand(j Job, name string, m map[string]any) (string, int, error) {
	var candidates []int
	for i, s := range j.Steps {
		n := s.Name
		if n == "" {
			if s.Run != "" {
				n = "Run " + strings.Split(s.Run, "\n")[0]
			} else {
				n = "Run " + s.Uses
			}
		}
		n, ok := Render(n, m)
		if ok && n == name {
			candidates = append(candidates, i)
		}
	}
	if len(candidates) != 1 {
		return "", -1, fmt.Errorf("failed step %q cannot be uniquely mapped", name)
	}
	i := candidates[0]
	cmd := j.Steps[i].Run
	if cmd == "" {
		cmd = "uses: " + j.Steps[i].Uses
	}
	return cmd, i, nil
}

// Replay keeps the chosen job, original conditions and steps, and exactly one matrix row.
func Replay(w Workflow, s Selection) ([]byte, error) {
	raw := clone(w.Raw)
	jobs := raw["jobs"].(map[string]any)
	job := clone(jobs[s.ID].(map[string]any))
	if s.Job.Strategy.Matrix != nil {
		job["strategy"] = map[string]any{"fail-fast": false, "matrix": map[string]any{"include": []any{s.Matrix}}}
	}
	raw["jobs"] = map[string]any{s.ID: job}
	delete(raw, "concurrency")
	return yaml.Marshal(raw)
}
