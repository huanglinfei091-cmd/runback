# RunBack real-world evaluation

Store discovered public run metadata in candidates/, case manifests in cases/,
machine-readable measurements in results/, and written conclusions in reports/.

A fixture proves a code path; it is never counted as a real reproduction.

## Execute

Export evidence in Windows with scripts/export-evidence.ps1. Copy the JSON bundles
to a directory in Ubuntu:

```bash
python3 benchmark/scripts/run.py --evidence ~/runback-work/cases --execute
```

Without --execute the tool runs inspection only. It explicitly reports no measured
reproduction rate. Each case uses a separate lock and workspace.

Report these counts together: collected, outside scope, eligible, unclassified,
executed and reproduced. Eligible reproduction rate is reproduced / eligible.
Publish unclassified cases as well; do not quietly remove hard failures from the sample.

SUCCESS, OUT_OF_SCOPE, RUNNER_UNSUPPORTED, SECRETS_REQUIRED, NETWORK_DEPENDENCY,
ENVIRONMENT_MISMATCH, ACT_INCOMPATIBILITY, MATRIX_RESOLUTION_FAILED,
WORKFLOW_UNSUPPORTED, FLAKY_TEST, FAILURE_NOT_MATCHED and UNKNOWN are allowed outcomes.

The current evaluator only assigns FLAKY_TEST after human/independent repeat evidence;
it does not infer flakiness from a mismatch. Similarity scores are not probabilities.

Targets: 30+ real runs across 10+ repos with Node, Python, Go and matrix success cases;
then 100 real runs. Targets are not reported as achievements.
