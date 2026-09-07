package workflow

import "testing"

func TestExecutionPrecedence(t *testing.T) {
	w, e := Parse([]byte("env:\n  A: global\ndefaults:\n  run:\n    shell: sh\n    working-directory: root\njobs:\n  test:\n    env:\n      A: job\n    steps:\n      - run: echo test\n        shell: bash\n        working-directory: pkg\n        env:\n          A: step\n"))
	if e != nil {
		t.Fatal(e)
	}
	c := Execution(w, "test", 0)
	if c.Shell != "bash" || c.WorkingDirectory != "pkg" || c.Environment["A"] != "step" {
		t.Fatal(c)
	}
}
