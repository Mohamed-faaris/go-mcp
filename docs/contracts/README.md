# Cross-Workstream Contracts

This directory holds stable interfaces agreed before parallel implementation. If a contract changes, create a decision record and notify affected workstreams.

## Required contracts

- `task-lifecycle.md`: task status transitions, task identity, ownership, cancellation semantics.
- `container-runtime.md`: runtime interface, container labels, resource limits, ownership checks, lifecycle behavior.
- `git-credentials.md`: remote detection, GitHub App permissions, host credential broker and Git transport.
- `tool-permissions.md`: tool catalog schema, policy evaluation, approval request lifecycle and hard denies.
- `agent-coordination.md`: session/message/context/claim/lock/handoff schemas.
- `http-api.md`: versioned API request/response schemas shared by backend and web UI.
- `ui-design-system.md`: monochrome token names, shadcn component conventions, page/routes and reusable patterns.

No workstream should invent an incompatible request/response shape in isolation. Propose interface additions in the relevant contract and ask the integration coordinator to approve the change.
