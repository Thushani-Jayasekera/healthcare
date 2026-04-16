package notification

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/healthcare/booking/internal/booking"
	"github.com/healthcare/booking/pkg/apierror"
)

type Handler struct {
	bookings *booking.Store
}

func NewHandler(b *booking.Store) *Handler {
	return &Handler{bookings: b}
}

func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /notifications/confirmation", h.SendConfirmation)
	return mux
}

// POST /notifications/confirmation
func (h *Handler) SendConfirmation(w http.ResponseWriter, r *http.Request) {
	var req SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "Invalid JSON body")
		return
	}

	if req.BookingID == "" {
		apierror.Unprocessable(w, "booking_id is required",
			apierror.FieldError{Field: "booking_id", Message: "required"})
		return
	}
	if len(req.Channels) == 0 {
		apierror.Unprocessable(w, "at least one channel is required",
			apierror.FieldError{Field: "channels", Message: "must include email or sms"})
		return
	}

	b, ok := h.bookings.GetByID(req.BookingID)
	if !ok {
		apierror.NotFound(w, fmt.Sprintf("Booking %s not found", req.BookingID))
		return
	}

	// Simulate async dispatch — in production this publishes to a queue
	go func() {
		for _, ch := range req.Channels {
			switch ch {
			case "email":
				log.Printf("[NOTIFICATION] EMAIL → booking=%s confirmation=%s",
					b.BookingID, b.ConfirmationCode)
			case "sms":
				log.Printf("[NOTIFICATION] SMS → booking=%s confirmation=%s",
					b.BookingID, b.ConfirmationCode)
			}
		}
	}()

	receipt := Receipt{
		NotificationID: newUUID(),
		BookingID:      req.BookingID,
		ChannelsSent:   req.Channels,
		Status:         "queued",
		QueuedAt:       time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(receipt)
}

func newUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
