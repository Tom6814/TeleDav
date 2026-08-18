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
