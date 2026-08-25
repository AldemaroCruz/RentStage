package webchat

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/rentstage/rentstage/apps/api/internal/webutil"
)

type Handler struct {
	service *Service
	logger  *slog.Logger
}

func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) CreateSession(
	w http.ResponseWriter,
	r *http.Request,
) {
	var input CreateSessionInput

	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(
			w,
			r,
			http.StatusBadRequest,
			"invalid_json",
			"The request body is not valid JSON.",
		)
		return
	}

	item, fields, err := h.service.CreateSession(
		r.Context(),
		r.PathValue("tenantSlug"),
		input,
	)
	if h.writeFailure(w, r, fields, err) {
		return
	}

	setPublicChatHeaders(w)
	webutil.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) GetSession(
	w http.ResponseWriter,
	r *http.Request,
) {
	item, err := h.service.GetSession(
		r.Context(),
		r.PathValue("tenantSlug"),
		r.PathValue("sessionID"),
		r.Header.Get(SessionTokenHeader),
	)
	if h.writeFailure(w, r, nil, err) {
		return
	}

	setPublicChatHeaders(w)
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) SendMessage(
	w http.ResponseWriter,
	r *http.Request,
) {
	var input SendMessageInput

	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(
			w,
			r,
			http.StatusBadRequest,
			"invalid_json",
			"The request body is not valid JSON.",
		)
		return
	}

	item, fields, err := h.service.SendMessage(
		r.Context(),
		r.PathValue("tenantSlug"),
		r.PathValue("sessionID"),
		r.Header.Get(SessionTokenHeader),
		input,
	)
	if h.writeFailure(w, r, fields, err) {
		return
	}

	setPublicChatHeaders(w)
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) writeFailure(
	w http.ResponseWriter,
	r *http.Request,
	fields map[string]string,
	err error,
) bool {
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return true
	}

	switch {
	case errors.Is(err, ErrNotFound),
		errors.Is(err, ErrDisabled):
		webutil.WriteError(
			w,
			r,
			http.StatusNotFound,
			"web_chat_unavailable",
			"Web chat is not available for this workspace.",
		)

	case errors.Is(err, ErrInvalidToken):
		webutil.WriteError(
			w,
			r,
			http.StatusNotFound,
			"web_chat_session_not_found",
			"Web chat session not found.",
		)

	case errors.Is(err, ErrSessionClosed):
		webutil.WriteError(
			w,
			r,
			http.StatusGone,
			"web_chat_session_closed",
			"This web chat session is closed.",
		)

	case errors.Is(err, ErrSessionExpired):
		webutil.WriteError(
			w,
			r,
			http.StatusGone,
			"web_chat_session_expired",
			"This web chat session expired. "+
				"Start a new conversation.",
		)

	case errors.Is(err, ErrRateLimited):
		w.Header().Set("Retry-After", "3600")
		webutil.WriteError(
			w,
			r,
			http.StatusTooManyRequests,
			"web_chat_rate_limited",
			"Too many messages were sent. "+
				"Try again later.",
		)

	case err != nil:
		h.logger.ErrorContext(
			r.Context(),
			"web chat operation failed",
			"request_id", webutil.RequestID(r.Context()),
			"error", err,
		)
		webutil.WriteError(
			w,
			r,
			http.StatusInternalServerError,
			"web_chat_operation_failed",
			"Could not complete the web chat operation.",
		)

	default:
		return false
	}

	return true
}

func setPublicChatHeaders(w http.ResponseWriter) {
	w.Header().Set(
		"Cache-Control",
		"no-store, max-age=0",
	)
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}
