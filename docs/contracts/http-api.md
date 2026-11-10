# HTTP API Contract

Use `/api/v1` for new task, approval, Git, tool policy, and agent coordination APIs. Existing `/api` routes remain compatibility endpoints during migration and must not be treated as the long-term contract.

## Tasks
- `POST /api/v1/tasks`: create task and queue it.
- `GET /api/v1/tasks`: list caller-visible tasks.
- `GET /api/v1/tasks/{taskId}`: get task status.
- `POST /api/v1/tasks/{taskId}/cancel`: cancel task.
- `GET /api/v1/tasks/{taskId}/events`: SSE or WebSocket events.
- `GET /api/v1/tasks/{taskId}/report`: final structured report.

## Projects and presets
- `GET /api/v1/projects`
- `POST /api/v1/projects`
- `GET /api/v1/projects/{projectId}`
- `GET /api/v1/presets`
- `POST /api/v1/presets` (admin custom preset)
- `PUT /api/v1/presets/{presetId}` (admin custom preset)

## Tools and approvals
- `GET /api/v1/tools`
- `PUT /api/v1/tools/{toolId}/policy`
- `GET /api/v1/approvals?status=pending`
- `POST /api/v1/approvals/{approvalId}/approve`
- `POST /api/v1/approvals/{approvalId}/deny`

## Git
- `GET /api/v1/projects/{projectId}/git`: provider and remote metadata, never credentials.
- `POST /api/v1/tasks/{taskId}/git/push`: requests push through credential broker and policy enforcement.

## Containers
- `GET /api/v1/containers`: caller-visible managed containers only.
- `GET /api/v1/containers/{containerId}`
- `POST /api/v1/containers/{containerId}/stop`
- `DELETE /api/v1/containers/{containerId}`: ownership or approval checks required.
- Raw runtime API access is never exposed.

## Agent coordination

See `docs/contracts/agent-coordination.md` for session, message, event, handoff, claim, and path-lock contracts.

## API conventions

- JSON request/response bodies.
- Authenticated principal and per-resource authorization on every route.
- Opaque immutable IDs; don't authorize by display name.
- Bounded request/response sizes and log redaction.
- Use 400 for invalid input, 401 for unauthenticated, 403 for unauthorized, 404 for missing/not-visible resources, 409 for state/lease conflicts, 422 for validation failures.
- Mutating calls should be idempotent where practical and accept an idempotency key for task/approval creation.
- Streaming events include monotonic cursors and support reconnect/resume.
