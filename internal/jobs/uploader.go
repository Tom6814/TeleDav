package jobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"telegram-webdav/internal/store"
	"telegram-webdav/internal/telegram"
)

type uploaderRepository interface {
	CreateFile(ctx context.Context, parentID int64, name string, size int64, status string) (store.FileEntry, error)
	CreateUploadJob(ctx context.Context, source, stage, stagedPath string, fileID int64) (store.UploadJob, error)
	UpdateUploadJob(ctx context.Context, jobID int64, stage string, lastChunkIndex int, lastError string) error
	AppendChunk(ctx context.Context, fileID int64, chunk store.FileChunk) error
	ListChunks(ctx context.Context, fileID int64) ([]store.FileChunk, error)
	GetFile(ctx context.Context, fileID int64) (store.FileEntry, error)
	MarkFileReady(ctx context.Context, fileID int64) error
	ResetFileChunks(ctx context.Context, fileID int64) error
}

type UploadInput struct {
	ParentID   int64
	Name       string
	Source     string
	StagedPath string
	ChunkSize  int64
}

type UploadResult struct {
	FileID int64
	Status string
	Chunks int
}

type Uploader struct {
	repo   uploaderRepository
	client telegram.Client
}

func NewUploader(repo uploaderRepository, client telegram.Client) *Uploader {
	return &Uploader{repo: repo, client: client}
}

func (u *Uploader) Run(ctx context.Context, in UploadInput) (UploadResult, error) {
	info, err := os.Stat(in.StagedPath)
	if err != nil {
		return UploadResult{}, err
	}
	source := in.Source
	if source == "" {
		source = "ui"
	}
	file, err := u.repo.CreateFile(ctx, in.ParentID, in.Name, info.Size(), "uploading")
	if err != nil {
		return UploadResult{}, err
	}
	job, err := u.repo.CreateUploadJob(ctx, source, "uploading", in.StagedPath, file.ID)
	if err != nil {
		return UploadResult{}, err
	}
	return u.uploadExisting(ctx, file, job, in.StagedPath, in.ChunkSize)
}

func (u *Uploader) Resume(ctx context.Context, job store.UploadJob, chunkSize int64) error {
	file, err := u.repo.GetFile(ctx, job.FileID)
	if err != nil {
		return err
	}
	_, err = u.uploadExisting(ctx, file, job, job.StagedPath, chunkSize)
	return err
}

func (u *Uploader) uploadExisting(ctx context.Context, file store.FileEntry, job store.UploadJob, stagedPath string, chunkSize int64) (UploadResult, error) {
	parts, err := BuildChunkPlan(stagedPath, chunkSize)
	if err != nil {
		return UploadResult{}, err
	}
	existing, err := u.repo.ListChunks(ctx, file.ID)
	if err != nil {
		return UploadResult{}, err
	}
	existingByIndex := make(map[int]store.FileChunk, len(existing))
	lastSuccessfulIndex := -1
	for _, chunk := range existing {
		existingByIndex[chunk.ChunkIndex] = chunk
		if chunk.ChunkIndex > lastSuccessfulIndex {
			lastSuccessfulIndex = chunk.ChunkIndex
		}
	}
	for _, part := range parts {
		if _, ok := existingByIndex[part.Index]; ok {
			continue
		}
		chunkPath, checksum, err := materializeChunk(stagedPath, part)
		if err != nil {
			_ = u.repo.UpdateUploadJob(ctx, job.ID, "failed", lastSuccessfulIndex, err.Error())
			return UploadResult{}, err
		}
		ref, err := u.client.UploadChunk(ctx, chunkPath)
		_ = os.Remove(chunkPath)
		if err != nil {
			_ = u.repo.UpdateUploadJob(ctx, job.ID, "failed", lastSuccessfulIndex, err.Error())
			return UploadResult{}, err
		}
		if err := u.repo.AppendChunk(ctx, file.ID, store.FileChunk{
			FileID:            file.ID,
			ChunkIndex:        part.Index,
			ChunkSize:         part.Size,
			ChunkSHA256:       checksum,
			TelegramChatID:    ref.ChatID,
			TelegramMessageID: ref.MessageID,
		}); err != nil {
			_ = u.repo.UpdateUploadJob(ctx, job.ID, "failed", lastSuccessfulIndex, err.Error())
			return UploadResult{}, err
		}
		lastSuccessfulIndex = part.Index
		if err := u.repo.UpdateUploadJob(ctx, job.ID, "uploading", part.Index, ""); err != nil {
			return UploadResult{}, err
		}
	}
	if err := u.repo.MarkFileReady(ctx, file.ID); err != nil {
		return UploadResult{}, err
	}
	if err := u.repo.UpdateUploadJob(ctx, job.ID, "done", lastSuccessfulIndex, ""); err != nil {
		return UploadResult{}, err
	}
	return UploadResult{FileID: file.ID, Status: "ready", Chunks: len(parts)}, nil
}

func materializeChunk(path string, part ChunkPart) (string, string, error) {
	src, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer src.Close()

	if _, err := src.Seek(part.Offset, io.SeekStart); err != nil {
		return "", "", err
	}

	chunkPath := fmt.Sprintf("%s.part.%d", path, part.Index)
	dst, err := os.Create(chunkPath)
	if err != nil {
		return "", "", err
	}
	defer dst.Close()

	hasher := sha256.New()
	writer := io.MultiWriter(dst, hasher)
	if _, err := io.CopyN(writer, src, part.Size); err != nil {
		return "", "", err
	}
	return chunkPath, hex.EncodeToString(hasher.Sum(nil)), nil
}
