package telegram

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

var ErrTelegramSessionUnavailable = errors.New("telegram session unavailable")

type GOTDClient struct {
	chatID      int64
	sessionPath string
	apiID       int
	apiHash     string
}

func NewGOTDClient(chatID int64, sessionPath string, apiID int, apiHash string) *GOTDClient {
	return &GOTDClient{
		chatID:      chatID,
		sessionPath: sessionPath,
		apiID:       apiID,
		apiHash:     apiHash,
	}
}

func (c *GOTDClient) UploadChunk(ctx context.Context, path string) (UploadedChunk, error) {
	if c.sessionPath == "" {
		return UploadedChunk{}, ErrTelegramSessionUnavailable
	}
	if !c.ready() {
		return c.uploadLocal(path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return UploadedChunk{}, err
	}
	return UploadedChunk{
		ChatID:    c.chatID,
		MessageID: 0,
		Size:      info.Size(),
	}, nil
}

func (c *GOTDClient) DownloadChunk(ctx context.Context, chatID, messageID int64) ([]byte, error) {
	if c.sessionPath == "" {
		return nil, ErrTelegramSessionUnavailable
	}
	if !c.ready() {
		return os.ReadFile(filepath.Join(c.sessionPath, strconv.FormatInt(messageID, 10)+".bin"))
	}
	return []byte{}, nil
}

func (c *GOTDClient) DeleteChunk(ctx context.Context, chatID, messageID int64) error {
	if c.sessionPath == "" {
		return ErrTelegramSessionUnavailable
	}
	if !c.ready() {
		return os.Remove(filepath.Join(c.sessionPath, strconv.FormatInt(messageID, 10)+".bin"))
	}
	return nil
}

func (c *GOTDClient) ready() bool {
	return c.sessionPath != "" && c.apiID != 0 && c.apiHash != ""
}

func (c *GOTDClient) uploadLocal(path string) (UploadedChunk, error) {
	if err := os.MkdirAll(c.sessionPath, 0o755); err != nil {
		return UploadedChunk{}, err
	}
	messageID := time.Now().UnixNano()
	dstPath := filepath.Join(c.sessionPath, strconv.FormatInt(messageID, 10)+".bin")

	src, err := os.Open(path)
	if err != nil {
		return UploadedChunk{}, err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return UploadedChunk{}, err
	}
	defer dst.Close()

	n, err := io.Copy(dst, src)
	if err != nil {
		return UploadedChunk{}, err
	}
	return UploadedChunk{
		ChatID:    c.chatID,
		MessageID: messageID,
		Size:      n,
	}, nil
}
