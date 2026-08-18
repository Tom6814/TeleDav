package jobs

import (
	"bytes"
	"context"
	"io"

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

func (d *Downloader) StreamTo(ctx context.Context, fileID int64, w io.Writer) error {
	chunks, err := d.repo.ListChunks(ctx, fileID)
	if err != nil {
		return err
	}
	for _, chunk := range chunks {
		data, err := d.client.DownloadChunk(ctx, chunk.TelegramChatID, chunk.TelegramMessageID)
		if err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
	}
	return nil
}

func (d *Downloader) ReadAll(ctx context.Context, fileID int64) ([]byte, error) {
	var buffer bytes.Buffer
	if err := d.StreamTo(ctx, fileID, &buffer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
