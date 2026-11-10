# UI Design System Contract

## Stack

React, TypeScript, Vite, Tailwind CSS, shadcn/ui conventions and Radix primitives. Keep component source in the repo; do not rely on a remote shadcn registry at runtime.

## Visual system

- Strict neutral black, white, and grey palette.
- No decorative gradients, glows, or colorful dashboard tiles.
- Use semantic status colors only where required for status/error meaning and accessibility.
- Strong typographic hierarchy, compact data tables, keyboard-accessible controls, visible focus rings.
- Use shadcn-inspired primitives: Button, Card, Badge, Dialog, AlertDialog, DropdownMenu, Sheet, Tabs, Table, Accordion, Collapsible, Form, Input, Select, Textarea, Tooltip, Separator, Skeleton, Sonner.

## Main navigation

Overview, Tasks, Projects, Containers, Approvals, Tools, Settings.

## Dashboard

Show only high-value operational summary: active/queued/waiting-approval task counts, recent tasks, latest activity, and New Task action. Do not put runtime configuration or every setting on the dashboard.

## New task

Keep project/repository, prompt, preset, and submit visible. Place CPU/memory/time, branch, environment, network, preview, retention, MCP and approval overrides under an Advanced accordion/collapsible. Use dialogs for custom preset, repository integration and risky confirmations.

## Task detail

Tabs: Overview, Activity, Changes, Checks, Commits, Report. Show live events and agents/handoff panel. Terminal and container internals belong in detail views.

## Tools

Searchable catalog. Each tool row shows name, purpose, category, scope, enablement and policy. Use a dialog to edit policy. Keep hard-denied tools immutable.

## Agent collaboration

Show sessions, assigned workstream, branch/commit, last heartbeat, progress, messages, active leases, and handoff snapshots. A resume action produces a bootstrap/context bundle for a new external web-agent chat.
