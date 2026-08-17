package webutil

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type ErrorResponse struct {
	Error     string            `json:"error"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if value != nil {
		_ = json.NewEncoder(w).Encode(value)
	}
}

func WriteError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	WriteJSON(w, status, ErrorResponse{
		Error:     code,
		Message:   message,
		RequestID: RequestID(r.Context()),
	})
}

func WriteValidationError(w http.ResponseWriter, r *http.Request, fields map[string]string) {
	WriteJSON(w, http.StatusUnprocessableEntity, ErrorResponse{
		Error:     "validation_error",
		Message:   "Please review the submitted fields.",
		Fields:    fields,
		RequestID: RequestID(r.Context()),
	})
}

func DecodeJSON(r *http.Request, destination any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}
