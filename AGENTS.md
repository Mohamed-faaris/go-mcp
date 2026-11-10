# Instructions for AI coding agents

This repository is the source of truth for the go-mcp rework. Read this file before making changes.

## Required bootstrap

1. Read `README.md`, `REWORK_PLAN.md`, and `ARCHITECTURE.md`.
2. Read `docs/agents/COORDINATION.md` and `docs/agents/WORKSTREAMS.md`.
3. Read the workstream handoff assigned to you, if one exists.
4. Inspect the current Git branch, working tree, and recent commits before editing.
5. Restore the latest project context from the coordinator when the agent-session API is available. Until then, use the latest committed handoff and repository state.
6. Confirm your assigned workstream, allowed paths, acceptance criteria, and dependencies before editing.

## Source-of-truth rules

- Committed repository files define product requirements, architecture, interfaces, decisions, and implementation.
- `REWORK_PLAN.md` is the product-level plan.
- `ARCHITECTURE.md` is the architecture and security contract.
- `docs/agents/COORDINATION.md` defines communication and context continuity.
- `docs/agents/WORKSTREAMS.md` defines parallel work ownership.
- `docs/decisions/` contains accepted architectural decisions.
- The control-plane event store will retain live agent messages, claims, progress, and context snapshots. It is coordination state, not a replacement for committed specifications.
- Never treat an uncommitted chat message as an accepted product decision. Update the appropriate repository document through a reviewed change.

## Parallel work rules

- Work only in your assigned paths. Do not make unrelated edits.
- Use a dedicated branch/worktree per workstream or agent.
- Never have two agents editing the same file concurrently. Acquire a file/work-item lease through the coordination service when available.
- Before beginning, announce the work item, intended files, and expected interface dependencies.
- Before changing a shared contract, send a proposal to the coordinator and update the contract through a dedicated review.
- Commit small, logically grouped changes. Do not push to the default branch directly.
- Send progress updates through the shared project/task channel, not only in the current chat.
- When blocked, publish the blocker and the exact decision/input required.
- At handoff, commit changes, publish a handoff snapshot, list commits and changed paths, report checks, and list remaining work.
- The integration agent reviews and merges workstream branches. Avoid cherry-picking or merging another agent's unreviewed branch.

## Coding and validation rules

- Preserve the Go and React/TypeScript stack unless the plan is explicitly changed.
- Keep the Podman socket and host control-plane credentials in the host control plane only.
- Workload agents have broad development access inside their own task container only.
- Container ownership is verified by the manager registry and immutable IDs, not names or caller-supplied labels alone.
- GitHub credentials are issued by the host credential manager and scoped to the assigned repository/task.
- Tool permissions use `automatic`, `approval_required`, or `disabled`; hard security denies cannot be overridden by UI settings.
- Run the tests relevant to your change. Do not claim a build or test passed unless you ran it and observed the exit status.
- Document commands that could not be run and why.

## Required handoff format

Every agent handoff must include:

- Workstream and agent/session ID.
- Branch and base commit.
- Summary of completed work.
- Commits and changed paths.
- Tests/checks run with results.
- Decisions made and links to decision records.
- Open issues, blockers, and remaining tasks.
- Exact next action for the receiving agent.
- Context snapshot ID or handoff file path.
