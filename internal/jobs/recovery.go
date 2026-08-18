package jobs

import (
	"context"

	"telegram-webdav/internal/store"
)

type pendingJobRepository interface {
	ListPendingJobs(ctx context.Context) ([]store.UploadJob, error)
}

type Recoverer interface {
	ResumePending(ctx context.Context) error
}

func RunRecovery(ctx context.Context, r Recoverer) error {
	return r.ResumePending(ctx)
}

type RecoveryService struct {
	repo     pendingJobRepository
	uploader interface {
		Resume(ctx context.Context, job store.UploadJob, chunkSize int64) error
	}
	chunkSize int64
}

func NewRecoveryService(repo pendingJobRepository, uploader interface {
	Resume(ctx context.Context, job store.UploadJob, chunkSize int64) error
}, chunkSize int64) *RecoveryService {
	return &RecoveryService{
		repo:      repo,
		uploader:  uploader,
		chunkSize: chunkSize,
	}
}

func (r *RecoveryService) ResumePending(ctx context.Context) error {
	if r.repo == nil || r.uploader == nil {
		return nil
	}
	jobs, err := r.repo.ListPendingJobs(ctx)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if err := r.uploader.Resume(ctx, job, r.chunkSize); err != nil {
			return err
		}
	}
	return nil
}
