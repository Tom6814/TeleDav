package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"telegram-webdav/internal/store"
)

type JobReader interface {
	ListPendingJobs(ctx context.Context) ([]store.UploadJob, error)
	GetUploadJob(ctx context.Context, jobID int64) (store.UploadJob, error)
}

type JobRetryer interface {
	RetryJob(ctx context.Context, jobID int64) error
}

func jobsHandler(deps Dependencies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if deps.Jobs == nil {
			http.Error(w, "jobs unavailable", http.StatusServiceUnavailable)
			return
		}
		switch {
		case r.URL.Path == "/api/jobs":
			jobs, err := deps.Jobs.ListPendingJobs(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, jobs)
		case r.Method == http.MethodGet:
			jobID, err := parseJobID(r.URL.Path, "/api/jobs/")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			job, err := deps.Jobs.GetUploadJob(r.Context(), jobID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, job)
		case r.Method == http.MethodPost && hasSuffix(r.URL.Path, "/retry"):
			if deps.Retryer == nil {
				http.Error(w, "retry unavailable", http.StatusServiceUnavailable)
				return
			}
			jobID, err := parseJobID(strings.TrimSuffix(r.URL.Path, "/retry"), "/api/jobs/")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := deps.Retryer.RetryJob(r.Context(), jobID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	})
}

func parseJobID(path, prefix string) (int64, error) {
	raw := strings.TrimPrefix(path, prefix)
	raw = strings.Trim(raw, "/")
	return strconv.ParseInt(raw, 10, 64)
}

func hasSuffix(path, suffix string) bool {
	return strings.HasSuffix(path, suffix)
}
