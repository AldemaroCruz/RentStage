package httpapi

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/rentstage/rentstage/apps/api/internal/authn"
	"github.com/rentstage/rentstage/apps/api/internal/config"
	"github.com/rentstage/rentstage/apps/api/internal/core/identity"
	"github.com/rentstage/rentstage/apps/api/internal/idutil"
	"github.com/rentstage/rentstage/apps/api/internal/webutil"
)

type middleware func(http.Handler) http.Handler

func chain(handler http.Handler, middlewares ...middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = idutil.NewUUID()
		}
		w.Header().Set("X-Request-ID", requestID)
		ctx := webutil.WithRequestID(r.Context(), requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func recoveryMiddleware(logger *slog.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("panic recovered",
						"request_id", webutil.RequestID(r.Context()),
						"error", recovered,
						"stack", string(debug.Stack()),
					)
					webutil.WriteError(w, r, http.StatusInternalServerError, "internal_error", "An unexpected error occurred.")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *responseRecorder) WriteHeader(status int) {
	if r.status != 0 {
		return
	}
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(payload []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	written, err := r.ResponseWriter.Write(payload)
	r.bytes += written
	return written, err
}

func loggingMiddleware(logger *slog.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			recorder := &responseRecorder{ResponseWriter: w}
			next.ServeHTTP(recorder, r)
			if recorder.status == 0 {
				recorder.status = http.StatusOK
			}
			logger.Info("http request",
				"request_id", webutil.RequestID(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.status,
				"bytes", recorder.bytes,
				"duration_ms", time.Since(started).Milliseconds(),
				"user_id", webutil.UserID(r.Context()),
				"tenant_id", webutil.TenantID(r.Context()),
			)
		})
	}
}

func corsMiddleware(allowedOrigins []string) middleware {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token, X-Request-ID, X-RentStage-Quote-Token")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func csrfMiddleware(cfg config.Config) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}
			if !strings.HasPrefix(r.URL.Path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}
			if !authn.ValidCSRF(r, cfg) {
				webutil.WriteError(w, r, http.StatusForbidden, "csrf_validation_failed", "Request protection token is missing or invalid.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func authenticationMiddleware(service *authn.Service, cfg config.Config) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cfg.SessionCookieName)
			if err != nil || strings.TrimSpace(cookie.Value) == "" {
				webutil.WriteError(w, r, http.StatusUnauthorized, "authentication_required", "Sign in to continue.")
				return
			}
			user, err := service.VerifySession(r.Context(), cookie.Value)
			if err != nil {
				authn.ClearAuthCookies(w, cfg)
				webutil.WriteError(w, r, http.StatusUnauthorized, "invalid_session", "Your session is invalid or expired. Sign in again.")
				return
			}
			ctx := webutil.WithUserID(r.Context(), user.ID)
			ctx = webutil.WithIdentityUID(ctx, user.IdentityUID)
			ctx = webutil.WithUserEmail(ctx, user.Email)
			ctx = webutil.WithUserName(ctx, user.DisplayName)
			ctx = webutil.WithActorID(ctx, user.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func tenantMiddleware(repository *identity.Repository, cfg config.Config) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			workspaces, err := repository.ListWorkspaces(r.Context(), webutil.UserID(r.Context()))
			if err != nil {
				webutil.WriteError(w, r, http.StatusInternalServerError, "workspace_context_failed", "Could not load workspace access.")
				return
			}
			requested := ""
			if cookie, cookieErr := r.Cookie(cfg.TenantCookieName); cookieErr == nil {
				requested = strings.TrimSpace(cookie.Value)
			}
			if requested == "" {
				requested, _ = repository.PreferredTenant(r.Context(), webutil.UserID(r.Context()))
			}
			var selected *identity.Workspace
			for index := range workspaces {
				if workspaces[index].TenantID == requested {
					candidate := workspaces[index]
					selected = &candidate
					break
				}
			}
			if selected == nil && len(workspaces) > 0 {
				candidate := workspaces[0]
				selected = &candidate
			}
			if selected == nil {
				webutil.WriteError(w, r, http.StatusForbidden, "workspace_required", "Create or join a workspace to continue.")
				return
			}
			if requested != selected.TenantID {
				_ = repository.SetActiveTenant(r.Context(), webutil.UserID(r.Context()), selected.TenantID)
				authn.SetTenantCookie(w, cfg, selected.TenantID)
			}
			ctx := webutil.WithTenantID(r.Context(), selected.TenantID)
			ctx = webutil.WithRole(ctx, string(selected.Role))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func permissionMiddleware(permission identity.Permission) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !identity.HasPermission(identity.Role(webutil.Role(r.Context())), permission) {
				webutil.WriteError(w, r, http.StatusForbidden, "permission_denied", "Your role does not allow this operation.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
