# Agent Coordination API Contract

This API allows independent browser/web agents and separate chat sessions to share durable project context. Chat transcripts are not the persistence mechanism.

## Identity

Every call is authenticated by a control-plane principal. An agent session is a child identity with project, task/workstream, capabilities, and branch scope. Every endpoint checks project membership and session ownership.

## Session

```json
{
  "id": "session_uuid",
  "projectId": "project_uuid",
  "taskId": "task_uuid",
  "workstreamId": "ui",
  "agentLabel": "web-agent-2",
  "branch": "agent/ui-shell",
  "commit": "full_sha",
  "status": "active",
  "lastHeartbeatAt": "RFC3339"
}
```

Register and resume APIs return a `contextBundle` with the latest snapshot, repository state, accepted decisions, active tasks, ownership claims, recent relevant messages, tests, blockers, and next actions.

## Message

```json
{
  "id": "message_uuid",
  "projectId": "project_uuid",
  "taskId": "task_uuid",
  "channel": "workstream:ui",
  "senderSessionId": "session_uuid",
  "recipientSessionId": null,
  "type": "progress",
  "body": {"text": "Task detail skeleton implemented"},
  "correlationId": "uuid",
  "cursor": "monotonic_cursor",
  "createdAt": "RFC3339"
}
```

Messages are durably stored before live broadcast. Clients read by cursor; SSE/WebSocket only delivers new events and can reconnect without data loss.

## Context snapshot

Snapshots are immutable and linked to a previous snapshot. Include repository URL, branch, base/latest commit, workstream, objective, constraints, accepted decisions, completed/in-progress work, changed paths, observed checks, blockers, next actions, relevant message cursor, and timestamp.

## Work claims and path leases

A claim/lock includes owner session, project, task/workstream, resource/path, lease expiry, and heartbeat. Leases expire if the agent disappears. Do not permit two live write leases for the same path. Claim conflicts are returned as HTTP 409 with current owner and expiry.

## Endpoints

- `POST /api/v1/agent-sessions`
- `POST /api/v1/agent-sessions/{id}/resume`
- `POST /api/v1/agent-sessions/{id}/heartbeat`
- `GET /api/v1/agent-sessions/{id}/context`
- `POST /api/v1/agent-sessions/{id}/handoffs`
- `GET /api/v1/projects/{projectId}/context`
- `GET /api/v1/projects/{projectId}/agents`
- `POST /api/v1/projects/{projectId}/messages`
- `GET /api/v1/projects/{projectId}/messages?after=<cursor>`
- `GET /api/v1/projects/{projectId}/events` (SSE or WebSocket)
- `POST /api/v1/work-items/{id}/claim`
- `POST /api/v1/work-items/{id}/release`
- `POST /api/v1/path-locks`
- `POST /api/v1/path-locks/{id}/renew`
- `DELETE /api/v1/path-locks/{id}`

All list reads are cursor-paginated. Context retrieval must filter messages to the requested project/task/workstream and avoid cross-project data leakage.
