package availability

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
	mux.HandleFunc("POST /providers/{providerId}/availability", h.CreateSlots)
	mux.HandleFunc("DELETE /providers/{providerId}/availability/{slotId}", h.DeleteSlot)
	mux.HandleFunc("PATCH /providers/{providerId}/availability/{slotId}", h.UpdateSlot)
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

// POST /providers/{providerId}/availability
func (h *Handler) CreateSlots(w http.ResponseWriter, r *http.Request) {
	providerID := r.PathValue("providerId")

	var req BatchCreateSlotsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "Invalid JSON body")
		return
	}
	if len(req.Slots) == 0 {
		apierror.Unprocessable(w, "At least one slot is required",
			apierror.FieldError{Field: "slots", Message: "required, min 1 item"})
		return
	}

	created := make([]*Slot, 0, len(req.Slots))
	for i, s := range req.Slots {
		h.store.mu.Lock()
		h.store.counter++
		slotID := genSlotID(h.store.counter)
		h.store.mu.Unlock()

		slot, err := h.store.CreateSlot(providerID, slotID, s.StartTime, s.EndTime)
		if err != nil {
			apierror.Unprocessable(w, fmt.Sprintf("Slot %d: %s", i, err.Error()))
			return
		}
		created = append(created, slot)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"created": len(created),
		"slots":   created,
	})
}

// DELETE /providers/{providerId}/availability/{slotId}
func (h *Handler) DeleteSlot(w http.ResponseWriter, r *http.Request) {
	slotID := r.PathValue("slotId")
	found, deleted := h.store.DeleteSlot(slotID)
	if !found {
		apierror.NotFound(w, fmt.Sprintf("Slot %s not found", slotID))
		return
	}
	if !deleted {
		apierror.Conflict(w, "Cannot delete a slot that is already booked")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PATCH /providers/{providerId}/availability/{slotId}
func (h *Handler) UpdateSlot(w http.ResponseWriter, r *http.Request) {
	slotID := r.PathValue("slotId")

	var req UpdateSlotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "Invalid JSON body")
		return
	}
	req.Status = strings.TrimSpace(req.Status)
	if req.Status != "available" && req.Status != "blocked" {
		apierror.Unprocessable(w, "status must be 'available' or 'blocked'",
			apierror.FieldError{Field: "status", Message: "must be available or blocked"})
		return
	}

	slot, ok := h.store.UpdateSlotStatus(slotID, req.Status)
	if !ok {
		apierror.NotFound(w, fmt.Sprintf("Slot %s not found", slotID))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(slot)
}
