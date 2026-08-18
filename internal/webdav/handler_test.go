package webdav

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"telegram-webdav/internal/jobs"
	"telegram-webdav/internal/store"
	"telegram-webdav/internal/telegram"
	"telegram-webdav/internal/vfs"
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

func TestWebDAVSupportsNestedMkcolPutGetAndMove(t *testing.T) {
	ctx := context.Background()
	repo, err := store.Open("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer repo.Close()
	if err := repo.Migrate(ctx); err != nil {
		t.Fatalf("Migrate returned error: %v", err)
	}
	if _, err := repo.EnsureRoot(ctx); err != nil {
		t.Fatalf("EnsureRoot returned error: %v", err)
	}

	spoolDir := filepath.Join(t.TempDir(), "spool")
	client := telegram.NewGOTDClient(1, spoolDir, 0, "")
	uploader := jobs.NewUploader(repo, client)
	downloader := jobs.NewDownloader(repo, client)
	handler := New(&Service{
		FS:               vfs.New(repo),
		Uploader:         uploader,
		Downloader:       downloader,
		StagingDir:       filepath.Join(t.TempDir(), "staging"),
		DefaultChunkSize: 4,
	})

	mkcol := httptest.NewRequest("MKCOL", "/docs", nil)
	mkcolRec := httptest.NewRecorder()
	handler.ServeHTTP(mkcolRec, mkcol)
	if mkcolRec.Code != http.StatusCreated {
		t.Fatalf("MKCOL /docs status = %d, want %d", mkcolRec.Code, http.StatusCreated)
	}

	nestedMkcol := httptest.NewRequest("MKCOL", "/docs/archive", nil)
	nestedRec := httptest.NewRecorder()
	handler.ServeHTTP(nestedRec, nestedMkcol)
	if nestedRec.Code != http.StatusCreated {
		t.Fatalf("MKCOL /docs/archive status = %d, want %d", nestedRec.Code, http.StatusCreated)
	}

	putReq := httptest.NewRequest(http.MethodPut, "/docs/archive/report.txt", strings.NewReader("hello world"))
	putRec := httptest.NewRecorder()
	handler.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusCreated {
		t.Fatalf("PUT status = %d, want %d", putRec.Code, http.StatusCreated)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/docs/archive/report.txt", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", getRec.Code, http.StatusOK)
	}
	if body := getRec.Body.String(); body != "hello world" {
		t.Fatalf("GET body = %q, want %q", body, "hello world")
	}

	moveReq := httptest.NewRequest("MOVE", "/docs/archive/report.txt", nil)
	moveReq.Header.Set("Destination", "/docs/archive/report-final.txt")
	moveRec := httptest.NewRecorder()
	handler.ServeHTTP(moveRec, moveReq)
	if moveRec.Code != http.StatusCreated {
		t.Fatalf("MOVE status = %d, want %d", moveRec.Code, http.StatusCreated)
	}

	oldGetReq := httptest.NewRequest(http.MethodGet, "/docs/archive/report.txt", nil)
	oldGetRec := httptest.NewRecorder()
	handler.ServeHTTP(oldGetRec, oldGetReq)
	if oldGetRec.Code != http.StatusNotFound {
		t.Fatalf("old GET status = %d, want %d", oldGetRec.Code, http.StatusNotFound)
	}

	newGetReq := httptest.NewRequest(http.MethodGet, "/docs/archive/report-final.txt", nil)
	newGetRec := httptest.NewRecorder()
	handler.ServeHTTP(newGetRec, newGetReq)
	if newGetRec.Code != http.StatusOK {
		t.Fatalf("new GET status = %d, want %d", newGetRec.Code, http.StatusOK)
	}
	if body := newGetRec.Body.String(); body != "hello world" {
		t.Fatalf("new GET body = %q, want %q", body, "hello world")
	}

	propfindReq := httptest.NewRequest("PROPFIND", "/docs/archive", nil)
	propfindRec := httptest.NewRecorder()
	handler.ServeHTTP(propfindRec, propfindReq)
	if propfindRec.Code != http.StatusMultiStatus {
		t.Fatalf("PROPFIND status = %d, want %d", propfindRec.Code, http.StatusMultiStatus)
	}
	if !strings.Contains(propfindRec.Body.String(), "report-final.txt") {
		t.Fatalf("PROPFIND body = %q, want nested file", propfindRec.Body.String())
	}
}

func TestWebDAVCopyCreatesSecondReadableFile(t *testing.T) {
	ctx := context.Background()
	repo, err := store.Open("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer repo.Close()
	if err := repo.Migrate(ctx); err != nil {
		t.Fatalf("Migrate returned error: %v", err)
	}
	if _, err := repo.EnsureRoot(ctx); err != nil {
		t.Fatalf("EnsureRoot returned error: %v", err)
	}

	spoolDir := filepath.Join(t.TempDir(), "spool")
	client := telegram.NewGOTDClient(1, spoolDir, 0, "")
	handler := New(&Service{
		FS:               vfs.New(repo),
		Uploader:         jobs.NewUploader(repo, client),
		Downloader:       jobs.NewDownloader(repo, client),
		StagingDir:       filepath.Join(t.TempDir(), "staging"),
		DefaultChunkSize: 4,
	})

	putReq := httptest.NewRequest(http.MethodPut, "/source.txt", strings.NewReader("copy me"))
	putRec := httptest.NewRecorder()
	handler.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusCreated {
		t.Fatalf("PUT status = %d, want %d", putRec.Code, http.StatusCreated)
	}

	copyReq := httptest.NewRequest("COPY", "/source.txt", nil)
	copyReq.Header.Set("Destination", "/copy.txt")
	copyRec := httptest.NewRecorder()
	handler.ServeHTTP(copyRec, copyReq)
	if copyRec.Code != http.StatusCreated {
		t.Fatalf("COPY status = %d, want %d", copyRec.Code, http.StatusCreated)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/copy.txt", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET copy status = %d, want %d", getRec.Code, http.StatusOK)
	}
	if body := getRec.Body.String(); body != "copy me" {
		t.Fatalf("GET copy body = %q, want %q", body, "copy me")
	}

	entries, err := os.ReadDir(spoolDir)
	if err != nil {
		t.Fatalf("ReadDir returned error: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("spool dir empty, want uploaded chunks")
	}
}
