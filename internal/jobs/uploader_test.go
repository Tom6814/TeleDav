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
