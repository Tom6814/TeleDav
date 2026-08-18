package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
