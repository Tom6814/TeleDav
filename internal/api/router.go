package api

import (
	"net/http"

	"telegram-webdav/internal/httpx"
	appwebdav "telegram-webdav/internal/webdav"
)

type Dependencies struct {
	AppPassword      string
	SessionSecret    string
	WebDir           string
	StagingDir       string
	DefaultChunkSize int64
	Quota            QuotaManager
	ConfigStore      ConfigStore
	FS               FileSystem
	Jobs             JobReader
	Uploader         UploadService
	Downloader       DownloadService
	WebDAV           http.Handler
}

func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/login", loginHandler(deps))
	mux.Handle("/api/config/storage", requireSession(deps, configHandler(deps)))
	mux.Handle("/api/fs/tree", requireSession(deps, fsTreeHandler(deps)))
	mux.Handle("/api/fs/mkdir", requireSession(deps, mkdirHandler(deps)))
	mux.Handle("/api/fs/delete", requireSession(deps, deleteHandler(deps)))
	mux.Handle("/api/fs/upload", requireSession(deps, uploadHandler(deps)))
	mux.Handle("/api/fs/file/", requireSession(deps, downloadHandler(deps)))
	mux.Handle("/api/jobs", requireSession(deps, jobsHandler(deps)))

	webdavHandler := deps.WebDAV
	if webdavHandler == nil {
		webdavHandler = appwebdav.New(nil)
	}
	mux.Handle("/webdav/", http.StripPrefix("/webdav", webdavHandler))
	mux.Handle("/", http.FileServer(http.Dir(deps.WebDir)))
	return mux
}

func requireSession(deps Dependencies, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !httpx.IsAuthorized(r, deps.SessionSecret) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
