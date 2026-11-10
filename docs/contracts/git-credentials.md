# Git Provider and Credential Contract

## Provider discovery

Inspect the selected project's Git metadata inside the manager-controlled workspace preparation stage. Record canonical fetch/push URLs and determine provider from normalized URL host. Never assume every remote is GitHub.

If a selected local folder has uncommitted work, detect it and require an explicit choice between clean remote clone and an approved snapshot of local changes. Never silently discard edits.

## GitHub path

- The host-side credential broker owns the GitHub App private key/installation credentials.
- Resolve the exact GitHub owner/repository from the validated remote.
- Confirm the installation can access that repository.
- Create a short-lived installation token limited to that repository and only required permissions, normally Contents read/write.
- The worker uses a Git transport/credential broker. Prefer a host-side proxy that injects the token and returns Git traffic without revealing the raw token to the agent or repository.
- Never put tokens in prompts, tool arguments, process command lines, shell history, Git remote URLs, logs, or project files.
- Revoke task credential material when possible and rely on token expiry as a backstop.

## Tool boundary

Expose logical operations: inspect remote, fetch, create branch, commit, push branch, and optionally create PR. Do not expose the credential broker's raw token API to workload agents.

## Local-only repository

Local commits and branches remain available without a remote credential. Never claim the changes were pushed if no valid remote push was observed.

## Other providers

Add provider-specific adapters. A GitHub App token must never be sent to GitLab, Bitbucket, arbitrary HTTPS, or SSH hosts. Unknown remotes default to no credentials until an explicit provider is configured.
