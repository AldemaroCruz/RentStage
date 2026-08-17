package authn

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/rentstage/rentstage/apps/api/internal/config"
)

func SetSessionCookie(w http.ResponseWriter, cfg config.Config, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cfg.SessionCookieName,
		Value:    value,
		Path:     "/",
		Expires:  time.Now().Add(cfg.SessionDuration),
		MaxAge:   int(cfg.SessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func SetTenantCookie(w http.ResponseWriter, cfg config.Config, tenantID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cfg.TenantCookieName,
		Value:    tenantID,
		Path:     "/",
		Expires:  time.Now().Add(cfg.SessionDuration),
		MaxAge:   int(cfg.SessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearAuthCookies(w http.ResponseWriter, cfg config.Config) {
	for _, name := range []string{cfg.SessionCookieName, cfg.TenantCookieName, cfg.CSRFCookieName} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			Expires:  time.Unix(1, 0),
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   cfg.CookieSecure,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

func IssueCSRFToken(w http.ResponseWriter, cfg config.Config) (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(value)
	http.SetCookie(w, &http.Cookie{
		Name:     cfg.CSRFCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int((2 * time.Hour).Seconds()),
		Expires:  time.Now().Add(2 * time.Hour),
		HttpOnly: true,
		Secure:   cfg.CookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
	return token, nil
}

func ValidCSRF(r *http.Request, cfg config.Config) bool {
	cookie, err := r.Cookie(cfg.CSRFCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	header := r.Header.Get("X-CSRF-Token")
	if len(header) != len(cookie.Value) || header == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(header), []byte(cookie.Value)) == 1
}
