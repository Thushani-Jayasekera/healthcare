package booking

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/healthcare/booking/internal/availability"
	"github.com/healthcare/booking/pkg/apierror"
)

type Handler struct {
	store     *Store
	availStore *availability.Store
}

func NewHandler(s *Store, av *availability.Store) *Handler {
	return &Handler{store: s, availStore: av}
}

func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /bookings", h.CreateBooking)
	mux.HandleFunc("GET /bookings/{bookingId}", h.GetBooking)
	mux.HandleFunc("POST /bookings/{bookingId}/cancel", h.CancelBooking)
	mux.HandleFunc("POST /bookings/{bookingId}/reschedule", h.RescheduleBooking)
	return mux
}

// POST /bookings
func (h *Handler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "Invalid JSON body")
		return
	}

	var fields []apierror.FieldError
	if req.PatientID == "" {
		fields = append(fields, apierror.FieldError{Field: "patient_id", Message: "required"})
	}
	if req.ProviderID == "" {
		fields = append(fields, apierror.FieldError{Field: "provider_id", Message: "required"})
	}
	if req.TreatmentID == "" {
		fields = append(fields, apierror.FieldError{Field: "treatment_id", Message: "required"})
	}
	if req.SlotID == "" {
		fields = append(fields, apierror.FieldError{Field: "slot_id", Message: "required"})
	}
	if len(fields) > 0 {
		apierror.Unprocessable(w, "Validation failed", fields...)
		return
	}

	// Lookup the slot for start/end times
	slot, ok := h.availStore.GetByID(req.SlotID)
	if !ok {
		apierror.Conflict(w, "Slot not found or no longer available")
		return
	}

	// Atomically book the slot
	if !h.availStore.BookSlot(req.SlotID) {
		apierror.Conflict(w, "This slot is no longer available")
		return
	}

	now := time.Now()
	b := &Booking{
		BookingID:        newUUID(),
		PatientID:        req.PatientID,
		ProviderID:       req.ProviderID,
		TreatmentID:      req.TreatmentID,
		SlotID:           req.SlotID,
		StartTime:        slot.StartTime,
		EndTime:          slot.EndTime,
		Status:           "confirmed",
		Notes:            req.Notes,
		ReferralID:       req.ReferralID,
		ConfirmationCode: newConfirmationCode(),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	h.store.Save(b)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(b)
}

// GET /bookings/{bookingId}
func (h *Handler) GetBooking(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("bookingId")
	b, ok := h.store.GetByID(id)
	if !ok {
		apierror.NotFound(w, fmt.Sprintf("Booking %s not found", id))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(b)
}

// POST /bookings/{bookingId}/cancel
func (h *Handler) CancelBooking(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("bookingId")
	b, ok := h.store.GetByID(id)
	if !ok {
		apierror.NotFound(w, fmt.Sprintf("Booking %s not found", id))
		return
	}

	var req CancelRequest
	json.NewDecoder(r.Body).Decode(&req)
	if strings.TrimSpace(req.Reason) == "" {
		apierror.Unprocessable(w, "Cancellation reason is required",
			apierror.FieldError{Field: "reason", Message: "required"})
		return
	}

	if b.Status == "cancelled" || b.Status == "completed" {
		apierror.Unprocessable(w, fmt.Sprintf("Cannot cancel a booking with status '%s'", b.Status))
		return
	}

	b.Status = "cancelled"
	b.UpdatedAt = time.Now()
	h.store.Save(b)

	// Free up the slot
	h.availStore.FreeSlot(b.SlotID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(b)
}

// POST /bookings/{bookingId}/reschedule
func (h *Handler) RescheduleBooking(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("bookingId")
	b, ok := h.store.GetByID(id)
	if !ok {
		apierror.NotFound(w, fmt.Sprintf("Booking %s not found", id))
		return
	}

	var req RescheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NewSlotID == "" {
		apierror.Unprocessable(w, "new_slot_id is required",
			apierror.FieldError{Field: "new_slot_id", Message: "required"})
		return
	}

	if b.Status == "cancelled" || b.Status == "completed" {
		apierror.Unprocessable(w, fmt.Sprintf("Cannot reschedule a booking with status '%s'", b.Status))
		return
	}

	newSlot, exists := h.availStore.GetByID(req.NewSlotID)
	if !exists {
		apierror.Conflict(w, "New slot not found or no longer available")
		return
	}

	if !h.availStore.BookSlot(req.NewSlotID) {
		apierror.Conflict(w, "The new slot is no longer available")
		return
	}

	// Free the old slot
	h.availStore.FreeSlot(b.SlotID)

	b.SlotID = req.NewSlotID
	b.StartTime = newSlot.StartTime
	b.EndTime = newSlot.EndTime
	b.Status = "rescheduled"
	b.UpdatedAt = time.Now()
	h.store.Save(b)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(b)
}

func newUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func newConfirmationCode() string {
	b := make([]byte, 3)
	rand.Read(b)
	return fmt.Sprintf("HC-%d-%06d", time.Now().Year(), int(b[0])<<16|int(b[1])<<8|int(b[2]))
}
