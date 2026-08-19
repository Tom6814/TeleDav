package api

import (
	"encoding/json"
	"net/http"

	"telegram-webdav/internal/store"
)

type phoneRequest struct {
	Phone string `json:"phone"`
}

type codeRequest struct {
	Code string `json:"code"`
}

type passwordRequest struct {
	Password string `json:"password"`
}

type selectChannelRequest struct {
	ChannelID int64 `json:"channel_id"`
}

type createChannelRequest struct {
	Title string `json:"title"`
}

func telegramAuthHandler(deps Dependencies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if deps.TelegramAuth == nil {
			http.Error(w, "telegram auth unavailable", http.StatusServiceUnavailable)
			return
		}
		switch r.URL.Path {
		case "/api/telegram/auth/status":
			writeJSON(w, http.StatusOK, deps.TelegramAuth.Status(r.Context()))
		case "/api/telegram/auth/start":
			var req phoneRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}
			if err := deps.TelegramAuth.Start(r.Context(), req.Phone); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, http.StatusOK, deps.TelegramAuth.Status(r.Context()))
		case "/api/telegram/auth/verify-code":
			var req codeRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}
			if err := deps.TelegramAuth.VerifyCode(r.Context(), req.Code); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := persistTelegramState(r, deps); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, deps.TelegramAuth.Status(r.Context()))
		case "/api/telegram/auth/verify-password":
			var req passwordRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}
			if err := deps.TelegramAuth.VerifyPassword(r.Context(), req.Password); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := persistTelegramState(r, deps); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, deps.TelegramAuth.Status(r.Context()))
		case "/api/telegram/auth/disconnect":
			if err := deps.TelegramAuth.Disconnect(r.Context()); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := clearTelegramState(r, deps); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, deps.TelegramAuth.Status(r.Context()))
		case "/api/telegram/channels":
			channels, err := deps.TelegramAuth.ListChannels(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, channels)
		case "/api/telegram/channels/select":
			var req selectChannelRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}
			if err := deps.TelegramAuth.SelectChannel(r.Context(), req.ChannelID); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := persistTelegramState(r, deps); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, deps.TelegramAuth.Status(r.Context()))
		case "/api/telegram/channels/create":
			var req createChannelRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}
			channel, err := deps.TelegramAuth.CreateChannel(r.Context(), req.Title)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := persistTelegramState(r, deps); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, channel)
		default:
			http.NotFound(w, r)
		}
	})
}

func persistTelegramState(r *http.Request, deps Dependencies) error {
	if deps.ConfigStore == nil {
		return nil
	}
	current, err := deps.ConfigStore.GetSystemConfig(r.Context())
	if err != nil {
		return err
	}
	status := deps.TelegramAuth.Status(r.Context())
	current.TelegramSessionBlob = status.SessionBlob
	current.TelegramTargetChatID = status.SelectedChannelID
	return deps.ConfigStore.UpsertSystemConfig(r.Context(), current)
}

func clearTelegramState(r *http.Request, deps Dependencies) error {
	if deps.ConfigStore == nil {
		return nil
	}
	current, err := deps.ConfigStore.GetSystemConfig(r.Context())
	if err != nil {
		return err
	}
	current.TelegramSessionBlob = ""
	current.TelegramTargetChatID = 0
	return deps.ConfigStore.UpsertSystemConfig(r.Context(), current)
}

func mergeTelegramConfig(current store.SystemConfig, status any) store.SystemConfig {
	return current
}
