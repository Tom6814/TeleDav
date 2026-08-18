package jobs

import (
	"bytes"
	"context"
	"testing"

	"telegram-webdav/internal/store"
	"telegram-webdav/internal/telegram"
)

func TestDownloaderStreamsChunksInOrder(t *testing.T) {
	ctx := context.Background()
	repo, err := store.Open("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer repo.Close()
	if err := repo.Migrate(ctx); err != nil {
		t.Fatalf("Migrate returned error: %v", err)
	}
	if err := repo.AppendChunk(ctx, 1, store.FileChunk{FileID: 1, ChunkIndex: 0, ChunkSize: 5, TelegramChatID: 1, TelegramMessageID: 101}); err != nil {
		t.Fatalf("AppendChunk returned error: %v", err)
	}
	if err := repo.AppendChunk(ctx, 1, store.FileChunk{FileID: 1, ChunkIndex: 1, ChunkSize: 5, TelegramChatID: 1, TelegramMessageID: 102}); err != nil {
		t.Fatalf("AppendChunk returned error: %v", err)
	}

	client := &streamFakeTelegramClient{
		dataByMessageID: map[int64][]byte{
			101: []byte("hello"),
			102: []byte("world"),
		},
	}
	downloader := NewDownloader(repo, client)
	var buffer bytes.Buffer
	if err := downloader.StreamTo(ctx, 1, &buffer); err != nil {
		t.Fatalf("StreamTo returned error: %v", err)
	}
	if got := buffer.String(); got != "helloworld" {
		t.Fatalf("buffer = %q, want %q", got, "helloworld")
	}
}

type streamFakeTelegramClient struct {
	dataByMessageID map[int64][]byte
}

func (f *streamFakeTelegramClient) UploadChunk(ctx context.Context, path string) (telegram.UploadedChunk, error) {
	return telegram.UploadedChunk{}, nil
}

func (f *streamFakeTelegramClient) DownloadChunk(ctx context.Context, chatID, messageID int64) ([]byte, error) {
	return f.dataByMessageID[messageID], nil
}

func (f *streamFakeTelegramClient) DeleteChunk(ctx context.Context, chatID, messageID int64) error {
	return nil
}
