# Quick Start

Get the go-mcp Web UI + Podman sandbox manager running in a few minutes.

> **Podman only.** This project talks exclusively to the **rootless Podman** socket.
> No Docker daemon is required or used.

## Prerequisites

| Tool | Purpose | Check |
|---|---|---|
| Go 1.21+ | Server build | `go version` |
| Bun (or npm) | Frontend build | `bun --version` |
| Podman 4.x (rootless) | Sandboxes | `podman --version` |
| cloudflared (optional) | Public HTTPS tunnel | `cloudflared --version` |

## 1. Verify rootless Podman

Run these checks — all should pass without `sudo`:

```bash
# Must print rootless=true
podman info --format 'rootless={{.Host.Security.Rootless}}'

# Your user must have subuid/subgid ranges
grep "$(id -un)" /etc/subuid /etc/subgid
# e.g.  mfk01:165536:65536

# Rootless API socket (used by the web server)
ls -l $XDG_RUNTIME_DIR/podman/podman.sock
# e.g.  /run/user/1001/podman/podman.sock

# Smoke test: must print podman-rootless-ok
podman run --rm docker.io/library/alpine:latest echo podman-rootless-ok
```

Expected on a healthy host:

```
rootless=true
mfk01:165536:65536        (subuid)
mfk01:165536:65536        (subgid)
srw-rw---- ... /run/user/1001/podman/podman.sock
podman-rootless-ok
```

### Start / persist the rootless socket

```bash
# Start now (active for the current session)
systemctl --user enable --now podman.socket

# Keep the socket alive after logout (needed if the server runs long-term)
loginctl enable-linger "$(id -un)"
```

### If something is wrong

- `rootless=false` → you are running as root; re-run as your normal user.
- Socket missing → `systemctl --user start podman.socket`.
- `subuid` missing → `sudo useradd --user --subuid-start 165536 "$(id -un)"` (or edit `/etc/subuid` + `/etc/subgid`), then re-login.
- Never mix in Docker: the server only connects to `podman.sock`.

## 2. Build the frontend

```bash
cd web
bun install          # or: npm install
bun run build        # outputs to web/dist
cd ..
```

## 3. Build and run the server

```bash
go build -o go-mcp-server ./cmd/go-mcp-server
./go-mcp-server --config config/web_config.yaml
```

Open **http://localhost:8080** and log in:

- **Admin**: `admin` / `admin123`
- **Viewer**: `viewer` / `viewer123`

Override secrets via environment variables (`ADMIN_PASSWORD`, `VIEWER_PASSWORD`)
or point the config at another socket (`PODMAN_SOCKET`).

### Rootless socket auto-detection

`config/web_config.yaml` defaults to `/run/podman/podman.sock` (rootful path).
If that path doesn't exist, the server automatically falls back to the rootless
socket `$XDG_RUNTIME_DIR/podman/podman.sock` — no config edit needed on a
rootless-only host. To pin it explicitly:

```yaml
podman:
  socket: "/run/user/1001/podman/podman.sock"
```

## 4. (Optional) Public HTTPS via Cloudflare Tunnel

```bash
cloudflared tunnel create econlap
cloudflared tunnel route dns econlap econlap.faaris.tech
cloudflared tunnel --config ~/.cloudflared/config.yml run econlap
```

The tunnel serves the UI at **https://econlap.faaris.tech** (HTTP 200, verified
end-to-end through Cloudflare).

> **Note:** Cloudflare Universal SSL only covers the zone apex and **one** level
> of subdomain (`x.faaris.tech`). Deeper names like `a.b.faaris.tech` fail the
> TLS handshake unless their sub-zone is delegated as its own Cloudflare zone or
> Advanced Certificate Manager is enabled.

## 5. Camera pass-through (V4L2)

Detected host cameras show up under **Devices** in the UI and can be attached to
sandboxes at creation time (e.g. `/dev/video0`, `/dev/video1`):

```bash
ls /sys/class/video4linux/*/name
```

## Run tests

```bash
go test ./...
```
