package api

import (
	"encoding/json"
	"net/http"

	"telegram-webdav/internal/httpx"
)

type loginRequest struct {
	Password string `json:"password"`
}

func loginHandler(deps Dependencies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if deps.AppPassword != "" && req.Password != deps.AppPassword {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		httpx.SetSession(w, "single-user", deps.SessionSecret)
		w.WriteHeader(http.StatusNoContent)
	})
}
