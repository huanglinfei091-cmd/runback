# GitHub Public Alpha Launch

M1, M2, Direct URL MVP and Case C are frozen as PASS. This stage publishes the existing
truthful core as a small public alpha without expanding Actions compatibility.

Execution order:

1. Scan the complete publication set for credentials, private keys, local credentials,
   runtime data and personal machine information. Stop before push if a real secret exists.
2. Finish the concise README, diagnostic-only doctor, `runback version`, Bug report form,
   Linux amd64 artifact and checksum.
3. Validate the artifact from a RunBack-owned clean HOME with no development checkout,
   prior session, lock, cache or hidden GitHub CLI configuration.
4. Run Go test/vet/build, M1/M2 integrity, Direct URL Case C, Online/Bundle parity and
   token-leak gates without weakening the Matcher.
5. Commit from the Windows control copy, create one public `runback` repository, push the
   verified commit, create tag and GitHub Release `v0.1.0-alpha`, then verify the public UI.
6. Build a candidate pool of at most 100 recent, active public Ubuntu failures, qualify 30,
   and contact at most 15 relevant maintainers in the first round. Never advertise through
   unrelated issues or PRs, ask for Stars, repeat after refusal, or fabricate feedback.
7. Record only real installs, doctor results, executions, verdicts and user responses.

The release contains only `runback-v0.1.0-alpha-linux-amd64.tar.gz` and `SHA256SUMS`.
The tarball contains the binary, README and existing MIT LICENSE. No Windows artifact is
published in this release.

Doctor and release validation must not install Docker/act/Git, modify docker0 or the
firewall, restart Docker, alter daemon configuration, or change host networking. GitHub
credentials are allowed only in the online acquisition process and are never printed,
stored, transferred, or passed into execution.
