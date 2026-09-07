"""Exercise the real runback dev CLI through a PTY, without product test flags."""
import json
import os
from pathlib import Path
import pty
import select
import shlex
import subprocess
import time

repo = Path(__file__).resolve().parents[2]
base = Path.home() / ".runback/sessions"
root = base / (base / "active").read_text().strip()
state = json.loads((root / "state.json").read_text())
worktree = Path(state["worktree"])
evidence = repo / "docs/m2/evidence"
master, slave = pty.openpty()
p = subprocess.Popen([str(repo / "bin/runback"), "dev", "--session", state["id"]], cwd=repo, stdin=slave, stdout=slave, stderr=slave, start_new_session=True)
os.close(slave)
transcript = bytearray()
sent = False
deadline = time.monotonic() + 40
# Only the Click example source is patched here. No executor behavior depends on repository identity.
fix = """from pathlib import Path
p=Path('src/click/types.py')
s=p.read_text()
assert 'import builtins\\n' in s
s=s.replace('import builtins\\n','',1)
s=s.replace('if t.TYPE_CHECKING:\\n','if t.TYPE_CHECKING:\\n    import builtins\\n',1)
p.write_text(s)
Path('.runback-bind-proof').write_text('persisted from dev container')
"""
try:
    while time.monotonic() < deadline:
        if select.select([master], [], [], .2)[0]:
            try: chunk = os.read(master, 65536)
            except OSError: break
            if not chunk: break
            transcript.extend(chunk)
        if not sent and b"# " in transcript:
            command = "pwd; python3 -c " + shlex.quote(fix) + "; exit\n"
            os.write(master, command.encode())
            sent = True
        if p.poll() is not None: break
    if p.poll() is None:
        p.terminate()
        raise RuntimeError("dev PTY timed out")
finally:
    os.close(master)
    (evidence / "dev-pty.log").write_bytes(transcript)
assert sent and p.wait() == 0
assert b"/github/workspace" in transcript
assert (worktree / ".runback-bind-proof").read_text() == "persisted from dev container"
(worktree / ".runback-bind-proof").unlink()
last = json.loads((root / "last-debug.json").read_text())
probe = subprocess.run(["docker", "inspect", last["container"]], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
assert probe.returncode != 0, "ephemeral container still exists"
diff = subprocess.check_output(["git","-C",str(worktree),"diff","--binary",state["lock"]["commit"]])
assert b"src/click/types.py" in diff and b".github/workflows" not in diff
(evidence / "source-fix.patch").write_bytes(diff)
(evidence / "dev-smoke.json").write_text(json.dumps({
    "session": state["id"], "cwd": "/github/workspace", "bind_write_persisted": True,
    "container_removed": True, "container": last["container"],
    "source_fix": "Move annotation-only builtins import under TYPE_CHECKING in src/click/types.py",
    "workflow_modified": False, "tests_modified": False
},indent=2))
print("PASS: real dev CLI PTY, source edit persisted, ephemeral container removed")
