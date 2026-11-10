# Parallel Agent Workstreams

This document splits the rework into independent tracks so several web agents can work concurrently without editing the same files.

## Rules for assigning parallel agents

- The coordinator assigns one workstream and one branch to each agent.
- Each agent reads `AGENTS.md`, `REWORK_PLAN.md`, `ARCHITECTURE.md`, and `docs/agents/COORDINATION.md`.
- Do not start implementation until the shared contract in `docs/contracts/` is agreed.
- One workstream owns each path group. Avoid edits outside that group.
- Use task claims/path locks when the coordination service exists; until then the coordinator maintains assignments in this file and in GitHub issues/PRs.
- Each agent creates a PR against `feat/container-native-agent-rework`; only the integrator merges.
- Agents publish session progress and handoffs through the coordinator once available. Until then, commit a handoff under `docs/agents/handoffs/`.

## Recommended parallel tracks

### Track A — Control plane and task orchestration
**Owns:** `internal/task/`, `internal/scheduler/`, `internal/events/`, selected `internal/server/task_*.go`, task API tests.

**Deliver:** task persistence, state machine, queue, task API, cancellation, status events, durable task report links.

**Depends on:** runtime and task model contracts.

**Must not edit:** `web/`, `internal/podman/`, `internal/credentials/`.

### Track B — Container runtime and security
**Owns:** `internal/container/`, `internal/podman/`, `internal/workspace/`, runtime integration tests.

**Deliver:** runtime interface, ownership registry integration, isolated task workspace provisioning, container resource/security defaults, safe stop/delete behavior.

**Depends on:** shared task/container labels and workspace contract.

**Must not edit:** `web/`, `internal/agentcoord/`, task UI routes.

### Track C — Git provider and credential broker
**Owns:** `internal/git/`, `internal/credentials/`, Git transport/proxy, Git provider tests, GitHub App setup docs.

**Deliver:** inspect remotes, detect GitHub provider, request repository-scoped installation tokens from host manager, provide safe push transport, redact credentials, support local-only repositories.

**Depends on:** task identity and project/workspace contract.

**Must not edit:** UI components or Podman internals.

### Track D — Agent worker and native tools
**Owns:** `internal/worker/`, `internal/tools/`, `internal/approval/`, agent loop adapters and tests.

**Deliver:** container-bound agent runtime, shell/filesystem/process/search/Git/test tools, configurable tool policy, one-time approval workflow, manager-mediated cross-container operation requests.

**Depends on:** runtime, task identity, approval and Git interfaces.

**Must not edit:** global server router or frontend except through contract requests.

### Track E — Agent sessions and communication
**Owns:** `internal/agentcoord/`, `internal/contextstore/`, `internal/events/agent_*.go`, communication API tests, `docs/agents/COORDINATION.md` proposals.

**Deliver:** agent session registration/resume, project context bundle, persisted messages, progress events, work claims, path locks, handoff snapshots, context bootstrap prompt generation.

**Depends on:** principal/project/task identity and durable storage interface.

**Must not edit:** core container lifecycle and frontend global shell.

### Track F — React and shadcn UI redesign
**Owns:** `web/src/`, `web/components.json`, `web/src/components/ui/`, UI tests and design tokens.

**Deliver:** monochrome shadcn-inspired application, task-first dashboard, project/task views, agent activity and handoff, approvals inbox, tools catalog with policy editor, advanced collapsibles and dialogs.

**Depends on:** agreed API schemas in `docs/contracts/http-api.md`.

**Must not edit:** Go backend or architecture source-of-truth without a coordinator request.

### Track G — Integration, testing, deployment
**Owns:** `tests/integration/`, `deploy/`, CI workflows, root integration scripts, deployment documentation.

**Deliver:** full integration checks, security regression tests, Cloudflare Tunnel deployment, environment setup, backups/retention, release checklist.

**Depends on:** all feature tracks reaching interface-complete state.

**Must not edit:** feature implementation paths unless assigned by integrator.

## Integration ownership

A single integration coordinator owns:
- changes to shared contracts
- merging workstream PRs
- resolving cross-track conflicts
- updating overall status and accepted decisions
- running full repository checks
- release/acceptance sign-off

If two tracks need the same file, they must submit a proposed interface change instead of both editing it.

## Workstream status

| Track | Status | Owner | Branch / PR |
|---|---|---|---|
| A — Control plane | Planned | Unassigned | — |
| B — Container runtime | Planned | Unassigned | — |
| C — Git credentials | Planned | Unassigned | — |
| D — Agent tools/approval | Planned | Unassigned | — |
| E — Agent coordination | Planned | Unassigned | — |
| F — UI redesign | Planned | Unassigned | — |
| G — Integration | Planned | Unassigned | — |
