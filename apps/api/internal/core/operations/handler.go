package operations

import (
	"net/http"
	"strings"
	"time"

	"github.com/rentstage/rentstage/apps/api/internal/idutil"
	"github.com/rentstage/rentstage/apps/api/internal/webutil"
)

type Handler struct {
	repository *Repository
}

func NewHandler(repository *Repository) *Handler {
	return &Handler{repository: repository}
}

func (h *Handler) Calendar(w http.ResponseWriter, r *http.Request) {
	from, fromErr := time.Parse(time.RFC3339, strings.TrimSpace(r.URL.Query().Get("from")))
	to, toErr := time.Parse(time.RFC3339, strings.TrimSpace(r.URL.Query().Get("to")))
	fields := map[string]string{}
	if fromErr != nil {
		fields["from"] = "Use an RFC3339 timestamp."
	}
	if toErr != nil {
		fields["to"] = "Use an RFC3339 timestamp."
	}
	if fromErr == nil && toErr == nil {
		if !to.After(from) {
			fields["to"] = "Calendar end must be after start."
		}
		if to.Sub(from) > 370*24*time.Hour {
			fields["to"] = "Calendar ranges cannot exceed 370 days."
		}
	}

	statuses, statusError := parseStatuses(r.URL.Query().Get("status"))
	if statusError != "" {
		fields["status"] = statusError
	}
	var customerID *string
	if value := strings.TrimSpace(r.URL.Query().Get("customer_id")); value != "" {
		if !idutil.IsUUID(value) {
			fields["customer_id"] = "Customer ID is invalid."
		} else {
			customerID = &value
		}
	}
	var resourceID *string
	if value := strings.TrimSpace(r.URL.Query().Get("resource_id")); value != "" {
		if !idutil.IsUUID(value) {
			fields["resource_id"] = "Resource ID is invalid."
		} else {
			resourceID = &value
		}
	}
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}

	result, err := h.repository.Calendar(r.Context(), webutil.TenantID(r.Context()), CalendarFilter{
		From:       from,
		To:         to,
		Statuses:   statuses,
		CustomerID: customerID,
		ResourceID: resourceID,
	})
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "calendar_load_failed", "Could not load the operational calendar.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) Agenda(w http.ResponseWriter, r *http.Request) {
	date := strings.TrimSpace(r.URL.Query().Get("date"))
	if date != "" {
		if _, err := time.Parse("2006-01-02", date); err != nil {
			webutil.WriteValidationError(w, r, map[string]string{"date": "Use a YYYY-MM-DD date."})
			return
		}
	}
	result, err := h.repository.Agenda(r.Context(), webutil.TenantID(r.Context()), date)
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "operations_agenda_failed", "Could not load the daily operations agenda.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) Alerts(w http.ResponseWriter, r *http.Request) {
	result, err := h.repository.Alerts(r.Context(), webutil.TenantID(r.Context()))
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "operations_alerts_failed", "Could not load operational alerts.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, result)
}

func parseStatuses(value string) ([]string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, ""
	}
	valid := map[string]bool{
		"PENDING": true, "CONFIRMED": true, "PREPARING": true, "READY": true,
		"CHECKED_OUT": true, "RETURNED": true, "COMPLETED": true, "CANCELLED": true,
	}
	seen := map[string]struct{}{}
	result := make([]string, 0)
	for _, raw := range strings.Split(value, ",") {
		status := strings.ToUpper(strings.TrimSpace(raw))
		if !valid[status] {
			return nil, "Unsupported reservation status."
		}
		if _, exists := seen[status]; exists {
			continue
		}
		seen[status] = struct{}{}
		result = append(result, status)
	}
	return result, ""
}
