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
