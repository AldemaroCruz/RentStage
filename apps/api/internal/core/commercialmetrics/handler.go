package commercialmetrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/rentstage/rentstage/apps/api/internal/webutil"
)

type Handler struct {
	repository *Repository
}

func NewHandler(repository *Repository) *Handler {
	return &Handler{repository: repository}
}

func (handler *Handler) Get(w http.ResponseWriter, r *http.Request) {
	days, valid := reportDays(r.URL.Query().Get("days"))
	if !valid {
		webutil.WriteValidationError(w, r, map[string]string{
			"days": "La ventana debe ser de 7, 30 o 90 días.",
		})
		return
	}
	endAt := time.Now().UTC()
	window := Window{Days: days, StartAt: endAt.AddDate(0, 0, -days), EndAt: endAt}
	report, err := handler.repository.Report(r.Context(), webutil.TenantID(r.Context()), window)
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "commercial_metrics_load_failed", "No fue posible cargar las métricas comerciales.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, report)
}

func reportDays(value string) (int, bool) {
	if value == "" {
		return 30, true
	}
	days, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	switch days {
	case 7, 30, 90:
		return days, true
	default:
		return 0, false
	}
}
