package vfs

import (
	"context"
	"fmt"
	"path"

	"telegram-webdav/internal/store"
)

type Repository interface {
	EnsureRoot(ctx context.Context) (store.Directory, error)
	GetDirectory(ctx context.Context, id int64) (store.Directory, error)
	CreateDirectory(ctx context.Context, parentID int64, name, path string) (store.Directory, error)
	CreateFile(ctx context.Context, parentID int64, name string, size int64, status string) (store.FileEntry, error)
	DeleteFileLogical(ctx context.Context, fileID int64) error
	GetFile(ctx context.Context, fileID int64) (store.FileEntry, error)
	ListDirectories(ctx context.Context, parentID int64) ([]store.Directory, error)
	ListReadyFiles(ctx context.Context, parentID int64) ([]store.FileEntry, error)
}

type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) EnsureRoot(ctx context.Context) (store.Directory, error) {
	return s.repo.EnsureRoot(ctx)
}

func (s *Service) CreateDirectory(ctx context.Context, parent store.Directory, name string) (store.Directory, error) {
	fullPath := path.Join(parent.Path, name)
	if parent.Path == "/" {
		fullPath = "/" + name
	}
	return s.repo.CreateDirectory(ctx, parent.ID, name, fullPath)
}

func (s *Service) GetDirectory(ctx context.Context, id int64) (store.Directory, error) {
	return s.repo.GetDirectory(ctx, id)
}

func (s *Service) CreateFile(ctx context.Context, parentID int64, name string, size int64, status string) (store.FileEntry, error) {
	return s.repo.CreateFile(ctx, parentID, name, size, status)
}

func (s *Service) ListDirectory(ctx context.Context, parentID int64) (DirectoryListing, error) {
	directories, err := s.repo.ListDirectories(ctx, parentID)
	if err != nil {
		return DirectoryListing{}, err
	}
	files, err := s.repo.ListReadyFiles(ctx, parentID)
	if err != nil {
		return DirectoryListing{}, err
	}
	return DirectoryListing{
		Directories: directories,
		Files:       files,
	}, nil
}

func (s *Service) DeleteFile(ctx context.Context, fileID int64) error {
	if _, err := s.repo.GetFile(ctx, fileID); err != nil {
		return fmt.Errorf("load file: %w", err)
	}
	return s.repo.DeleteFileLogical(ctx, fileID)
}
