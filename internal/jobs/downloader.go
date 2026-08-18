package jobs

import (
	"bytes"
	"context"

	"telegram-webdav/internal/store"
	"telegram-webdav/internal/telegram"
)

type chunkReader interface {
	ListChunks(ctx context.Context, fileID int64) ([]store.FileChunk, error)
}

type Downloader struct {
	repo   chunkReader
	client telegram.Client
}

func NewDownloader(repo chunkReader, client telegram.Client) *Downloader {
	return &Downloader{repo: repo, client: client}
}

func (d *Downloader) ReadAll(ctx context.Context, fileID int64) ([]byte, error) {
	chunks, err := d.repo.ListChunks(ctx, fileID)
	if err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	for _, chunk := range chunks {
		data, err := d.client.DownloadChunk(ctx, chunk.TelegramChatID, chunk.TelegramMessageID)
		if err != nil {
			return nil, err
		}
		if _, err := buffer.Write(data); err != nil {
			return nil, err
		}
	}
	return buffer.Bytes(), nil
}
