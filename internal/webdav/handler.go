package webdav

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"os"
	gopath "path"
	"strings"

	"telegram-webdav/internal/jobs"
	"telegram-webdav/internal/store"
	"telegram-webdav/internal/vfs"
)

type FileSystem interface {
	EnsureRoot(ctx context.Context) (store.Directory, error)
	CreateDirectory(ctx context.Context, parent store.Directory, name string) (store.Directory, error)
	ListDirectory(ctx context.Context, parentID int64) (vfs.DirectoryListing, error)
	DeleteFile(ctx context.Context, fileID int64) error
}

type UploadService interface {
	Run(ctx context.Context, in jobs.UploadInput) (jobs.UploadResult, error)
}

type DownloadService interface {
	ReadAll(ctx context.Context, fileID int64) ([]byte, error)
	StreamTo(ctx context.Context, fileID int64, w io.Writer) error
}

type Service struct {
	FS               FileSystem
	Uploader         UploadService
	Downloader       DownloadService
	StagingDir       string
	DefaultChunkSize int64
}

type multiStatus struct {
	XMLName   xml.Name        `xml:"DAV: multistatus"`
	Responses []propfindEntry `xml:"response"`
}

type propfindEntry struct {
	Href string `xml:"href"`
}

func New(service *Service) http.Handler {
	if service == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "PROPFIND":
				w.Header().Set("Content-Type", "application/xml; charset=utf-8")
				w.WriteHeader(http.StatusMultiStatus)
				_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?><multistatus xmlns="DAV:"/>`))
			case http.MethodGet, http.MethodPut, http.MethodDelete:
				w.WriteHeader(http.StatusNoContent)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		root, err := service.FS.EnsureRoot(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cleanPath := cleanWebDAVPath(r.URL.Path)
		parentDir, name, err := resolveParent(r.Context(), service.FS, root, cleanPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		switch r.Method {
		case "PROPFIND":
			targetDir := root
			if cleanPath != "/" {
				targetDir, err = resolveDirectory(r.Context(), service.FS, root, cleanPath)
				if err != nil {
					http.Error(w, err.Error(), http.StatusNotFound)
					return
				}
			}
			listing, err := service.FS.ListDirectory(r.Context(), targetDir.ID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			response := multiStatus{}
			for _, directory := range listing.Directories {
				response.Responses = append(response.Responses, propfindEntry{Href: "/" + directory.Name})
			}
			for _, file := range listing.Files {
				response.Responses = append(response.Responses, propfindEntry{Href: "/" + file.Name})
			}
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = w.Write([]byte(xml.Header))
			_ = xml.NewEncoder(w).Encode(response)
		case "MKCOL":
			_, err := service.FS.CreateDirectory(r.Context(), parentDir, name)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
		case http.MethodPut:
			if service.Uploader == nil {
				http.Error(w, "uploader unavailable", http.StatusServiceUnavailable)
				return
			}
			if err := os.MkdirAll(service.StagingDir, 0o755); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			dst, err := os.CreateTemp(service.StagingDir, "webdav-*-"+name)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			stagedPath := dst.Name()
			if _, err := dst.ReadFrom(r.Body); err != nil {
				_ = dst.Close()
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			_ = dst.Close()
			defer os.Remove(stagedPath)

			if _, err := service.Uploader.Run(r.Context(), jobs.UploadInput{
				ParentID:   parentDir.ID,
				Name:       name,
				Source:     "webdav",
				StagedPath: stagedPath,
				ChunkSize:  service.DefaultChunkSize,
			}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
		case http.MethodGet:
			file, ok, err := lookupFile(r.Context(), service.FS, parentDir.ID, name)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if !ok {
				http.NotFound(w, r)
				return
			}
			if service.Downloader == nil {
				http.Error(w, "downloader unavailable", http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
			if err := service.Downloader.StreamTo(r.Context(), file.ID, w); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		case http.MethodDelete:
			file, ok, err := lookupFile(r.Context(), service.FS, parentDir.ID, name)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if !ok {
				http.NotFound(w, r)
				return
			}
			if err := service.FS.DeleteFile(r.Context(), file.ID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case "MOVE", "COPY":
			if service.Uploader == nil || service.Downloader == nil {
				http.Error(w, "transfer unavailable", http.StatusServiceUnavailable)
				return
			}
			file, ok, err := lookupFile(r.Context(), service.FS, parentDir.ID, name)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if !ok {
				http.NotFound(w, r)
				return
			}
			destination := cleanWebDAVPath(r.Header.Get("Destination"))
			destParent, destName, err := resolveParent(r.Context(), service.FS, root, destination)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			data, err := service.Downloader.ReadAll(r.Context(), file.ID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := os.MkdirAll(service.StagingDir, 0o755); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			dst, err := os.CreateTemp(service.StagingDir, "move-copy-*-"+destName)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			stagedPath := dst.Name()
			if _, err := dst.Write(data); err != nil {
				_ = dst.Close()
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			_ = dst.Close()
			defer os.Remove(stagedPath)

			if _, err := service.Uploader.Run(r.Context(), jobs.UploadInput{
				ParentID:   destParent.ID,
				Name:       destName,
				Source:     "webdav",
				StagedPath: stagedPath,
				ChunkSize:  service.DefaultChunkSize,
			}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if r.Method == "MOVE" {
				if err := service.FS.DeleteFile(r.Context(), file.ID); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			w.WriteHeader(http.StatusCreated)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func lookupFile(ctx context.Context, fs FileSystem, parentID int64, name string) (store.FileEntry, bool, error) {
	listing, err := fs.ListDirectory(ctx, parentID)
	if err != nil {
		return store.FileEntry{}, false, err
	}
	for _, file := range listing.Files {
		if file.Name == name {
			return file, true, nil
		}
	}
	return store.FileEntry{}, false, nil
}

func resolveDirectory(ctx context.Context, fs FileSystem, root store.Directory, rawPath string) (store.Directory, error) {
	if rawPath == "/" {
		return root, nil
	}
	current := root
	parts := strings.Split(strings.Trim(rawPath, "/"), "/")
	for _, part := range parts {
		listing, err := fs.ListDirectory(ctx, current.ID)
		if err != nil {
			return store.Directory{}, err
		}
		found := false
		for _, dir := range listing.Directories {
			if dir.Name == part {
				current = dir
				found = true
				break
			}
		}
		if !found {
			return store.Directory{}, os.ErrNotExist
		}
	}
	return current, nil
}

func resolveParent(ctx context.Context, fs FileSystem, root store.Directory, rawPath string) (store.Directory, string, error) {
	clean := cleanWebDAVPath(rawPath)
	if clean == "/" {
		return root, "", nil
	}
	parentPath := gopath.Dir(clean)
	name := gopath.Base(clean)
	if parentPath == "." {
		parentPath = "/"
	}
	parent, err := resolveDirectory(ctx, fs, root, parentPath)
	return parent, name, err
}

func cleanWebDAVPath(raw string) string {
	if raw == "" {
		return "/"
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		if req, err := http.NewRequest(http.MethodGet, raw, nil); err == nil {
			raw = req.URL.Path
		}
	}
	clean := gopath.Clean("/" + strings.TrimSpace(raw))
	if clean == "." {
		return "/"
	}
	return clean
}
