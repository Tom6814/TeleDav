package api

import (
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"telegram-webdav/internal/jobs"
	"telegram-webdav/internal/store"
	"telegram-webdav/internal/vfs"
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

func TestFSUploadUnauthorizedWithoutSession(t *testing.T) {
	h := NewRouter(Dependencies{
		AppPassword:   "secret",
		SessionSecret: "session-secret",
		WebDir:        t.TempDir(),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/fs/upload", strings.NewReader("x"))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestFSUploadUnavailableWithoutUploader(t *testing.T) {
	h := NewRouter(Dependencies{
		AppPassword:      "secret",
		SessionSecret:    "",
		WebDir:           t.TempDir(),
		StagingDir:       t.TempDir(),
		DefaultChunkSize: 4 << 20,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/fs/upload", strings.NewReader("x"))
	req.AddCookie(&http.Cookie{
		Name:  "session",
		Value: "single-user",
	})
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestJobDetailAndRetryRoutes(t *testing.T) {
	jobs := &fakeJobController{
		job: store.UploadJob{ID: 7, Stage: "failed", LastError: "boom"},
	}
	h := NewRouter(Dependencies{
		AppPassword:   "secret",
		SessionSecret: "",
		WebDir:        t.TempDir(),
		Jobs:          jobs,
		Retryer:       jobs,
	})

	getReq := httptest.NewRequest(http.MethodGet, "/api/jobs/7", nil)
	getReq.AddCookie(&http.Cookie{Name: "session", Value: "single-user"})
	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET /api/jobs/7 status = %d, want %d", getRec.Code, http.StatusOK)
	}
	var job store.UploadJob
	if err := json.Unmarshal(getRec.Body.Bytes(), &job); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if job.ID != 7 {
		t.Fatalf("job.ID = %d, want 7", job.ID)
	}

	retryReq := httptest.NewRequest(http.MethodPost, "/api/jobs/7/retry", nil)
	retryReq.AddCookie(&http.Cookie{Name: "session", Value: "single-user"})
	retryRec := httptest.NewRecorder()
	h.ServeHTTP(retryRec, retryReq)
	if retryRec.Code != http.StatusNoContent {
		t.Fatalf("POST /api/jobs/7/retry status = %d, want %d", retryRec.Code, http.StatusNoContent)
	}
	if jobs.retriedID != 7 {
		t.Fatalf("retriedID = %d, want 7", jobs.retriedID)
	}
}

func TestStorageConfigPatchMergesExistingValues(t *testing.T) {
	ctx := context.Background()
	repo, err := store.Open("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer repo.Close()
	if err := repo.Migrate(ctx); err != nil {
		t.Fatalf("Migrate returned error: %v", err)
	}
	if err := repo.UpsertSystemConfig(ctx, store.SystemConfig{
		TelegramSessionBlob:  "session-data",
		TelegramTargetChatID: 42,
		DefaultChunkSize:     1024,
		MaxStagingBytes:      2048,
		DownloadCacheTTL:     3600,
		AppPassword:          "secret",
	}); err != nil {
		t.Fatalf("UpsertSystemConfig returned error: %v", err)
	}

	h := NewRouter(Dependencies{
		AppPassword:   "secret",
		SessionSecret: "",
		WebDir:        t.TempDir(),
		ConfigStore:   repo,
	})
	req := httptest.NewRequest(http.MethodPatch, "/api/config/storage", strings.NewReader(`{"default_chunk_size":4096}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session", Value: "single-user"})
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH /api/config/storage status = %d, want %d", rec.Code, http.StatusOK)
	}

	cfg, err := repo.GetSystemConfig(ctx)
	if err != nil {
		t.Fatalf("GetSystemConfig returned error: %v", err)
	}
	if cfg.DefaultChunkSize != 4096 {
		t.Fatalf("cfg.DefaultChunkSize = %d, want 4096", cfg.DefaultChunkSize)
	}
	if cfg.TelegramSessionBlob != "session-data" {
		t.Fatalf("cfg.TelegramSessionBlob = %q, want %q", cfg.TelegramSessionBlob, "session-data")
	}
	if cfg.TelegramTargetChatID != 42 {
		t.Fatalf("cfg.TelegramTargetChatID = %d, want 42", cfg.TelegramTargetChatID)
	}
	if cfg.MaxStagingBytes != 2048 {
		t.Fatalf("cfg.MaxStagingBytes = %d, want 2048", cfg.MaxStagingBytes)
	}
	if cfg.DownloadCacheTTL != 3600 {
		t.Fatalf("cfg.DownloadCacheTTL = %d, want 3600", cfg.DownloadCacheTTL)
	}
	if cfg.AppPassword != "secret" {
		t.Fatalf("cfg.AppPassword = %q, want %q", cfg.AppPassword, "secret")
	}
}

func TestFSTreeSupportsParentDirectoryQuery(t *testing.T) {
	ctx := context.Background()
	repo, err := store.Open("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer repo.Close()
	if err := repo.Migrate(ctx); err != nil {
		t.Fatalf("Migrate returned error: %v", err)
	}
	root, err := repo.EnsureRoot(ctx)
	if err != nil {
		t.Fatalf("EnsureRoot returned error: %v", err)
	}
	dir, err := repo.CreateDirectory(ctx, root.ID, "docs", "/docs")
	if err != nil {
		t.Fatalf("CreateDirectory returned error: %v", err)
	}
	file, err := repo.CreateFile(ctx, dir.ID, "note.txt", 4, "uploading")
	if err != nil {
		t.Fatalf("CreateFile returned error: %v", err)
	}
	if err := repo.MarkFileReady(ctx, file.ID); err != nil {
		t.Fatalf("MarkFileReady returned error: %v", err)
	}

	h := NewRouter(Dependencies{
		AppPassword:   "secret",
		SessionSecret: "",
		WebDir:        t.TempDir(),
		FS:            vfs.New(repo),
	})
	req := httptest.NewRequest(http.MethodGet, "/api/fs/tree?parent_id="+strconv.FormatInt(dir.ID, 10), nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "single-user"})
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/fs/tree status = %d, want %d", rec.Code, http.StatusOK)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	directory, ok := payload["directory"].(map[string]any)
	if !ok {
		t.Fatal("directory payload missing")
	}
	if got := directory["path"]; got != "/docs" {
		t.Fatalf("directory.path = %v, want /docs", got)
	}
}

func TestFSUploadUsesProvidedParentID(t *testing.T) {
	ctx := context.Background()
	repo, err := store.Open("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer repo.Close()
	if err := repo.Migrate(ctx); err != nil {
		t.Fatalf("Migrate returned error: %v", err)
	}
	root, err := repo.EnsureRoot(ctx)
	if err != nil {
		t.Fatalf("EnsureRoot returned error: %v", err)
	}
	dir, err := repo.CreateDirectory(ctx, root.ID, "docs", "/docs")
	if err != nil {
		t.Fatalf("CreateDirectory returned error: %v", err)
	}

	uploader := &captureUploader{}
	h := NewRouter(Dependencies{
		AppPassword:      "secret",
		SessionSecret:    "",
		WebDir:           t.TempDir(),
		StagingDir:       t.TempDir(),
		DefaultChunkSize: 4,
		FS:               vfs.New(repo),
		Uploader:         uploader,
	})

	var body strings.Builder
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("parent_id", strconv.FormatInt(dir.ID, 10)); err != nil {
		t.Fatalf("WriteField returned error: %v", err)
	}
	part, err := writer.CreateFormFile("file", "note.txt")
	if err != nil {
		t.Fatalf("CreateFormFile returned error: %v", err)
	}
	if _, err := part.Write([]byte("data")); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/fs/upload", strings.NewReader(body.String()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: "session", Value: "single-user"})
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/fs/upload status = %d, want %d", rec.Code, http.StatusOK)
	}
	if uploader.lastInput.ParentID != dir.ID {
		t.Fatalf("uploader.lastInput.ParentID = %d, want %d", uploader.lastInput.ParentID, dir.ID)
	}
}

type fakeJobController struct {
	job       store.UploadJob
	retriedID int64
}

func (f *fakeJobController) ListPendingJobs(ctx context.Context) ([]store.UploadJob, error) {
	return []store.UploadJob{f.job}, nil
}

func (f *fakeJobController) GetUploadJob(ctx context.Context, jobID int64) (store.UploadJob, error) {
	return f.job, nil
}

func (f *fakeJobController) RetryJob(ctx context.Context, jobID int64) error {
	f.retriedID = jobID
	return nil
}

type captureUploader struct {
	lastInput jobs.UploadInput
}

func (c *captureUploader) Run(ctx context.Context, in jobs.UploadInput) (jobs.UploadResult, error) {
	c.lastInput = in
	return jobs.UploadResult{FileID: 1, Status: "ready", Chunks: 1}, nil
}
