"""Measure actual CLI process launch-to-exit; use the active, still-unmodified session."""
import json
import os
from pathlib import Path
import statistics
import subprocess
import time

repo = Path(__file__).resolve().parents[2]
evidence = repo / "docs/m2/evidence"
base = Path.home() / ".runback/sessions"
session = base / (base / "active").read_text().strip()
state = json.loads((session / "state.json").read_text())
assert subprocess.check_output(["git", "-C", state["worktree"], "status", "--porcelain"]).strip() == b"", "measure before fixing source"
measurements = []
def invoke(label, args):
    path = evidence / (label + ".log")
    with path.open("wb") as log:
        started = time.perf_counter()
        p = subprocess.run([str(repo / "bin/runback"), *args, "--timeout", "6m"], cwd=repo, stdout=log, stderr=subprocess.STDOUT)
        duration = time.perf_counter() - started
    text = path.read_text()
    assert p.returncode == 0 and "Result: SAME_FAILURE" in text, (label, p.returncode, text[-1200:])
    row = {"label": label, "cli_launch_to_exit_seconds": duration, "exit": p.returncode,
           "image_already_present": True, "image_id": state["image_id"],
           "session": state["id"], "dependency_setup_observed": "Installed " in text,
           "network_download": "OBSERVED" if "Downloading " in text or "Downloaded " in text else "NOT_OBSERVED_NOT_PROVEN_OFFLINE"}
    measurements.append(row)
    (evidence / "performance.json").write_text(json.dumps({"measurements": measurements}, indent=2))
    print(label, round(duration, 3), flush=True)

# Populate this session's act action downloads and the shared uv cache before the baseline.
invoke("full-prime", ["reproduce", "--session", state["id"]])
for i in range(1, 4):
    invoke(f"warm-full-{i}", ["reproduce", "--session", state["id"]])
    invoke(f"warm-fast-{i}", ["replay", "--step", "--session", state["id"]])
full = statistics.median(r["cli_launch_to_exit_seconds"] for r in measurements if r["label"].startswith("warm-full"))
fast = statistics.median(r["cli_launch_to_exit_seconds"] for r in measurements if r["label"].startswith("warm-fast"))
report = {"measurements": measurements, "warm_full_reproduce_duration": full, "fast_replay_duration": fast,
          "speedup_ratio": full / fast, "aggregation": "median of 3 interleaved pairs",
          "conditions": "Same host, same unmodified session, same image ID; session uv cache and act tools primed. Full job creates project environments in a fresh checkout; fast replay reuses worktree .venv/.tox. Network remains enabled; setup actions can contact GitHub even with cached tools."}
(evidence / "performance.json").write_text(json.dumps(report, indent=2))
print(json.dumps({k:v for k,v in report.items() if k != "measurements"}), flush=True)
