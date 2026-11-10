# Tool Catalog and Permission Contract

## Policy values

- `automatic`: allowed within the current task's explicit scope.
- `approval_required`: create a pending approval for the exact operation and pause execution.
- `disabled`: do not expose the tool to the model and reject direct invocation.
- `hard_denied`: a security invariant; cannot be changed in UI or approved.

## Tool definition

```json
{
  "id": "container.delete.other",
  "name": "delete_other_container",
  "description": "Request deletion of another managed task container",
  "category": "container",
  "scope": "manager-mediated",
  "enabled": true,
  "policy": "approval_required",
  "risk": "high",
  "source": "native",
  "updatedAt": "RFC3339"
}
```

## Approval request

```json
{
  "id": "approval_uuid",
  "taskId": "task_uuid",
  "agentSessionId": "session_uuid",
  "toolId": "container.delete.other",
  "resource": {"type": "container", "id": "immutable_runtime_id"},
  "argumentsDigest": "sha256:...",
  "summary": "Delete managed container created by task ...",
  "status": "pending",
  "expiresAt": "RFC3339"
}
```

Approval is single-use, expires, and is bound to the exact operation/resource/arguments digest. The control plane re-evaluates policy and ownership immediately before execution. A changed argument set requires a new approval.

## UI configuration

An admin can change tool enablement and policy with a dialog. The UI displays scope, risk, policy, source and description. Server-side authorization remains authoritative. The UI cannot modify `hard_denied` tools.

## Initial policy defaults

| Operation | Policy |
|---|---|
| Project file operations inside task workspace | automatic |
| Shell and process tools inside assigned container | automatic |
| Local Git status/diff/branch/commit | automatic |
| Push to assigned GitHub repo | approval_required initially |
| Start additional container | approval_required |
| Delete another managed task container | approval_required |
| Delete own task container as part of cleanup | automatic manager cleanup |
| Delete unmanaged/protected/host container | hard_denied |
| Host filesystem/root/Podman socket access from workload | hard_denied |
| Change global security/auth settings | approval_required, admin-only |
