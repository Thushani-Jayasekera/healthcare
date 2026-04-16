package availability

import (
	"encoding/json"
	"net/http"

	"github.com/healthcare/booking/pkg/apierror"
)

type Handler struct {
	store *Store
}

func NewHandler(s *Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /providers/{providerId}/availability", h.GetAvailability)
}

// GET /providers/{providerId}/availability?date_from=&date_to=&treatment_id=
func (h *Handler) GetAvailability(w http.ResponseWriter, r *http.Request) {
	providerID := r.PathValue("providerId")
	q := r.URL.Query()

	dateFrom := q.Get("date_from")
	dateTo := q.Get("date_to")

	if dateFrom == "" || dateTo == "" {
		apierror.BadRequest(w, "'date_from' and 'date_to' are required")
		return
	}

	slots, nextAvail := h.store.Query(providerID, dateFrom, dateTo)

	resp := AvailabilityResponse{
		ProviderID:        providerID,
		DateFrom:          dateFrom,
		DateTo:            dateTo,
		Slots:             slots,
		NextAvailableDate: nextAvail,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
