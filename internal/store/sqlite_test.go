package store

import (
	"context"
	"testing"
)

func TestEnsureRootPath(t *testing.T) {
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
