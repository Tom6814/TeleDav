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
