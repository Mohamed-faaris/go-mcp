# Agent Coordination and Context Continuity

This document defines how separate web agents, browser agents, and coding sessions communicate and resume work after a chat ends.

## Core rule

A chat is temporary. A project context is persistent.

The Git repository is the canonical source of truth for code, product requirements, architecture, contracts, and accepted decisions. The go-mcp control plane stores live coordination state and context snapshots so an agent in a new chat can resume without access to the previous transcript.

## Components

- **Project context service:** assembles repository facts, current branch/commit, architecture, decisions, and current work items.
- **Agent session service:** tracks each external web-agent session, its capabilities, activity, heartbeat, project, task, branch, and status.
- **Message/event store:** durable, append-only messages and activity events.
- **Work coordinator:** leases work items and path locks to avoid concurrent edits.
- **Handoff service:** captures compact, versioned context snapshots and bootstrap instructions.
- **Repository adapter:** reads canonical docs and records accepted decisions/handoffs into Git when requested.

## Session bootstrap

A web agent starting in a new chat should:

1. Receive a project ID or repository URL and optional task/workstream ID.
2. Register a new session using `register_agent_session`.
3. Request `resume_agent_session` or `get_project_context`.
4. Read `AGENTS.md`, `REWORK_PLAN.md`, `ARCHITECTURE.md`, this file, and its assigned workstream file from the current repository branch.
5. Verify the branch, commit SHA, and active work claims.
6. Receive a context bundle containing current goal, progress, decisions, work ownership, open issues, relevant messages, tests, and exact next actions.
7. Announce that the session has resumed and claim only its assigned work item.

Do not inject an entire previous chat transcript by default. Restore a compact summary plus recent relevant messages and pointers to source files/commits. If a required detail is missing, ask the project coordinator or inspect the repository; do not invent old decisions.

## Context snapshot schema

```json
{
  "snapshotId": "ctx_...",
  "projectId": "project_...",
  "taskId": "task_...",
  "workstreamId": "ui",
  "repository": {
    "url": "https://github.com/owner/repo",
    "branch": "feat/example",
    "commit": "full-commit-sha"
  },
  "goal": "Implement task-first monochrome application shell",
  "summary": "Completed navigation and base layout; task views remain.",
  "constraints": ["Do not add host access to workload containers"],
  "decisions": [
    {"id": "ADR-0002", "path": "docs/decisions/0002-agent-sessions.md"}
  ],
  "completed": ["App shell", "Navigation"],
  "inProgress": ["Task detail page"],
  "changedPaths": ["web/src/App.tsx"],
  "checks": [{"name": "web build", "status": "passed", "commit": "full-commit-sha"}],
  "blockers": [],
  "nextActions": ["Implement task detail tabs"],
  "previousSnapshotId": "ctx_previous",
  "createdAt": "RFC3339"
}
```

Snapshots are immutable. A new handoff creates a new snapshot linked to the previous snapshot. The control plane can materialize the bundle into a generated bootstrap prompt for a new web-agent chat.

## Communication channels

Channels are project-, task-, and workstream-scoped:
- `project:<project-id>`: project-wide coordination and architecture proposals.
- `task:<task-id>`: task-specific messages.
- `workstream:<workstream-id>`: implementation coordination.
- `agent:<session-id>`: private agent-to-agent messages.

Every message must include sender session, project ID, channel, timestamp, sequence cursor, correlation ID, type, and body. Message types: `announce`, `progress`, `question`, `decision_proposal`, `blocker`, `review_request`, `handoff`, `completed`.

Messages are stored before being broadcast. SSE/WebSocket provides live delivery; the database is the persistence layer. Clients recover missed messages using a monotonic cursor.

## Tool/API contracts

| Tool | Purpose |
|---|---|
| `register_agent_session` | Create a session and join project/task channels |
| `resume_agent_session` | Restore latest snapshot and relevant history |
| `get_project_context` | Retrieve canonical project context and current handoff |
| `list_project_agents` | List active/idle sessions and work claims |
| `send_agent_message` | Persist and broadcast a project/task/agent message |
| `read_agent_messages` | Read messages after a cursor |
| `publish_agent_progress` | Publish a progress, blocker, or completion event |
| `publish_agent_artifact` | Attach report, patch summary, test output, or contract |
| `create_agent_handoff` | Store a new immutable context snapshot and optional repo document |
| `claim_work_item` | Claim a work item with an expiring lease |
| `release_work_item` | Release a claim |
| `acquire_path_lock` | Acquire one or more path locks with a lease |
| `renew_path_lock` | Renew an existing lease |
| `release_path_lock` | Release path locks |
| `request_agent_review` | Ask another agent to review a commit or artifact |

Proposed HTTP endpoints, versioned under `/api/v1`:
- `POST /agent-sessions`
- `POST /agent-sessions/{sessionID}/resume`
- `GET /projects/{projectID}/context`
- `GET /projects/{projectID}/agents`
- `POST /projects/{projectID}/messages`
- `GET /projects/{projectID}/messages?after=<cursor>`
- `GET /projects/{projectID}/events` (SSE or WebSocket)
- `POST /work-items/{workItemID}/claim`
- `POST /work-items/{workItemID}/release`
- `POST /locks`, `POST /locks/{lockID}/renew`, `DELETE /locks/{lockID}`
- `POST /agent-sessions/{sessionID}/handoffs`
- `GET /agent-sessions/{sessionID}/context`

## Persistence model

Suggested tables/collections:
- `projects`
- `tasks`
- `agent_sessions`
- `agent_messages`
- `agent_events`
- `work_items`
- `work_claims`
- `path_locks`
- `context_snapshots`
- `artifacts`
- `approvals`

Store messages and events append-only. Store snapshot metadata and payload immutably. Retain project summaries and latest snapshots after the browser chat/session ends. Apply access checks to every project/task/session resource. Use TTL/retention settings only for transient logs and leases, not for accepted decisions or snapshots needed for resumption.

## Multi-agent concurrency

- Each agent declares its target workstream and file ownership before editing.
- Prefer separate branches/worktrees per agent and merge through one integration workstream.
- Claims and path locks are leases. If heartbeat stops, the lease expires and another agent can claim the work after refreshing the branch.
- Never silently steal an unexpired claim. Send a message to the owner and ask the coordinator for reassignment.
- Different agents must not modify the same files at the same time.
- Shared interfaces are proposed in `docs/contracts/` and accepted through coordinator review before implementation fans out.
- Every decision that changes requirements or interfaces must be recorded in `docs/decisions/` and committed.

## Handoff at end of chat

Before ending:
1. Commit work on your branch.
2. Publish progress and a handoff snapshot.
3. Include branch, base commit, latest commit, changed paths, test results, blockers, and next actions.
4. Release or renew claims deliberately.
5. Leave the project channel with an explicit status message.

A replacement agent starts from the latest snapshot and repository commit, verifies both, and continues. It does not need the previous chat transcript.
