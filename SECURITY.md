# Security

Report a vulnerability privately through the repository's security reporting feature
when enabled. Do not post tokens, credentials or private logs in a public issue.

RunBack executes public repository workflows using act and Docker. Use a disposable lab.
The resolver never needs to pass its GitHub credential into a replay container.
Review imported locks/bundles as executable workflow inputs before using replay.

The project does not disable host security controls, clear unrelated Docker resources,
or request user credentials in a lockfile. Logs and bundles are local evidence and should
be reviewed before publication, even when their source repository is public.
