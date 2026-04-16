package provider

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

func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /providers", h.ListProviders)
	mux.HandleFunc("GET /providers/{providerId}", h.GetProvider)
	return mux
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
