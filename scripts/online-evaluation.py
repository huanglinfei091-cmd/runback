"""Ubuntu-local live evaluation. Uses only the environment supplied on Ubuntu; never accepts credentials in requests."""
import json
import os
from pathlib import Path
import subprocess
import sys
import time

request = json.loads(sys.stdin.readline())
if "token" in request:
    raise SystemExit("Credential input is not accepted; configure the Ubuntu process environment manually")
env = dict(os.environ)
if request.get("use_local_gh", False):
    credential = subprocess.run(["gh", "auth", "token"], capture_output=True, text=True)
    if credential.returncode or not credential.stdout.strip():
        raise SystemExit("Ubuntu-local GitHub credential unavailable; no credential output was retained")
    env["RUNBACK_GITHUB_TOKEN"] = credential.stdout.strip()
    del credential
token_used = bool(env.get("RUNBACK_GITHUB_TOKEN") or env.get("GH_TOKEN"))
env.pop("GITHUB_TOKEN", None)
repo = Path("/root/runback-work/runback")
out = repo / "docs/online/evidence" / request["name"]
assert out.parent == repo / "docs/online/evidence"
out.parent.mkdir(parents=True, exist_ok=True)
command = [str(repo / "bin/runback"), *request["args"]]
start = time.perf_counter()
result_time = None
result = None
with out.open("w") as log:
    p = subprocess.Popen(command, cwd=repo, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    for line in p.stdout:
        log.write(line)
        if line.startswith("Result: "):
            result_time = time.perf_counter() - start
            result = line.strip().split(": ", 1)[1]
    code = p.wait()
elapsed = time.perf_counter() - start
measurement = {"args": request["args"], "token_used": token_used, "exit": code,
               "ttfr_to_result_seconds": result_time, "cli_launch_to_exit_seconds": elapsed, "result": result}
out.with_suffix(".timing.json").write_text(json.dumps(measurement, indent=2))
print(json.dumps(measurement))
print("Log:", str(out))
