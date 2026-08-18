package api

import (
	"context"
	"net/http"

	"telegram-webdav/internal/store"
)

type JobReader interface {
	ListPendingJobs(ctx context.Context) ([]store.UploadJob, error)
}

func jobsHandler(deps Dependencies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if deps.Jobs == nil {
			http.Error(w, "jobs unavailable", http.StatusServiceUnavailable)
			return
		}
		jobs, err := deps.Jobs.ListPendingJobs(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, jobs)
	})
}
