package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"telegram-webdav/internal/jobs"
	"telegram-webdav/internal/store"
	"telegram-webdav/internal/vfs"
)

type FileSystem interface {
	EnsureRoot(ctx context.Context) (store.Directory, error)
	GetDirectory(ctx context.Context, id int64) (store.Directory, error)
	CreateDirectory(ctx context.Context, parent store.Directory, name string) (store.Directory, error)
	CreateFile(ctx context.Context, parentID int64, name string, size int64, status string) (store.FileEntry, error)
	ListDirectory(ctx context.Context, parentID int64) (vfs.DirectoryListing, error)
	DeleteFile(ctx context.Context, fileID int64) error
}

type UploadService interface {
	Run(ctx context.Context, in jobs.UploadInput) (jobs.UploadResult, error)
}

type QuotaManager interface {
	Reserve(jobID string, bytes int64) error
	Release(jobID string)
}

type DownloadService interface {
	ReadAll(ctx context.Context, fileID int64) ([]byte, error)
	StreamTo(ctx context.Context, fileID int64, w io.Writer) error
}

type mkdirRequest struct {
	ParentID int64  `json:"parent_id"`
	Name     string `json:"name"`
}

type deleteRequest struct {
	FileID int64 `json:"file_id"`
}

func fsTreeHandler(deps Dependencies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if deps.FS == nil {
			http.Error(w, "filesystem unavailable", http.StatusServiceUnavailable)
			return
		}
		root, err := deps.FS.EnsureRoot(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		current := root
		if raw := strings.TrimSpace(r.URL.Query().Get("parent_id")); raw != "" {
			parentID, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				http.Error(w, "invalid parent_id", http.StatusBadRequest)
				return
			}
			current, err = deps.FS.GetDirectory(r.Context(), parentID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		listing, err := deps.FS.ListDirectory(r.Context(), current.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"root":      root,
			"directory": current,
			"listing":   listing,
		})
	})
}

func mkdirHandler(deps Dependencies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if deps.FS == nil {
			http.Error(w, "filesystem unavailable", http.StatusServiceUnavailable)
			return
		}
		var req mkdirRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		parent, err := deps.FS.EnsureRoot(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if req.ParentID != 0 && req.ParentID != parent.ID {
			parent, err = deps.FS.GetDirectory(r.Context(), req.ParentID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		dir, err := deps.FS.CreateDirectory(r.Context(), parent, req.Name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, dir)
	})
}

func deleteHandler(deps Dependencies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if deps.FS == nil {
			http.Error(w, "filesystem unavailable", http.StatusServiceUnavailable)
			return
		}
		var req deleteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := deps.FS.DeleteFile(r.Context(), req.FileID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func uploadHandler(deps Dependencies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if deps.Uploader == nil {
			http.Error(w, "uploader unavailable", http.StatusServiceUnavailable)
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		fileName := filepath.Base(header.Filename)
		if fileName == "." || fileName == "" || fileName == string(filepath.Separator) {
			http.Error(w, "invalid file name", http.StatusBadRequest)
			return
		}
		if deps.Quota != nil {
			if err := deps.Quota.Reserve(fileName, header.Size); err != nil {
				http.Error(w, err.Error(), http.StatusInsufficientStorage)
				return
			}
			defer deps.Quota.Release(fileName)
		}
		if err := os.MkdirAll(deps.StagingDir, 0o755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		dst, err := os.CreateTemp(deps.StagingDir, "upload-*-"+fileName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		stagedPath := dst.Name()
		defer func() {
			_ = dst.Close()
			_ = os.Remove(stagedPath)
		}()
		if _, err := dst.ReadFrom(file); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		root, err := deps.FS.EnsureRoot(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		parentID := root.ID
		if raw := strings.TrimSpace(r.FormValue("parent_id")); raw != "" {
			parentID, err = strconv.ParseInt(raw, 10, 64)
			if err != nil {
				http.Error(w, "invalid parent_id", http.StatusBadRequest)
				return
			}
			if _, err := deps.FS.GetDirectory(r.Context(), parentID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		result, err := deps.Uploader.Run(r.Context(), jobs.UploadInput{
			ParentID:   parentID,
			Name:       fileName,
			Source:     "ui",
			StagedPath: stagedPath,
			ChunkSize:  deps.DefaultChunkSize,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, result)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func downloadHandler(deps Dependencies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if deps.Downloader == nil {
			http.Error(w, "downloader unavailable", http.StatusServiceUnavailable)
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/download") {
			http.NotFound(w, r)
			return
		}
		idPart := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/fs/file/"), "/download")
		fileID, err := strconv.ParseInt(strings.Trim(idPart, "/"), 10, 64)
		if err != nil {
			http.Error(w, "invalid file id", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		if err := deps.Downloader.StreamTo(r.Context(), fileID, w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}
