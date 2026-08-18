package jobs

import (
	"context"

	"telegram-webdav/internal/store"
)

type jobRepository interface {
	ListPendingJobs(ctx context.Context) ([]store.UploadJob, error)
	GetUploadJob(ctx context.Context, jobID int64) (store.UploadJob, error)
	UpdateUploadJob(ctx context.Context, jobID int64, stage string, lastChunkIndex int, lastError string) error
}

type JobController struct {
	repo     jobRepository
	uploader interface {
		Resume(ctx context.Context, job store.UploadJob, chunkSize int64) error
	}
	chunkSize int64
}

func NewJobController(repo jobRepository, uploader interface {
	Resume(ctx context.Context, job store.UploadJob, chunkSize int64) error
}, chunkSize int64) *JobController {
	return &JobController{
		repo:      repo,
		uploader:  uploader,
		chunkSize: chunkSize,
	}
}

func (j *JobController) ListPendingJobs(ctx context.Context) ([]store.UploadJob, error) {
	return j.repo.ListPendingJobs(ctx)
}

func (j *JobController) GetUploadJob(ctx context.Context, jobID int64) (store.UploadJob, error) {
	return j.repo.GetUploadJob(ctx, jobID)
}

func (j *JobController) RetryJob(ctx context.Context, jobID int64) error {
	job, err := j.repo.GetUploadJob(ctx, jobID)
	if err != nil {
		return err
	}
	if err := j.repo.UpdateUploadJob(ctx, job.ID, "uploading", job.LastChunkIndex, ""); err != nil {
		return err
	}
	return j.uploader.Resume(ctx, job, j.chunkSize)
}
