# ADR 0001: Repository as the Source of Truth

- Status: Accepted
- Date: 2026-11-10

## Context

The project will be implemented by multiple independent coding agents, potentially in separate browser sessions or chats. A chat transcript may not be available to the next agent and must not be relied on as durable project state.

## Decision

The Git repository is the canonical source of truth for product requirements, architecture, interfaces, accepted decisions, workstream ownership, implementation, and committed handoffs.

The control plane will store live coordination state: agent sessions, persisted messages, progress events, claims, leases, approvals, artifacts, and immutable context snapshots. This state helps agents resume and collaborate, but it cannot silently override committed repository specifications.

A new agent must read the current repository documents and verify the current branch/commit before continuing. A context snapshot is a concise index into the repository and current work, not an authoritative replacement for code or accepted decisions.

## Consequences

- Every accepted requirement/interface/decision must be committed.
- Parallel agents use separate branches/workstreams and publish handoffs.
- A new chat can resume from the latest context snapshot and Git state without reading the previous chat transcript.
- Stale context is detected by comparing branch/commit and snapshot versions.
- Until the coordination service exists, Git branches, PRs, and handoff documents provide durable collaboration.
