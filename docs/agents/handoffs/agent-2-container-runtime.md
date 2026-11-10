# Agent 2 Handoff — Container Runtime and Security

- Workstream: B — Container runtime and security
- Agent: Agent 2
- Branch: `agent-2/container-runtime-security`
- Base branch: `feat/container-native-agent-rework`
- Status: ready for integration review
- Scope: runtime abstraction, Podman workload adapter, container ownership registry/manager, workspace safety

## Summary

Implemented the Track B runtime foundation without editing the server/task runner or UI paths owned by other workstreams.

### Completed

- Added `internal/container.Runtime`, `Spec`, `Info`, and resource limit types.
- Added `internal/podman.Runtime` adapter. It creates workloads from typed task/project/owner identity and deliberately does not forward host devices or arbitrary mounts.
- Hardened Podman container listing: a `gomcp-` name prefix no longer proves a container is managed. The list includes only the manager label, and authorization still requires the registry.
- Reserved the complete `gomcp.*` label namespace for the manager. Caller labels cannot forge current or future lifecycle/security metadata. The input environment slice is copied before adding internal environment values.
- Added read-only root filesystem, no-new-privileges, dropped capabilities, optional PID limits, and a configurable named network to workload create requests.
- Added explicit Podman stop timeout support using the Podman stop API's `t` query parameter. The old `StopContainer` method retains default server timeout semantics.
- Added atomic JSON `FileRegistry` with 0600 file mode, private parent directory, deterministic serialization, and a single-writer deployment caveat.
- Added `container.Manager` with allowlisted images, workspace-root validation, resource limit validation, manager-issued ownership labels, registry insertion, rollback on registry failure, and ownership-checked inspect/start/stop/delete/list methods.
- Added `workspace.Manager` for unique task workspace directories with 0700 permissions and safe removal of a direct child only.
- Strengthened workspace path validation against symlink escapes, path traversal, root removal, siblings, and non-directory paths.
- Removed the duplicate runtime preset catalog from this workstream. The shared `internal/presets` service on the integration branch is the preset source of truth; pass its image list to `container.ManagerOptions.AllowedImages`.

## Changed paths

- `internal/container/runtime.go`
- `internal/container/registry.go`
- `internal/container/manager.go`
- `internal/container/manager_test.go`
- `internal/podman/types.go`
- `internal/podman/client.go`
- `internal/podman/runtime.go`
- `internal/podman/runtime_test.go`
- `internal/podman/stop_test.go`
- `internal/workspace/path.go`
- `internal/workspace/path_test.go`
- `internal/workspace/manager.go`
- `internal/workspace/manager_test.go`

## Integration required from the server/task workstream

The shared `internal/server/rework.go` still creates containers directly with `s.podman.CreateContainer`. This bypasses the new ownership registry and manager policy. Before task containers are considered protected in production, the server constructor and task worker must be wired to the manager:

1. Create a dedicated task workspace root, e.g. `filepath.Join(cfg.Directories.TmpPodmans, "go-mcp-workspaces")`, using `workspace.NewManager` before constructing the container manager.
2. Open a durable `container.FileRegistry` under the configured data directory (e.g. `data/containers.json`).
3. Build `podman.NewRuntime(s.podman)`.
4. Build `container.NewManager(runtime, registry, ManagerOptions{WorkspaceRoot: workspaceRoot, AllowedImages: imagesFromSharedPresets, Network: configuredAgentNetwork, ...})`.
5. In `runTask`, call `CreateTaskContainer` with the owner, task, project, preset, image and prepared workspace. Call `StartOwned` before running agent commands.
6. Before using low-level Podman exec/log calls, verify ownership with `InspectOwned` or introduce a manager-mediated exec/log method.
7. Replace direct `StopContainer`/`RemoveContainer` cleanup with `DeleteOwned`. Remove the workspace only after the container has stopped/deleted and only when the task retention policy permits it.
8. Keep the legacy admin/manual container endpoints separate. They are not task-owned workloads and must not be deletable through agent-scoped methods.

The helper is intentionally not wired by this workstream because `internal/server/rework.go` and `internal/server/server.go` belong to the control-plane/task orchestration workstream. The integration owner should apply this wiring and add server-level tests.

## Security and behavior notes

- "Full access" is limited to the assigned workload container/workspace. The workload does not receive the Podman socket, host root, arbitrary device mounts, or sibling task workspace paths.
- Manager identity is cross-checked between registry records and live runtime labels before inspect/start/stop/delete.
- Deleting another task's container is not part of `DeleteOwned`; it must remain a separate approval-gated manager operation. Unmanaged or protected containers must remain hard-denied.
- `FileRegistry` is an atomic JSON store for a single control-plane writer. Use a transactional database when deploying multiple server replicas.
- The manager validates but does not itself enforce wall-clock timeout; the task scheduler/context must enforce it. Disk quotas are not implemented by this Podman adapter yet.
- The configured network must be a dedicated workload network chosen by the control plane. Leaving it empty allows Podman's default networking and is not an egress-isolation policy.
- The root filesystem is read-only with a writable `/workspace` bind mount and tmpfs support. Rootless Podman remains the recommended deployment mode.

## Validation

A network-isolated local Go 1.22-compatible harness was built from the Agent 2 container manager/registry/runtime definitions and workspace path/manager code, with a stub Podman client and an isolated stop-timeout HTTP adapter. It ran:

```bash
gofmt -w internal
go test -count=1 ./...
go vet ./...
```

Result: PASS for the isolated harness packages covering manager ownership/resource checks, registry persistence, workspace path/manager behavior, and stop-timeout query behavior. This harness does **not** compile the repository's actual full Podman client/runtime adapter or unrelated application packages.

A full repository clone could not be downloaded in this execution environment because DNS resolution for `github.com` / `codeload.github.com` failed. Therefore the actual repository-wide `go test ./...`, backend build, UI build, and live Podman daemon integration test were **not run**. Do not mark those checks as passed.

## Next action

The integration coordinator should review this branch, merge it into `feat/container-native-agent-rework`, then wire `container.Manager` into the task worker following the integration steps above. The next runtime validation should run the full repository tests and an end-to-end task create/start/stop/delete test against a rootless Podman daemon.
