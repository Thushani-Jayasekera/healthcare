package provider

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/healthcare/booking/pkg/apierror"
)

type Handler struct {
	store *Store
}

func NewHandler(s *Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /providers", h.ListProviders)
	mux.HandleFunc("POST /providers", h.CreateProvider)
	mux.HandleFunc("GET /providers/{providerId}", h.GetProvider)
	mux.HandleFunc("PUT /providers/{providerId}", h.UpdateProvider)
	mux.HandleFunc("PATCH /providers/{providerId}/status", h.UpdateProviderStatus)
	mux.HandleFunc("DELETE /providers/{providerId}", h.DeleteProvider)
}

// GET /providers?specialty=&location=&accepting_new_patients=&page=&limit=
func (h *Handler) ListProviders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	specialty := q.Get("specialty")
	if specialty == "" {
		apierror.BadRequest(w, "'specialty' query parameter is required")
		return
	}

	location := q.Get("location")
	accepting := true
	if v := q.Get("accepting_new_patients"); v == "false" {
		accepting = false
	}

	page := queryInt(q.Get("page"), 1)
	limit := queryInt(q.Get("limit"), 10)
	if limit > 50 {
		limit = 50
	}

	resp := h.store.List(specialty, location, accepting, page, limit)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GET /providers/{providerId}
func (h *Handler) GetProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("providerId")
	p, ok := h.store.GetByID(id)
	if !ok {
		apierror.NotFound(w, fmt.Sprintf("Provider %s not found", id))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

// POST /providers
func (h *Handler) CreateProvider(w http.ResponseWriter, r *http.Request) {
	var req CreateProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "Invalid JSON body")
		return
	}

	var fields []apierror.FieldError
	if req.Name == "" {
		fields = append(fields, apierror.FieldError{Field: "name", Message: "required"})
	}
	if req.Type != "doctor" && req.Type != "hospital" && req.Type != "clinic" {
		fields = append(fields, apierror.FieldError{Field: "type", Message: "must be doctor, hospital, or clinic"})
	}
	if req.Specialty == "" {
		fields = append(fields, apierror.FieldError{Field: "specialty", Message: "required"})
	}
	if len(fields) > 0 {
		apierror.Unprocessable(w, "Validation failed", fields...)
		return
	}

	p := h.store.Create(newUUID(), req)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

// PUT /providers/{providerId}
func (h *Handler) UpdateProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("providerId")
	var req UpdateProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "Invalid JSON body")
		return
	}
	p, ok := h.store.Update(id, req)
	if !ok {
		apierror.NotFound(w, fmt.Sprintf("Provider %s not found", id))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

// PATCH /providers/{providerId}/status
func (h *Handler) UpdateProviderStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("providerId")
	var req StatusUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "Invalid JSON body")
		return
	}
	p, ok := h.store.UpdateStatus(id, req.AcceptingNewPatients)
	if !ok {
		apierror.NotFound(w, fmt.Sprintf("Provider %s not found", id))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

// DELETE /providers/{providerId}
func (h *Handler) DeleteProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("providerId")
	if !h.store.Delete(id) {
		apierror.NotFound(w, fmt.Sprintf("Provider %s not found", id))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func newUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func queryInt(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return def
	}
	return v
}
