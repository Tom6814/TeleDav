# Telegram WebDAV Netdisk

A single-user netdisk service that exposes a Go REST API, a Go WebDAV endpoint,
and a Flutter Web control plane. The virtual filesystem is stored in SQLite,
while file chunks are intended to be relayed into a private Telegram channel.

## Components

- `cmd/server`: Go server entry point
- `internal/api`: REST routes for login, config, filesystem, and jobs
- `internal/webdav`: WebDAV adapter
- `internal/vfs`: ready-only virtual filesystem view
- `internal/jobs`: staging quota, chunking, upload workflow
- `internal/store`: SQLite schema and repository
- `internal/telegram`: Telegram client abstraction and gotd placeholder client
- `web/`: Flutter Web control plane

## Local Development

1. Set `APP_PASSWORD`, `APP_DB_PATH`, `APP_STAGING_DIR`, and `APP_WEB_DIR`.
2. Run `go test ./...`.
3. Build the web app with `make web-build`.
4. Start the server with `go run ./cmd/server`.
5. Open `http://localhost:8080/`.

## Suggested Environment

```bash
export APP_PASSWORD=secret
export APP_DB_PATH=data/app.db
export APP_STAGING_DIR=data/staging
export APP_WEB_DIR=web/build/web
```

## Phase 1 Validation

- Log in through the web UI.
- Create directories from the web UI or REST API.
- Upload a small file through the web UI or `/api/fs/upload`.
- Save storage settings from the web UI and restart the server to verify persisted values are applied when env vars are not overriding them.
- Retry a failed upload job from the Jobs tab or `POST /api/jobs/:id/retry`.
- Mount `/webdav/` with Cyberduck or `rclone`.
- Verify only `ready` files appear in listings.
- Restart the server and re-check pending jobs.

## Web UI Capabilities

- Login with the single-user password.
- Browse the virtual directory tree by directory.
- Create subdirectories in the current folder.
- Upload files into the current folder.
- Inspect and retry failed jobs.
- Edit persisted storage settings, including Telegram target chat ID and session blob.

## Current Limits

- Telegram upload, download, and delete are wired through an interface and a
  minimal `gotd` client shell, but full MTProto media transfer is still
  dependent on real Telegram credentials and session material at runtime.
- Flutter CLI is required to build the web frontend locally.
