package treatment

import (
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
	mux.HandleFunc("GET /treatments/{treatmentId}", h.GetTreatment)
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
