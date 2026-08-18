package jobs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"telegram-webdav/internal/store"
	"telegram-webdav/internal/telegram"
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

func TestUploaderMarksFileReadyAfterAllChunksUpload(t *testing.T) {
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

	stagedPath := filepath.Join(t.TempDir(), "movie.bin")
	if err := os.WriteFile(stagedPath, []byte("12345678"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	client := &fakeTelegramClient{
		uploads: []telegram.UploadedChunk{
			{ChatID: 1, MessageID: 101, Size: 4},
			{ChatID: 1, MessageID: 102, Size: 4},
		},
	}
	uploader := NewUploader(repo, client)
	result, err := uploader.Run(ctx, UploadInput{
		ParentID:   root.ID,
		Name:       "movie.bin",
		StagedPath: stagedPath,
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

func TestUploaderPersistsChunkReferences(t *testing.T) {
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

	stagedPath := filepath.Join(t.TempDir(), "movie.bin")
	if err := os.WriteFile(stagedPath, []byte("12345678"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	client := &fakeTelegramClient{
		uploads: []telegram.UploadedChunk{
			{ChatID: 1, MessageID: 101, Size: 4},
			{ChatID: 1, MessageID: 102, Size: 4},
		},
	}
	uploader := NewUploader(repo, client)
	result, err := uploader.Run(ctx, UploadInput{
		ParentID:   root.ID,
		Name:       "movie.bin",
		StagedPath: stagedPath,
		ChunkSize:  4,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	chunks, err := repo.ListChunks(ctx, result.FileID)
	if err != nil {
		t.Fatalf("ListChunks returned error: %v", err)
	}
	if len(chunks) != 2 {
		t.Fatalf("len(chunks) = %d, want 2", len(chunks))
	}
	if chunks[0].TelegramMessageID != 101 {
		t.Fatalf("chunks[0].TelegramMessageID = %d, want 101", chunks[0].TelegramMessageID)
	}
	if chunks[1].TelegramMessageID != 102 {
		t.Fatalf("chunks[1].TelegramMessageID = %d, want 102", chunks[1].TelegramMessageID)
	}
}

type fakeTelegramClient struct {
	uploads     []telegram.UploadedChunk
	calledPaths []string
}

func (f *fakeTelegramClient) UploadChunk(ctx context.Context, path string) (telegram.UploadedChunk, error) {
	f.calledPaths = append(f.calledPaths, path)
	return f.uploads[len(f.calledPaths)-1], nil
}

func (f *fakeTelegramClient) DownloadChunk(ctx context.Context, chatID, messageID int64) ([]byte, error) {
	return []byte("ok"), nil
}

func (f *fakeTelegramClient) DeleteChunk(ctx context.Context, chatID, messageID int64) error {
	return nil
}
