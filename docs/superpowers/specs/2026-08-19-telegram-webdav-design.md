# Telegram WebDAV Netdisk Design

## 1. Overview

This document defines the first-phase design for a personal-use netdisk service with the following goals:

- Provide a `Flutter Web` management UI for browsing, uploading, and configuring storage.
- Provide a `Go` backend that exposes both a REST API and a WebDAV endpoint.
- Use a private Telegram channel as the physical storage backend.
- Maintain a virtual directory tree in the backend so folders can be represented even though Telegram stores files as flat messages.
- Route all upload and download traffic through the server so clients do not need direct Telegram access.
- Enforce a configurable local staging cache limit to avoid exhausting server disk space during relay operations.

The target user model for phase 1 is a single trusted user for personal use. Multi-user isolation, RBAC, and tenant separation are explicitly out of scope for this phase.

## 2. Product Scope

### In Scope

- Flutter Web control plane
- Manual file upload from the web UI
- WebDAV mount for upload, download, list, create directory, move, copy, and delete
- Telegram user-account-based storage driver
- File chunking for files larger than Telegram single-upload limits
- Virtual folder tree backed by database metadata
- Background job tracking and resumable upload workflow
- Local staging cache quota enforcement
- Streaming reconstruction of files during reads

### Out of Scope

- Multi-user accounts and per-user storage isolation
- Random-write or block-level file mutation
- Strong file locks across WebDAV clients
- Native desktop or mobile app
- Multiple storage backends in phase 1
- Deduplicated content store
- End-to-end encryption above Telegram transport
- Search indexing, thumbnails, media preview pipelines

## 3. Design Principles

- Telegram is treated as an object carrier, not as a filesystem.
- The database is the source of truth for directory structure and file visibility.
- A file is visible only after all chunks are uploaded successfully and metadata is committed.
- WebDAV and the web UI must share one virtual filesystem model.
- The system should be logically layered, even if phase 1 is deployed as a single process.
- Large files must be handled with bounded local disk usage and resumable jobs.

## 4. High-Level Architecture

The recommended architecture is a layered design with separate control-plane and data-plane responsibilities.

### 4.1 Components

#### Flutter Web UI

- Provides login/configuration workflow for the single-user setup.
- Supports browsing the virtual directory tree.
- Supports manual upload and job status visibility.
- Supports system configuration such as target Telegram channel, chunk size, and staging cache limit.

#### Go API Server

- Exposes REST endpoints for the Flutter Web UI.
- Handles configuration, filesystem actions, download requests, and job queries.
- Performs request validation and translates UI actions into virtual filesystem operations.

#### Go WebDAV Server

- Exposes a WebDAV endpoint for standard clients.
- Adapts WebDAV verbs to the same virtual filesystem service used by the REST API.
- Never writes directly to a local filesystem as the source of truth.

#### Virtual Filesystem Service

- Core domain service for directory, file, and job orchestration.
- Implements filesystem semantics on top of metadata plus Telegram object storage.
- Controls file visibility transitions and delete semantics.

#### Telegram Storage Driver

- Uses a Telegram user account session to read and write files in a private channel.
- Uploads file chunks as Telegram messages or media objects.
- Retrieves file content by Telegram message references.
- Handles retries, rate limiting, and Telegram-specific error classification.

#### Metadata Store

- Stores directory tree, logical file entries, chunk mappings, upload jobs, and system configuration.
- Serves as the authoritative state for list, move, delete, and ready/not-ready visibility.

#### Background Job Worker

- Runs upload, resume, delete cleanup, and cache cleanup workflows.
- Recovers unfinished jobs on restart.

### 4.2 Deployment Shape

Phase 1 should be deployed as one Go service process plus one SQLite database file, with the Flutter Web assets served either by the Go service or by a reverse proxy such as Nginx.

This keeps deployment simple while preserving internal boundaries so later extraction into separate services remains possible.

## 5. Data Model

### 5.1 directory

Represents a logical folder in the virtual tree.

Suggested fields:

- `id`
- `parent_id`
- `name`
- `path`
- `created_at`
- `updated_at`

Notes:

- `path` can be denormalized for fast lookup but should always be derived from parent relationships during repair workflows.
- Root should be modeled explicitly rather than inferred.

### 5.2 file_entry

Represents one logical file in the virtual filesystem.

Suggested fields:

- `id`
- `parent_id`
- `name`
- `size`
- `mime`
- `sha256`
- `status`
- `source`
- `created_at`
- `updated_at`
- `deleted_at`

Suggested `status` values:

- `pending`
- `uploading`
- `ready`
- `failed`
- `deleting`

Notes:

- Only `ready` entries are returned to WebDAV and normal file list views.
- `source` distinguishes uploads from `webdav` or `ui`.

### 5.3 file_chunk

Represents one chunk of a logical file stored in Telegram.

Suggested fields:

- `id`
- `file_id`
- `chunk_index`
- `chunk_size`
- `chunk_sha256`
- `telegram_chat_id`
- `telegram_message_id`
- `created_at`

Notes:

- `chunk_index` defines reconstruction order.
- `telegram_message_id` is mandatory for later reads and deletes.

### 5.4 upload_job

Represents the lifecycle of a backend upload task.

Suggested fields:

- `id`
- `file_id`
- `source`
- `stage`
- `retry_count`
- `last_error`
- `last_chunk_index`
- `created_at`
- `updated_at`

Suggested `stage` values:

- `staged`
- `chunking`
- `uploading`
- `committing`
- `done`
- `failed`

### 5.5 system_config

Stores single-user system configuration.

Suggested fields:

- `id`
- `telegram_session_blob`
- `telegram_target_chat_id`
- `default_chunk_size`
- `max_staging_bytes`
- `download_cache_ttl_seconds`
- `created_at`
- `updated_at`

### 5.6 cache_ledger

Tracks temporary local storage usage.

Suggested fields:

- `id`
- `job_id`
- `file_id`
- `reserved_bytes`
- `actual_bytes`
- `state`
- `expires_at`
- `created_at`
- `updated_at`

Suggested `state` values:

- `reserved`
- `active`
- `released`
- `expired`

## 6. File Visibility and Consistency Rules

- A logical file becomes visible only after every chunk upload succeeds and metadata commit succeeds.
- A partially uploaded file must never appear as a normal file in directory listings.
- WebDAV listings and REST listings must follow the same visibility rule.
- Delete operations should mark the logical object first, then perform Telegram cleanup asynchronously.
- Directory operations are metadata-first and should not trigger Telegram writes unless they affect file content.

## 7. Write and Read Flows

### 7.1 Web UI Upload Flow

1. The user uploads a file from the Flutter Web UI to the Go API.
2. The API requests staging capacity from the cache quota manager.
3. If insufficient local staging quota is available, the request is rejected before upload starts or before the upload is fully accepted, depending on transport behavior.
4. The uploaded data is written to a staging file on local disk.
5. The system creates a `file_entry` in `pending` state and an `upload_job`.
6. The background worker chunks the staged file according to configured chunk size.
7. Each chunk is uploaded to Telegram and its message reference is recorded.
8. After all chunks succeed, metadata is committed in one final transaction.
9. The file transitions to `ready`.
10. The staging file is deleted and the cache reservation is released.

### 7.2 WebDAV Upload Flow

1. A WebDAV client sends `PUT` or writes a temp file before `MOVE`.
2. The WebDAV adapter writes incoming bytes to staging storage, not to a persistent filesystem namespace.
3. On request completion or final rename commit, the adapter creates or finalizes an `upload_job`.
4. The file remains invisible until chunk upload and metadata commit finish.
5. Only then does the path become available in normal directory listings.

This matches the confirmed product rule that files become available only after upload completion.

### 7.3 Download Flow

1. The user or WebDAV client requests a logical file.
2. The virtual filesystem verifies that the file exists and is in `ready` state.
3. The chunk list is loaded in `chunk_index` order.
4. The Telegram driver fetches the content chunk by chunk.
5. The Go service streams reconstructed bytes back to the client.

Notes:

- Reconstruction should be streaming and should avoid buffering the entire file in memory.
- A short-lived download cache can be added later, but it is not required for phase 1.

### 7.4 Delete Flow

1. A delete request marks the logical file or directory as deleted in metadata.
2. The object disappears from normal listings.
3. A background cleanup task deletes associated Telegram messages.
4. Cleanup failures are retried separately and do not block the user-facing delete response.

## 8. Telegram Storage Strategy

### 8.1 Access Mode

Phase 1 uses a Telegram user account session, not the Bot API. This is required because the target deployment is a private personal channel workflow and needs more complete file handling behavior.

### 8.2 Channel Strategy

- Phase 1 should target one private Telegram channel per storage space.
- Files are stored as one or more Telegram messages in that channel.
- Telegram message ordering must not be treated as the filesystem index.

### 8.3 Chunking Strategy

- Files larger than the Telegram single-upload limit must be chunked automatically.
- Smaller files may also be chunked if a uniform implementation is preferred, but the default recommendation is:
  - single chunk for small files
  - multiple chunks only when needed

Each chunk must record:

- logical file association
- chunk order
- chunk checksum
- Telegram message reference

### 8.4 Retry and Rate Control

The Telegram driver must classify errors into:

- retryable transient errors
- rate limit/backoff conditions
- terminal failures requiring operator intervention

The worker should use bounded retries with exponential backoff and persist the last successful chunk index to support resume behavior.

## 9. WebDAV Behavior Contract

### 9.1 Supported Operations

Phase 1 should support:

- `PROPFIND`
- `GET`
- `PUT`
- `DELETE`
- `MKCOL`
- `MOVE`
- `COPY`

### 9.2 Compatibility Notes

- Some WebDAV clients upload to a temporary file and then rename it into place.
- The adapter should treat final rename as a possible commit boundary.
- The implementation should tolerate directory probes and repeated metadata reads.

### 9.3 Explicit Non-Goals

Phase 1 does not guarantee:

- strong distributed locks
- random-write semantics
- low-latency partial overwrite
- full fidelity with every WebDAV client implementation

The system is optimized for whole-file lifecycle operations.

## 10. API Surface

The API names below are a recommended initial shape. Exact request and response schemas can be finalized during implementation planning.

### 10.1 Configuration

- `POST /api/auth/telegram/session`
- `GET /api/config/storage`
- `PATCH /api/config/storage`

### 10.2 Filesystem

- `GET /api/fs/tree`
- `POST /api/fs/upload`
- `POST /api/fs/mkdir`
- `POST /api/fs/move`
- `POST /api/fs/delete`
- `GET /api/fs/file/:id/download`

### 10.3 Jobs

- `GET /api/jobs`
- `GET /api/jobs/:id`
- `POST /api/jobs/:id/retry`

### 10.4 WebDAV

- `GET|PROPFIND|PUT|DELETE|MOVE|COPY /webdav/*`

## 11. Cache and Capacity Control

The local server acts as a relay and staging buffer, so the staging quota is a core safety mechanism.

Rules:

- Every upload must reserve staging space before or during acceptance.
- Reservations must be tracked explicitly in `cache_ledger`.
- The system must release reservations on success, terminal failure cleanup, or expiry.
- If the requested upload would exceed `max_staging_bytes`, the system must reject it or queue it, depending on the chosen implementation. Phase 1 should prefer rejection over queuing for simplicity.
- Download-side caching is optional and must not consume unbounded disk.

## 12. Failure Recovery

The system must survive process restarts and transient Telegram failures without corrupting logical state.

Recovery rules:

- On startup, scan unfinished `upload_job` rows.
- Resume retryable jobs from the last confirmed chunk.
- Expire stale cache reservations and clean abandoned staging files.
- Never expose a file as `ready` if chunk metadata is incomplete.
- Keep delete cleanup as a separate compensating workflow.

## 13. Security and Operational Notes

### 13.1 Security

- Protect the web UI and API with at least a simple authenticated session.
- Treat Telegram session material as sensitive secret data.
- Avoid logging file contents or raw Telegram credentials.
- Use HTTPS in any internet-facing deployment.

### 13.2 Operational Constraints

- Telegram can apply rate limits or anti-abuse controls.
- WebDAV client behavior varies by platform.
- Browser upload UX for very large files may be weaker than WebDAV.
- SQLite is acceptable for phase 1 because the deployment model is single-user and low concurrency.

## 14. Recommended Module Layout

Suggested Go layout:

- `cmd/server`
- `internal/api`
- `internal/webdav`
- `internal/vfs`
- `internal/telegram`
- `internal/jobs`
- `internal/store`
- `internal/config`

Suggested frontend layout:

- `web/` for the Flutter Web project

## 15. Milestones

### M1: Telegram Storage Proof

- Establish Telegram user session
- Upload one logical file to the private channel
- Record Telegram message references
- Download and reconstruct the file successfully

### M2: Virtual Filesystem Core

- Implement directory and file metadata
- Support create/list/move/delete in metadata
- Implement ready-only visibility

### M3: Web UI Control Plane

- Build Flutter Web configuration pages
- Build tree browsing and manual upload
- Surface upload job status

### M4: WebDAV Integration

- Expose WebDAV endpoint
- Map basic filesystem verbs into the virtual filesystem
- Verify behavior with at least one target WebDAV client

### M5: Resilience and Cleanup

- Resume failed uploads
- Enforce staging quota
- Add stale staging cleanup
- Add Telegram delete cleanup retries

## 16. Open Decisions Deferred to Planning

The following items do not block this design, but should be made concrete in the implementation plan:

- Exact Telegram library and session management approach in Go
- Exact chunk size defaults and thresholds
- Exact authentication strategy for the single-user web UI
- Target WebDAV clients to optimize for first
- Whether Flutter Web assets are served by Go or a reverse proxy
- Whether small files bypass staging and stream directly into chunk assembly

## 17. Final Recommendation

Phase 1 should follow a logically layered architecture:

- Flutter Web as control plane
- Go API and WebDAV surface as access layer
- Virtual filesystem as the shared domain core
- Telegram driver as object carrier
- SQLite-backed metadata as the source of truth

This design is intentionally conservative. It prioritizes correctness, recoverability, and compatibility with Telegram's flat object model over aggressive optimization or premature multi-user features.
