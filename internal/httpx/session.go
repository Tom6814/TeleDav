package httpx

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
)

const sessionCookieName = "session"

func SetSession(w http.ResponseWriter, token, secret string) {
	value := token
	if secret != "" {
		value = token + "." + sign(token, secret)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func IsAuthorized(r *http.Request, secret string) bool {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	if secret == "" {
		return cookie.Value == "single-user"
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		return false
	}
	return hmac.Equal([]byte(sign(parts[0], secret)), []byte(parts[1]))
}

func sign(token, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}
