package reports

import (
	"encoding/json"
	"math"
	"net/http"
	"time"

	"github.com/healthcare/booking/internal/availability"
	"github.com/healthcare/booking/internal/booking"
)

type Handler struct {
	bookingStore  *booking.Store
	availStore    *availability.Store
}

func NewHandler(b *booking.Store, a *availability.Store) *Handler {
	return &Handler{bookingStore: b, availStore: a}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /providers/{providerId}/reports/summary", h.ProviderSummary)
}

// GET /providers/{providerId}/reports/summary?date_from=&date_to=
func (h *Handler) ProviderSummary(w http.ResponseWriter, r *http.Request) {
	providerID := r.PathValue("providerId")
	q := r.URL.Query()

	dateFrom := q.Get("date_from")
	dateTo := q.Get("date_to")
	if dateFrom == "" {
		dateFrom = time.Now().Format("2006-01-02")
	}
	if dateTo == "" {
		dateTo = time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	}

	// Collect all bookings for this provider in range (large limit for aggregation)
	resp := h.bookingStore.ListByProvider(providerID, "", dateFrom, dateTo, 1, 10000)

	counts := BookingStatusCounts{}
	for _, b := range resp.Items {
		switch b.Status {
		case "confirmed":
			counts.Confirmed++
		case "completed":
			counts.Completed++
		case "cancelled":
			counts.Cancelled++
		case "rescheduled":
			counts.Rescheduled++
		case "no_show":
			counts.NoShow++
		case "pending":
			counts.Pending++
		}
	}

	// Count available vs booked slots in range
	slots, _ := h.availStore.Query(providerID, dateFrom, dateTo)
	available := len(slots)

	// booked slots = total slots in period minus available (approximate via bookings in range)
	bookedSlots := counts.Confirmed + counts.Completed + counts.NoShow + counts.Rescheduled
	totalSlots := available + bookedSlots
	util := 0.0
	if totalSlots > 0 {
		util = math.Round(float64(bookedSlots)/float64(totalSlots)*1000) / 10
	}

	summary := ProviderSummary{
		ProviderID:     providerID,
		PeriodFrom:     dateFrom,
		PeriodTo:       dateTo,
		TotalBookings:  resp.Total,
		ByStatus:       counts,
		AvailableSlots: available,
		BookedSlots:    bookedSlots,
		UtilizationPct: util,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}
