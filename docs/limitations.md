# Known limits and trust boundary

- act's container is not the GitHub-hosted virtual machine. Kernel, preinstalled tools,
  filesystem, network and service behavior can differ.
- The run head SHA is not necessarily the PR merge commit tested by checkout.
  RunBack uses historical checkout log evidence; missing PR evidence blocks execution.
- The historical webhook payload cannot generally be downloaded from the run API.
  Reconstructed event fields may be incomplete, especially for deleted/empty PR references.
- Only static matrices and a small expression subset used in names/runner labels are
  resolved. Commands remain unchanged for act to evaluate.
- Missing/expired logs do not become a synthetic remote fingerprint.
- Floating action refs, tool ranges, package registries and external services can drift.
- Only one supported failed job is replayed per invocation. Choose a remote job with
  --job when investigating another failure.
- Secrets, repository variables, custom checkout layouts, needs outputs, reusable job
  workflows, services and container jobs are not implemented.
- Matching is conservative line-set comparison. Some identical failures will be
  unverified or unmatched. Flakiness cannot be proven from one run.
- GitHub API transient failures are surfaced; rerun inspect/export to retry.
- Standard workflow code is untrusted executable code. Replay only repositories you
  are willing to execute in a disposable Linux lab. A Docker container is not a VM
  security boundary. RunBack does not forward resolver tokens or mount Docker's socket
  into the job container, and uses bridge networking and a separate act HOME.
- The Docker host still requires sufficient disk, memory and network access.
- Shell is available only when one retained container was recorded. Interrupted runs
  can leave containers for manual inspection; never use a global docker prune command
  to clean a RunBack case.
