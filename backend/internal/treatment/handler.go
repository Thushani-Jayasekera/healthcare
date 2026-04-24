package treatment

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
	mux.HandleFunc("GET /treatments/search", h.SearchTreatments)
	mux.HandleFunc("POST /treatments", h.CreateTreatment)
	mux.HandleFunc("GET /treatments/{treatmentId}", h.GetTreatment)
	mux.HandleFunc("PUT /treatments/{treatmentId}", h.UpdateTreatment)
	mux.HandleFunc("DELETE /treatments/{treatmentId}", h.DeleteTreatment)
}

// GET /treatments/search?condition=&specialty=&page=&limit=
func (h *Handler) SearchTreatments(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	condition := q.Get("condition")
	specialty := q.Get("specialty")

	if condition == "" && specialty == "" {
		apierror.BadRequest(w, "At least one of 'condition' or 'specialty' is required")
		return
	}

	page := queryInt(q.Get("page"), 1)
	limit := queryInt(q.Get("limit"), 10)
	if limit > 50 {
		limit = 50
	}

	resp := h.store.Search(condition, specialty, page, limit)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GET /treatments/{treatmentId}
func (h *Handler) GetTreatment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("treatmentId")
	t, ok := h.store.GetByID(id)
	if !ok {
		apierror.NotFound(w, fmt.Sprintf("Treatment %s not found", id))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

// POST /treatments
func (h *Handler) CreateTreatment(w http.ResponseWriter, r *http.Request) {
	var req CreateTreatmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "Invalid JSON body")
		return
	}

	var fields []apierror.FieldError
	if req.Name == "" {
		fields = append(fields, apierror.FieldError{Field: "name", Message: "required"})
	}
	if req.Specialty == "" {
		fields = append(fields, apierror.FieldError{Field: "specialty", Message: "required"})
	}
	if req.DurationMin <= 0 {
		fields = append(fields, apierror.FieldError{Field: "duration_min", Message: "must be greater than 0"})
	}
	if len(fields) > 0 {
		apierror.Unprocessable(w, "Validation failed", fields...)
		return
	}

	t := h.store.Create(newUUID(), req)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

// PUT /treatments/{treatmentId}
func (h *Handler) UpdateTreatment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("treatmentId")
	var req UpdateTreatmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "Invalid JSON body")
		return
	}
	t, ok := h.store.Update(id, req)
	if !ok {
		apierror.NotFound(w, fmt.Sprintf("Treatment %s not found", id))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

// DELETE /treatments/{treatmentId}
func (h *Handler) DeleteTreatment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("treatmentId")
	if !h.store.Delete(id) {
		apierror.NotFound(w, fmt.Sprintf("Treatment %s not found", id))
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
