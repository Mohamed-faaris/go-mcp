# Cross-Workstream Contracts

These documents define shared interfaces that parallel workstreams must follow. A change to a shared contract requires coordinator review and a decision record when it changes an accepted architectural choice.

- [Task lifecycle](task-lifecycle.md): task resource, state transitions, cancellation and reports.
- [Container runtime](container-runtime.md): runtime contract, labels, isolation and ownership.
- [Git credentials](git-credentials.md): remote detection, GitHub App broker and safe transport.
- [Tool permissions](tool-permissions.md): catalog, policy levels, approval requests and hard denies.
- [Agent coordination](agent-coordination.md): web-agent sessions, messaging, context bundles, handoffs, claims and path locks.
- [HTTP API](http-api.md): versioned API routes and common conventions.
- [UI design system](ui-design-system.md): monochrome shadcn conventions, navigation and page structure.

Do not invent a parallel request/response shape without updating the relevant contract first.
