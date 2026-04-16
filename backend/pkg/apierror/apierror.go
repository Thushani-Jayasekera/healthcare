package apierror

import (
	"encoding/json"
	"net/http"
)

type Error struct {
	Error   string       `json:"error"`
	Message string       `json:"message"`
	Fields  []FieldError `json:"fields,omitempty"`
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func Write(w http.ResponseWriter, status int, code, message string, fields ...FieldError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Error{Error: code, Message: message, Fields: fields})
}

func NotFound(w http.ResponseWriter, msg string) {
	Write(w, http.StatusNotFound, "not_found", msg)
}

func BadRequest(w http.ResponseWriter, msg string) {
	Write(w, http.StatusBadRequest, "bad_request", msg)
}

func Conflict(w http.ResponseWriter, msg string) {
	Write(w, http.StatusConflict, "conflict", msg)
}

func Unprocessable(w http.ResponseWriter, msg string, fields ...FieldError) {
	Write(w, http.StatusUnprocessableEntity, "validation_error", msg, fields...)
}

func Internal(w http.ResponseWriter) {
	Write(w, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
}
