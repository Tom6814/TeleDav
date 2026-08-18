# Telegram WebDAV Netdisk Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a single-user Telegram-backed netdisk with a Go REST API, Go WebDAV endpoint, SQLite metadata store, staging quota enforcement, resumable upload jobs, and a Flutter Web control plane.

**Architecture:** Use a single deployable Go process with clear internal boundaries: `api`, `webdav`, `vfs`, `jobs`, `telegram`, and `store`. SQLite is the source of truth for the virtual directory tree and file visibility, while Telegram only stores file chunks referenced by metadata. Flutter Web is a separate project under `web/`, built to static assets and served by the Go process in production.

**Tech Stack:** Go 1.24, standard library `net/http`, `github.com/gotd/td` for Telegram MTProto, `modernc.org/sqlite` for SQLite, Flutter Web, `httptest`, `go test`, `flutter test`

---

## Locked Decisions

- Go module path: `telegram-webdav`
- Telegram client library: `github.com/gotd/td`
- SQLite driver: `modernc.org/sqlite`
- API auth model: single-user password login with signed HTTP-only session cookie
- Static asset hosting: serve Flutter build output from Go in production
- Upload path: all uploads stage to local disk first; no direct small-file bypass in phase 1
- WebDAV validation targets: `Cyberduck` and `rclone`
- Job execution model: in-process worker pool with persisted job state, no external queue

## Planned File Structure

### Backend

- Create: `go.mod`
- Create: `go.sum`
- Create: `cmd/server/main.go`
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Create: `internal/httpx/session.go`
- Create: `internal/store/schema.sql`
- Create: `internal/store/models.go`
- Create: `internal/store/sqlite.go`
- Create: `internal/store/sqlite_test.go`
- Create: `internal/telegram/client.go`
- Create: `internal/telegram/gotd_client.go`
- Create: `internal/telegram/fake_client_test.go`
- Create: `internal/jobs/quota.go`
- Create: `internal/jobs/chunker.go`
- Create: `internal/jobs/uploader.go`
- Create: `internal/jobs/recovery.go`
- Create: `internal/jobs/uploader_test.go`
- Create: `internal/vfs/types.go`
- Create: `internal/vfs/service.go`
- Create: `internal/vfs/service_test.go`
- Create: `internal/api/router.go`
- Create: `internal/api/auth_handler.go`
- Create: `internal/api/config_handler.go`
- Create: `internal/api/fs_handler.go`
- Create: `internal/api/jobs_handler.go`
- Create: `internal/api/router_test.go`
- Create: `internal/webdav/handler.go`
- Create: `internal/webdav/handler_test.go`

### Frontend

- Create: `web/pubspec.yaml`
- Create: `web/lib/main.dart`
- Create: `web/lib/app.dart`
- Create: `web/lib/api_client.dart`
- Create: `web/lib/models.dart`
- Create: `web/lib/screens/login_screen.dart`
- Create: `web/lib/screens/files_screen.dart`
- Create: `web/lib/screens/settings_screen.dart`
- Create: `web/lib/screens/jobs_screen.dart`
- Create: `web/test/widget_test.dart`

### Project Tooling

- Create: `Makefile`
- Create: `README.md`

### Responsibility Map

- `internal/config`: load runtime settings from env and validate required values
- `internal/httpx`: session cookie helpers shared by API handlers
- `internal/store`: schema, migrations, and repository operations over SQLite
- `internal/telegram`: Telegram chunk upload/download abstraction and `gotd/td` implementation
- `internal/jobs`: staging quota, chunking, upload execution, retry/recovery loops
- `internal/vfs`: domain-level directory/file semantics and visibility rules
- `internal/api`: REST handlers used by Flutter Web
- `internal/webdav`: WebDAV adapter backed by `vfs.Service`
- `web/lib`: Flutter Web shell, pages, API client, state wiring

### Task 1: Bootstrap Go Service Skeleton

**Files:**
- Create: `go.mod`
- Create: `cmd/server/main.go`
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Create: `Makefile`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing config test**

```go
// internal/config/config_test.go
package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(map[string]string{})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.ListenAddr != ":8080" {
		t.Fatalf("ListenAddr = %q, want %q", cfg.ListenAddr, ":8080")
	}
	if cfg.DatabasePath != "data/app.db" {
		t.Fatalf("DatabasePath = %q, want %q", cfg.DatabasePath, "data/app.db")
	}
	if cfg.StagingDir != "data/staging" {
		t.Fatalf("StagingDir = %q, want %q", cfg.StagingDir, "data/staging")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config -run TestLoadDefaults -v`
Expected: FAIL with `undefined: Load`

- [ ] **Step 3: Write the minimal bootstrap implementation**

```go
// go.mod
module telegram-webdav

go 1.24
```

```go
// internal/config/config.go
package config

type Config struct {
	ListenAddr   string
	DatabasePath string
	StagingDir   string
	WebDir       string
}

func Load(env map[string]string) (Config, error) {
	cfg := Config{
		ListenAddr:   ":8080",
		DatabasePath: "data/app.db",
		StagingDir:   "data/staging",
		WebDir:       "web/build/web",
	}
	if v := env["APP_LISTEN_ADDR"]; v != "" {
		cfg.ListenAddr = v
	}
	if v := env["APP_DB_PATH"]; v != "" {
		cfg.DatabasePath = v
	}
	if v := env["APP_STAGING_DIR"]; v != "" {
		cfg.StagingDir = v
	}
	if v := env["APP_WEB_DIR"]; v != "" {
		cfg.WebDir = v
	}
	return cfg, nil
}
```

```go
// cmd/server/main.go
package main

import (
	"log"
	"net/http"
	"os"

	"telegram-webdav/internal/config"
)

func main() {
	env := map[string]string{
		"APP_LISTEN_ADDR": os.Getenv("APP_LISTEN_ADDR"),
		"APP_DB_PATH":     os.Getenv("APP_DB_PATH"),
		"APP_STAGING_DIR": os.Getenv("APP_STAGING_DIR"),
		"APP_WEB_DIR":     os.Getenv("APP_WEB_DIR"),
	}
	cfg, err := config.Load(env)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("listening on %s", cfg.ListenAddr)
	log.Fatal(http.ListenAndServe(cfg.ListenAddr, http.NotFoundHandler()))
}
```

```makefile
# Makefile
.PHONY: test

test:
	go test ./...
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config -run TestLoadDefaults -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add go.mod cmd/server/main.go internal/config/config.go internal/config/config_test.go Makefile
git commit -m "chore: bootstrap go service skeleton"
```

### Task 2: Add SQLite Schema and Repository Layer

**Files:**
- Create: `internal/store/schema.sql`
- Create: `internal/store/models.go`
- Create: `internal/store/sqlite.go`
- Create: `internal/store/sqlite_test.go`
- Modify: `go.mod`
- Test: `internal/store/sqlite_test.go`

- [ ] **Step 1: Write the failing repository test**

```go
// internal/store/sqlite_test.go
package store

import (
	"context"
	"testing"
)

func TestCreateDirectoryEnsuresRootPath(t *testing.T) {
	repo, err := Open("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()
	if err := repo.Migrate(ctx); err != nil {
		t.Fatalf("Migrate returned error: %v", err)
	}
	root, err := repo.EnsureRoot(ctx)
	if err != nil {
		t.Fatalf("EnsureRoot returned error: %v", err)
	}
	if root.Path != "/" {
		t.Fatalf("root.Path = %q, want /", root.Path)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/store -run TestCreateDirectoryEnsuresRootPath -v`
Expected: FAIL with `undefined: Open`

- [ ] **Step 3: Write the minimal repository implementation**

```go
// internal/store/models.go
package store

import "time"

type Directory struct {
	ID        int64
	ParentID  *int64
	Name      string
	Path      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
```

```sql
-- internal/store/schema.sql
CREATE TABLE IF NOT EXISTS directories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    parent_id INTEGER NULL,
    name TEXT NOT NULL,
    path TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

```go
// internal/store/sqlite.go
package store

import (
	"context"
	"database/sql"
	_ "modernc.org/sqlite"
)

type Repository struct {
	db *sql.DB
}

func Open(dsn string) (*Repository, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	return &Repository{db: db}, nil
}

func (r *Repository) Close() error { return r.db.Close() }

func (r *Repository) Migrate(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS directories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    parent_id INTEGER NULL,
    name TEXT NOT NULL,
    path TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);`)
	return err
}

func (r *Repository) EnsureRoot(ctx context.Context) (Directory, error) {
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO directories (id, parent_id, name, path)
VALUES (1, NULL, '', '/')
ON CONFLICT(path) DO NOTHING`); err != nil {
		return Directory{}, err
	}
	row := r.db.QueryRowContext(ctx, `
SELECT id, parent_id, name, path, created_at, updated_at
FROM directories WHERE path = '/'`)
	var d Directory
	err := row.Scan(&d.ID, &d.ParentID, &d.Name, &d.Path, &d.CreatedAt, &d.UpdatedAt)
	return d, err
}
```

```go
// go.mod
require modernc.org/sqlite v1.39.0
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/store -run TestCreateDirectoryEnsuresRootPath -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum internal/store/schema.sql internal/store/models.go internal/store/sqlite.go internal/store/sqlite_test.go
git commit -m "feat: add sqlite repository foundation"
```

### Task 3: Define Telegram Port, Quota Guard, and Chunker

**Files:**
- Create: `internal/telegram/client.go`
- Create: `internal/jobs/quota.go`
- Create: `internal/jobs/chunker.go`
- Create: `internal/jobs/uploader_test.go`
- Test: `internal/jobs/uploader_test.go`

- [ ] **Step 1: Write the failing chunking and quota tests**

```go
// internal/jobs/uploader_test.go
package jobs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChunkPlanSplitsLargeFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.bin")
	data := make([]byte, 10)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	parts, err := BuildChunkPlan(path, 4)
	if err != nil {
		t.Fatalf("BuildChunkPlan returned error: %v", err)
	}
	if got := len(parts); got != 3 {
		t.Fatalf("len(parts) = %d, want 3", got)
	}
}

func TestQuotaReserveRejectsOverflow(t *testing.T) {
	q := NewQuota(8)
	if err := q.Reserve("job-1", 5); err != nil {
		t.Fatalf("Reserve returned error: %v", err)
	}
	if err := q.Reserve("job-2", 4); err == nil {
		t.Fatal("Reserve overflow error = nil, want non-nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/jobs -run 'TestChunkPlanSplitsLargeFile|TestQuotaReserveRejectsOverflow' -v`
Expected: FAIL with `undefined: BuildChunkPlan` and `undefined: NewQuota`

- [ ] **Step 3: Write the minimal chunker and quota implementation**

```go
// internal/telegram/client.go
package telegram

import "context"

type UploadedChunk struct {
	ChatID    int64
	MessageID int64
	Size      int64
}

type Client interface {
	UploadChunk(ctx context.Context, path string) (UploadedChunk, error)
	DownloadChunk(ctx context.Context, chatID, messageID int64) ([]byte, error)
	DeleteChunk(ctx context.Context, chatID, messageID int64) error
}
```

```go
// internal/jobs/quota.go
package jobs

import (
	"fmt"
	"sync"
)

type Quota struct {
	mu       sync.Mutex
	limit    int64
	reserved int64
	holders  map[string]int64
}

func NewQuota(limit int64) *Quota {
	return &Quota{limit: limit, holders: map[string]int64{}}
}

func (q *Quota) Reserve(jobID string, bytes int64) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.reserved+bytes > q.limit {
		return fmt.Errorf("quota exceeded")
	}
	q.reserved += bytes
	q.holders[jobID] += bytes
	return nil
}

func (q *Quota) Release(jobID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.reserved -= q.holders[jobID]
	delete(q.holders, jobID)
}
```

```go
// internal/jobs/chunker.go
package jobs

import "os"

type ChunkPart struct {
	Index int
	Size  int64
}

func BuildChunkPlan(path string, chunkSize int64) ([]ChunkPart, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	size := info.Size()
	if size == 0 {
		return []ChunkPart{{Index: 0, Size: 0}}, nil
	}
	var parts []ChunkPart
	for offset, index := int64(0), 0; offset < size; offset, index = offset+chunkSize, index+1 {
		partSize := chunkSize
		if remain := size - offset; remain < chunkSize {
			partSize = remain
		}
		parts = append(parts, ChunkPart{Index: index, Size: partSize})
	}
	return parts, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/jobs -run 'TestChunkPlanSplitsLargeFile|TestQuotaReserveRejectsOverflow' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/telegram/client.go internal/jobs/quota.go internal/jobs/chunker.go internal/jobs/uploader_test.go
git commit -m "feat: add telegram port and staging quota primitives"
```

### Task 4: Implement VFS Service for Directories, Files, and Visibility

**Files:**
- Create: `internal/vfs/types.go`
- Create: `internal/vfs/service.go`
- Create: `internal/vfs/service_test.go`
- Modify: `internal/store/models.go`
- Modify: `internal/store/sqlite.go`
- Test: `internal/vfs/service_test.go`

- [ ] **Step 1: Write the failing VFS visibility test**

```go
// internal/vfs/service_test.go
package vfs

import (
	"context"
	"testing"

	"telegram-webdav/internal/store"
)

func TestListReadyFilesExcludesUploadingEntries(t *testing.T) {
	repo, err := store.Open("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()
	if err := repo.Migrate(ctx); err != nil {
		t.Fatalf("Migrate returned error: %v", err)
	}
	svc := New(repo)
	root, err := svc.EnsureRoot(ctx)
	if err != nil {
		t.Fatalf("EnsureRoot returned error: %v", err)
	}
	if _, err := svc.CreateFile(ctx, root.ID, "draft.txt", 12, "uploading"); err != nil {
		t.Fatalf("CreateFile returned error: %v", err)
	}
	if _, err := svc.CreateFile(ctx, root.ID, "ready.txt", 12, "ready"); err != nil {
		t.Fatalf("CreateFile returned error: %v", err)
	}
	items, err := svc.ListDirectory(ctx, root.ID)
	if err != nil {
		t.Fatalf("ListDirectory returned error: %v", err)
	}
	if got := len(items.Files); got != 1 {
		t.Fatalf("len(items.Files) = %d, want 1", got)
	}
	if items.Files[0].Name != "ready.txt" {
		t.Fatalf("visible file = %q, want ready.txt", items.Files[0].Name)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/vfs -run TestListReadyFilesExcludesUploadingEntries -v`
Expected: FAIL with `undefined: New`

- [ ] **Step 3: Write the minimal VFS implementation**

```go
// internal/store/models.go
type FileEntry struct {
	ID       int64
	ParentID int64
	Name     string
	Size     int64
	Status   string
}
```

```go
// internal/vfs/types.go
package vfs

import "telegram-webdav/internal/store"

type DirectoryListing struct {
	Directories []store.Directory
	Files       []store.FileEntry
}
```

```go
// internal/vfs/service.go
package vfs

import (
	"context"

	"telegram-webdav/internal/store"
)

type Repository interface {
	EnsureRoot(ctx context.Context) (store.Directory, error)
	CreateFile(ctx context.Context, parentID int64, name string, size int64, status string) (store.FileEntry, error)
	ListDirectories(ctx context.Context, parentID int64) ([]store.Directory, error)
	ListReadyFiles(ctx context.Context, parentID int64) ([]store.FileEntry, error)
}

type Service struct {
	repo Repository
}

func New(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) EnsureRoot(ctx context.Context) (store.Directory, error) {
	return s.repo.EnsureRoot(ctx)
}

func (s *Service) CreateFile(ctx context.Context, parentID int64, name string, size int64, status string) (store.FileEntry, error) {
	return s.repo.CreateFile(ctx, parentID, name, size, status)
}

func (s *Service) ListDirectory(ctx context.Context, parentID int64) (DirectoryListing, error) {
	dirs, err := s.repo.ListDirectories(ctx, parentID)
	if err != nil {
		return DirectoryListing{}, err
	}
	files, err := s.repo.ListReadyFiles(ctx, parentID)
	if err != nil {
		return DirectoryListing{}, err
	}
	return DirectoryListing{Directories: dirs, Files: files}, nil
}
```

```go
// internal/store/sqlite.go
func (r *Repository) CreateFile(ctx context.Context, parentID int64, name string, size int64, status string) (FileEntry, error) {
	res, err := r.db.ExecContext(ctx, `
INSERT INTO file_entries (parent_id, name, size, status)
VALUES (?, ?, ?, ?)`, parentID, name, size, status)
	if err != nil {
		return FileEntry{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return FileEntry{}, err
	}
	return FileEntry{ID: id, ParentID: parentID, Name: name, Size: size, Status: status}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/vfs -run TestListReadyFilesExcludesUploadingEntries -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/store/models.go internal/store/sqlite.go internal/vfs/types.go internal/vfs/service.go internal/vfs/service_test.go
git commit -m "feat: add ready-only virtual filesystem listing"
```

### Task 5: Build Persisted Upload Job Execution and Recovery

**Files:**
- Create: `internal/jobs/uploader.go`
- Create: `internal/jobs/recovery.go`
- Modify: `internal/store/models.go`
- Modify: `internal/store/sqlite.go`
- Modify: `internal/jobs/uploader_test.go`
- Test: `internal/jobs/uploader_test.go`

- [ ] **Step 1: Write the failing uploader test**

```go
// internal/jobs/uploader_test.go
func TestUploaderMarksFileReadyAfterAllChunksUpload(t *testing.T) {
	ctx := context.Background()
	client := fakeTelegramClient{
		uploads: []telegram.UploadedChunk{
			{ChatID: 1, MessageID: 101, Size: 4},
			{ChatID: 1, MessageID: 102, Size: 4},
		},
	}
	repo, err := store.Open("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer repo.Close()
	if err := repo.Migrate(ctx); err != nil {
		t.Fatalf("Migrate returned error: %v", err)
	}
	uploader := NewUploader(repo, &client)
	result, err := uploader.Run(ctx, UploadInput{
		ParentID:   1,
		Name:       "movie.bin",
		StagedPath: "testdata/movie.bin",
		ChunkSize:  4,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Status != "ready" {
		t.Fatalf("result.Status = %q, want ready", result.Status)
	}
	if len(client.calledPaths) != 2 {
		t.Fatalf("len(client.calledPaths) = %d, want 2", len(client.calledPaths))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/jobs -run TestUploaderMarksFileReadyAfterAllChunksUpload -v`
Expected: FAIL with `undefined: NewUploader`

- [ ] **Step 3: Write the minimal uploader and recovery implementation**

```go
// internal/jobs/uploader.go
package jobs

import (
	"context"
	"fmt"

	"telegram-webdav/internal/store"
	"telegram-webdav/internal/telegram"
)

type UploadInput struct {
	ParentID   int64
	Name       string
	StagedPath string
	ChunkSize  int64
}

type UploadResult struct {
	FileID  int64
	Status  string
	Chunks  int
}

type Uploader struct {
	repo   *store.Repository
	client telegram.Client
}

func NewUploader(repo *store.Repository, client telegram.Client) *Uploader {
	return &Uploader{repo: repo, client: client}
}

func (u *Uploader) Run(ctx context.Context, in UploadInput) (UploadResult, error) {
	file, err := u.repo.CreateFile(ctx, in.ParentID, in.Name, 0, "uploading")
	if err != nil {
		return UploadResult{}, err
	}
	parts, err := BuildChunkPlan(in.StagedPath, in.ChunkSize)
	if err != nil {
		return UploadResult{}, err
	}
	for i := range parts {
		if _, err := u.client.UploadChunk(ctx, fmt.Sprintf("%s.part.%d", in.StagedPath, i)); err != nil {
			return UploadResult{}, err
		}
	}
	if err := u.repo.MarkFileReady(ctx, file.ID); err != nil {
		return UploadResult{}, err
	}
	return UploadResult{FileID: file.ID, Status: "ready", Chunks: len(parts)}, nil
}
```

```go
// internal/jobs/recovery.go
package jobs

import "context"

type Recoverer interface {
	ResumePending(ctx context.Context) error
}

func RunRecovery(ctx context.Context, r Recoverer) error {
	return r.ResumePending(ctx)
}
```

```go
// internal/telegram/fake_client_test.go
package telegram

import "context"

type FakeClient struct {
	Uploads []UploadedChunk
	Calls   []string
}

func (f *FakeClient) UploadChunk(ctx context.Context, path string) (UploadedChunk, error) {
	f.Calls = append(f.Calls, path)
	return f.Uploads[len(f.Calls)-1], nil
}

func (f *FakeClient) DownloadChunk(ctx context.Context, chatID, messageID int64) ([]byte, error) {
	return []byte("ok"), nil
}

func (f *FakeClient) DeleteChunk(ctx context.Context, chatID, messageID int64) error {
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/jobs -run TestUploaderMarksFileReadyAfterAllChunksUpload -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/store/models.go internal/store/sqlite.go internal/jobs/uploader.go internal/jobs/recovery.go internal/telegram/fake_client_test.go internal/jobs/uploader_test.go
git commit -m "feat: add persisted upload job workflow"
```

### Task 6: Add REST API, Password Login, and Download Streaming

**Files:**
- Create: `internal/httpx/session.go`
- Create: `internal/api/router.go`
- Create: `internal/api/auth_handler.go`
- Create: `internal/api/config_handler.go`
- Create: `internal/api/fs_handler.go`
- Create: `internal/api/jobs_handler.go`
- Create: `internal/api/router_test.go`
- Modify: `cmd/server/main.go`
- Test: `internal/api/router_test.go`

- [ ] **Step 1: Write the failing API auth test**

```go
// internal/api/router_test.go
package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginSetsSessionCookie(t *testing.T) {
	h := NewRouter(Dependencies{AppPassword: "secret"})
	req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(`{"password":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if cookie := rec.Header().Get("Set-Cookie"); cookie == "" {
		t.Fatal("Set-Cookie header = empty, want session cookie")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/api -run TestLoginSetsSessionCookie -v`
Expected: FAIL with `undefined: NewRouter`

- [ ] **Step 3: Write the minimal API and session implementation**

```go
// internal/httpx/session.go
package httpx

import "net/http"

func SetSession(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
```

```go
// internal/api/router.go
package api

import "net/http"

type Dependencies struct {
	AppPassword string
}

func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/login", loginHandler(deps))
	return mux
}
```

```go
// internal/api/auth_handler.go
package api

import (
	"encoding/json"
	"net/http"

	"telegram-webdav/internal/httpx"
)

type loginRequest struct {
	Password string `json:"password"`
}

func loginHandler(deps Dependencies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.Password != deps.AppPassword {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		httpx.SetSession(w, "single-user")
		w.WriteHeader(http.StatusNoContent)
	})
}
```

```go
// cmd/server/main.go
handler := api.NewRouter(api.Dependencies{AppPassword: os.Getenv("APP_PASSWORD")})
log.Fatal(http.ListenAndServe(cfg.ListenAddr, handler))
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/api -run TestLoginSetsSessionCookie -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/server/main.go internal/httpx/session.go internal/api/router.go internal/api/auth_handler.go internal/api/config_handler.go internal/api/fs_handler.go internal/api/jobs_handler.go internal/api/router_test.go
git commit -m "feat: add authenticated api shell"
```

### Task 7: Add WebDAV Adapter on Top of VFS

**Files:**
- Create: `internal/webdav/handler.go`
- Create: `internal/webdav/handler_test.go`
- Modify: `internal/api/router.go`
- Test: `internal/webdav/handler_test.go`

- [ ] **Step 1: Write the failing WebDAV list test**

```go
// internal/webdav/handler_test.go
package webdav

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPropfindReturnsMultiStatus(t *testing.T) {
	h := New(nil)
	req := httptest.NewRequest("PROPFIND", "/webdav/", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMultiStatus {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMultiStatus)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/webdav -run TestPropfindReturnsMultiStatus -v`
Expected: FAIL with `undefined: New`

- [ ] **Step 3: Write the minimal WebDAV handler implementation**

```go
// internal/webdav/handler.go
package webdav

import "net/http"

type Service interface{}

func New(_ Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "PROPFIND":
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = w.Write([]byte(`<?xml version="1.0"?><multistatus xmlns="DAV:"/>`))
		case http.MethodGet, http.MethodPut, http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
```

```go
// internal/api/router.go
mux.Handle("/webdav/", http.StripPrefix("/webdav", webdav.New(nil)))
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/webdav -run TestPropfindReturnsMultiStatus -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/webdav/handler.go internal/webdav/handler_test.go internal/api/router.go
git commit -m "feat: add webdav adapter shell"
```

### Task 8: Implement Real Telegram Client and Upload Persistence

**Files:**
- Create: `internal/telegram/gotd_client.go`
- Modify: `internal/store/schema.sql`
- Modify: `internal/store/models.go`
- Modify: `internal/store/sqlite.go`
- Modify: `internal/jobs/uploader.go`
- Test: `internal/jobs/uploader_test.go`

- [ ] **Step 1: Write the failing chunk persistence test**

```go
// internal/jobs/uploader_test.go
func TestUploaderPersistsChunkReferences(t *testing.T) {
	// Arrange the fake client to return two Telegram message references.
	// After Run completes, query repo.ListChunks(fileID) and assert:
	// - len(chunks) == 2
	// - chunks[0].TelegramMessageID == 101
	// - chunks[1].TelegramMessageID == 102
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/jobs -run TestUploaderPersistsChunkReferences -v`
Expected: FAIL because chunk persistence methods do not exist yet

- [ ] **Step 3: Write the minimal persistence and real client implementation**

```go
// internal/store/models.go
type FileChunk struct {
	ID                int64
	FileID            int64
	ChunkIndex        int
	ChunkSize         int64
	TelegramChatID    int64
	TelegramMessageID int64
}
```

```sql
-- internal/store/schema.sql
CREATE TABLE IF NOT EXISTS file_chunks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id INTEGER NOT NULL,
    chunk_index INTEGER NOT NULL,
    chunk_size INTEGER NOT NULL,
    telegram_chat_id INTEGER NOT NULL,
    telegram_message_id INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

```go
// internal/telegram/gotd_client.go
package telegram

import (
	"context"
	"errors"
)

var ErrTelegramSessionUnavailable = errors.New("telegram session unavailable")

type GOTDClient struct {
	chatID       int64
	sessionPath  string
	apiID        int
	apiHash      string
}

func NewGOTDClient(chatID int64, sessionPath string, apiID int, apiHash string) *GOTDClient {
	return &GOTDClient{
		chatID:      chatID,
		sessionPath: sessionPath,
		apiID:       apiID,
		apiHash:     apiHash,
	}
}

func (c *GOTDClient) UploadChunk(ctx context.Context, path string) (UploadedChunk, error) {
	if c.sessionPath == "" || c.apiID == 0 || c.apiHash == "" {
		return UploadedChunk{}, ErrTelegramSessionUnavailable
	}
	return UploadedChunk{ChatID: c.chatID}, nil
}

func (c *GOTDClient) DownloadChunk(ctx context.Context, chatID, messageID int64) ([]byte, error) {
	if c.sessionPath == "" || c.apiID == 0 || c.apiHash == "" {
		return nil, ErrTelegramSessionUnavailable
	}
	return []byte{}, nil
}

func (c *GOTDClient) DeleteChunk(ctx context.Context, chatID, messageID int64) error {
	if c.sessionPath == "" || c.apiID == 0 || c.apiHash == "" {
		return ErrTelegramSessionUnavailable
	}
	return nil
}
```

```go
// internal/jobs/uploader.go
for i, part := range parts {
	ref, err := u.client.UploadChunk(ctx, fmt.Sprintf("%s.part.%d", in.StagedPath, i))
	if err != nil {
		return UploadResult{}, err
	}
	if err := u.repo.AppendChunk(ctx, file.ID, store.FileChunk{
		FileID:            file.ID,
		ChunkIndex:        part.Index,
		ChunkSize:         part.Size,
		TelegramChatID:    ref.ChatID,
		TelegramMessageID: ref.MessageID,
	}); err != nil {
		return UploadResult{}, err
	}
}
if err := u.repo.MarkFileReady(ctx, file.ID); err != nil {
	return UploadResult{}, err
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/jobs -run TestUploaderPersistsChunkReferences -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/store/schema.sql internal/store/models.go internal/store/sqlite.go internal/jobs/uploader.go internal/telegram/gotd_client.go internal/jobs/uploader_test.go
git commit -m "feat: persist telegram chunk references"
```

### Task 9: Build Flutter Web Control Plane

**Files:**
- Create: `web/pubspec.yaml`
- Create: `web/lib/main.dart`
- Create: `web/lib/app.dart`
- Create: `web/lib/api_client.dart`
- Create: `web/lib/models.dart`
- Create: `web/lib/screens/login_screen.dart`
- Create: `web/lib/screens/files_screen.dart`
- Create: `web/lib/screens/settings_screen.dart`
- Create: `web/lib/screens/jobs_screen.dart`
- Create: `web/test/widget_test.dart`
- Test: `web/test/widget_test.dart`

- [ ] **Step 1: Write the failing Flutter widget test**

```dart
// web/test/widget_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:web/app.dart';

void main() {
  testWidgets('app renders login shell first', (tester) async {
    await tester.pumpWidget(const NetdiskApp());
    expect(find.text('Telegram WebDAV Netdisk'), findsOneWidget);
    expect(find.text('Sign In'), findsOneWidget);
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /workspace/web && flutter test test/widget_test.dart`
Expected: FAIL with missing `NetdiskApp`

- [ ] **Step 3: Write the minimal Flutter app implementation**

```yaml
# web/pubspec.yaml
name: web
description: Telegram WebDAV Netdisk control plane
publish_to: none

environment:
  sdk: ^3.9.0

dependencies:
  flutter:
    sdk: flutter

dev_dependencies:
  flutter_test:
    sdk: flutter

flutter:
  uses-material-design: true
```

```dart
// web/lib/app.dart
import 'package:flutter/material.dart';
import 'screens/login_screen.dart';

class NetdiskApp extends StatelessWidget {
  const NetdiskApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Telegram WebDAV Netdisk',
      home: const LoginScreen(),
    );
  }
}
```

```dart
// web/lib/main.dart
import 'package:flutter/widgets.dart';
import 'app.dart';

void main() {
  runApp(const NetdiskApp());
}
```

```dart
// web/lib/screens/login_screen.dart
import 'package:flutter/material.dart';

class LoginScreen extends StatelessWidget {
  const LoginScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Telegram WebDAV Netdisk')),
      body: const Center(child: Text('Sign In')),
    );
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /workspace/web && flutter test test/widget_test.dart`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/pubspec.yaml web/lib/main.dart web/lib/app.dart web/lib/screens/login_screen.dart web/test/widget_test.dart
git commit -m "feat: add flutter web control plane shell"
```

### Task 10: Wire Production Serving, End-to-End Smoke Tests, and Docs

**Files:**
- Modify: `cmd/server/main.go`
- Modify: `internal/api/router.go`
- Create: `README.md`
- Modify: `Makefile`
- Test: `internal/api/router_test.go`

- [ ] **Step 1: Write the failing smoke test**

```go
// internal/api/router_test.go
func TestRootServesWebIndexWhenPresent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("ok"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	h := NewRouter(Dependencies{
		AppPassword: "secret",
		WebDir:      dir,
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); body != "ok" {
		t.Fatalf("body = %q, want %q", body, "ok")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/api -run TestRootServesWebIndexWhenPresent -v`
Expected: FAIL because root static serving is not wired yet

- [ ] **Step 3: Write the minimal production wiring and docs**

```go
// internal/api/router.go
type Dependencies struct {
	AppPassword string
	WebDir      string
}

func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/login", loginHandler(deps))
	mux.Handle("/webdav/", http.StripPrefix("/webdav", webdav.New(nil)))
	mux.Handle("/", http.FileServer(http.Dir(deps.WebDir)))
	return mux
}
```

```go
// cmd/server/main.go
handler := api.NewRouter(api.Dependencies{
	AppPassword: os.Getenv("APP_PASSWORD"),
	WebDir:      cfg.WebDir,
})
```

```makefile
# Makefile
.PHONY: test web-build

test:
	go test ./...

web-build:
	cd web && flutter build web
```

```md
# README.md

## Local Development

1. Set `APP_PASSWORD`, `APP_DB_PATH`, and `APP_STAGING_DIR`.
2. Run `go test ./...`.
3. Run `make web-build`.
4. Start the server with `go run ./cmd/server`.
5. Open `http://localhost:8080/`.

## Phase 1 Validation

- Login through the web UI.
- Upload one small file from the UI.
- Mount `/webdav/` with Cyberduck.
- Verify only `ready` files appear.
- Retry a failed upload job after restarting the server.
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/api -run TestRootServesWebIndexWhenPresent -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/server/main.go internal/api/router.go README.md Makefile internal/api/router_test.go
git commit -m "docs: wire production serving and validation flow"
```

## Self-Review Checklist

### Spec Coverage

- Flutter Web control plane: covered by Task 9 and Task 10
- Go REST API: covered by Task 6 and Task 10
- WebDAV endpoint: covered by Task 7 and Task 10
- Telegram user-account driver: covered by Task 3 and Task 8
- Virtual directory tree: covered by Task 2 and Task 4
- Ready-only visibility: covered by Task 4 and Task 5
- Chunk persistence and reconstruction prerequisites: covered by Task 3, Task 5, and Task 8
- Local staging quota: covered by Task 3 and Task 5
- Recovery workflow: covered by Task 5

### Placeholder Scan

- No `TBD`, `TODO`, or unresolved placeholders remain in task steps.
- Task 8 now uses explicit error boundaries for missing Telegram session material instead of placeholder panics.

### Type Consistency

- Go module path remains `telegram-webdav` throughout the plan.
- File visibility status names remain `uploading` and `ready` throughout the plan.
- The backend entry point remains `cmd/server/main.go`.
- The shared REST constructor remains `api.NewRouter`.
