package audit

import (
	"net/http"
	"strconv"

	"github.com/rentstage/rentstage/apps/api/internal/webutil"
)

type Handler struct {
	repository *Repository
}

func NewHandler(repository *Repository) *Handler {
	return &Handler{repository: repository}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	events, err := h.repository.List(r.Context(), webutil.TenantID(r.Context()), limit)
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "audit_list_failed", "Could not load the audit trail.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, map[string]any{"items": events})
}
