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
	TelegramTargetChatID    int64 `json:"telegram_target_chat_id"`
	DefaultChunkSize        int64 `json:"default_chunk_size"`
	MaxStagingBytes         int64 `json:"max_staging_bytes"`
	DownloadCacheTTL        int64 `json:"download_cache_ttl_seconds"`
	TelegramSessionReady    bool  `json:"telegram_session_ready"`
	ApplicationPasswordSet  bool  `json:"application_password_set"`
}

type configPatchRequest struct {
	TelegramSessionBlob  *string `json:"telegram_session_blob"`
	TelegramTargetChatID *int64  `json:"telegram_target_chat_id"`
	DefaultChunkSize     *int64  `json:"default_chunk_size"`
	MaxStagingBytes      *int64  `json:"max_staging_bytes"`
	DownloadCacheTTL     *int64  `json:"download_cache_ttl_seconds"`
	AppPassword          *string `json:"app_password"`
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
			current, err := deps.ConfigStore.GetSystemConfig(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			var patch configPatchRequest
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}
			cfg := applyConfigPatch(current, patch)
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
		TelegramTargetChatID:   cfg.TelegramTargetChatID,
		DefaultChunkSize:       cfg.DefaultChunkSize,
		MaxStagingBytes:        cfg.MaxStagingBytes,
		DownloadCacheTTL:       cfg.DownloadCacheTTL,
		TelegramSessionReady:   cfg.TelegramSessionBlob != "",
		ApplicationPasswordSet: cfg.AppPassword != "",
	}
}

func applyConfigPatch(current store.SystemConfig, patch configPatchRequest) store.SystemConfig {
	if patch.TelegramSessionBlob != nil {
		current.TelegramSessionBlob = *patch.TelegramSessionBlob
	}
	if patch.TelegramTargetChatID != nil {
		current.TelegramTargetChatID = *patch.TelegramTargetChatID
	}
	if patch.DefaultChunkSize != nil {
		current.DefaultChunkSize = *patch.DefaultChunkSize
	}
	if patch.MaxStagingBytes != nil {
		current.MaxStagingBytes = *patch.MaxStagingBytes
	}
	if patch.DownloadCacheTTL != nil {
		current.DownloadCacheTTL = *patch.DownloadCacheTTL
	}
	if patch.AppPassword != nil {
		current.AppPassword = *patch.AppPassword
	}
	return current
}
