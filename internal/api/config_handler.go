package api

import (
	"context"
	"encoding/json"
	"net/http"

	"telegram-webdav/internal/store"
)

type ConfigStore interface {
	GetSystemConfig(ctx context.Context) (store.SystemConfig, error)
	UpsertSystemConfig(ctx context.Context, cfg store.SystemConfig) error
}

type configResponse struct {
	TelegramTargetChatID int64 `json:"telegram_target_chat_id"`
	DefaultChunkSize     int64 `json:"default_chunk_size"`
	MaxStagingBytes      int64 `json:"max_staging_bytes"`
	DownloadCacheTTL     int64 `json:"download_cache_ttl_seconds"`
}

func configHandler(deps Dependencies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if deps.ConfigStore == nil {
			http.Error(w, "config store unavailable", http.StatusServiceUnavailable)
			return
		}
		switch r.Method {
		case http.MethodGet:
			cfg, err := deps.ConfigStore.GetSystemConfig(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, toConfigResponse(cfg))
		case http.MethodPatch:
			var cfg store.SystemConfig
			if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}
			if err := deps.ConfigStore.UpsertSystemConfig(r.Context(), cfg); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, toConfigResponse(cfg))
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func toConfigResponse(cfg store.SystemConfig) configResponse {
	return configResponse{
		TelegramTargetChatID: cfg.TelegramTargetChatID,
		DefaultChunkSize:     cfg.DefaultChunkSize,
		MaxStagingBytes:      cfg.MaxStagingBytes,
		DownloadCacheTTL:     cfg.DownloadCacheTTL,
	}
}
