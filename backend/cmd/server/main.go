package main

import (
	"log"
	"net/http"
	"os"

	"github.com/healthcare/booking/internal/availability"
	"github.com/healthcare/booking/internal/booking"
	"github.com/healthcare/booking/internal/notification"
	"github.com/healthcare/booking/internal/patient"
	"github.com/healthcare/booking/internal/pricing"
	"github.com/healthcare/booking/internal/provider"
	"github.com/healthcare/booking/internal/reports"
	"github.com/healthcare/booking/internal/treatment"
	"github.com/healthcare/booking/pkg/middleware"
)

// All APIs (patient booking + provider admin) are served from a single port.
// Each API has its own OpenAPI spec; they share the same base URL.
//
// Patient / Booking APIs:
//   POST   /patients
//   GET    /patients/{patientId}
//   GET    /treatments/search
//   GET    /treatments/{treatmentId}
//   GET    /providers
//   GET    /providers/{providerId}
//   GET    /providers/{providerId}/availability
//   GET    /pricing/estimate
//   POST   /bookings
//   GET    /bookings/{bookingId}
//   POST   /bookings/{bookingId}/cancel
//   POST   /bookings/{bookingId}/reschedule
//   POST   /notifications/confirmation
//
// Provider Administration APIs:
//   POST   /providers                                          (register provider)
//   PUT    /providers/{providerId}                            (update profile)
//   PATCH  /providers/{providerId}/status                     (toggle accepting)
//   DELETE /providers/{providerId}                            (remove provider)
//   POST   /providers/{providerId}/availability               (add slots)
//   DELETE /providers/{providerId}/availability/{slotId}      (remove slot)
//   PATCH  /providers/{providerId}/availability/{slotId}      (block/unblock slot)
//   GET    /providers/{providerId}/bookings                   (provider dashboard)
//   PATCH  /bookings/{bookingId}/status                       (mark completed/no-show)
//   POST   /treatments                                        (add treatment)
//   PUT    /treatments/{treatmentId}                          (update treatment)
//   DELETE /treatments/{treatmentId}                          (remove treatment)
//   GET    /providers/{providerId}/reports/summary            (admin reporting)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	addr := ":" + port

	// ------------------------------------------------------------------ //
	// Stores
	// ------------------------------------------------------------------ //
	patientStore   := patient.NewStore()
	treatmentStore := treatment.NewStore()
	providerStore  := provider.NewStore()
	availStore     := availability.NewStore()
	bookingStore   := booking.NewStore()

	// ------------------------------------------------------------------ //
	// Seed reference data
	// ------------------------------------------------------------------ //
	treatment.Seed(treatmentStore)
	provider.Seed(providerStore)
	availability.Seed(availStore)

	// ------------------------------------------------------------------ //
	// Handlers
	// ------------------------------------------------------------------ //
	patientH      := patient.NewHandler(patientStore)
	treatmentH    := treatment.NewHandler(treatmentStore)
	providerH     := provider.NewHandler(providerStore)
	availH        := availability.NewHandler(availStore)
	bookingH      := booking.NewHandler(bookingStore, availStore)
	pricingH      := pricing.NewHandler(treatmentStore, providerStore)
	notificationH := notification.NewHandler(bookingStore)
	reportsH      := reports.NewHandler(bookingStore, availStore)

	// ------------------------------------------------------------------ //
	// Single shared mux — all 7 APIs register their routes here
	// ------------------------------------------------------------------ //
	mux := http.NewServeMux()

	patientH.Register(mux)
	treatmentH.Register(mux)
	providerH.Register(mux)
	availH.Register(mux)
	bookingH.Register(mux)
	pricingH.Register(mux)
	notificationH.Register(mux)
	reportsH.Register(mux)

	// ------------------------------------------------------------------ //
	// Single server
	// ------------------------------------------------------------------ //
	handler := middleware.Chain(mux,
		middleware.Logger,
		middleware.CORS,
		middleware.JSON,
	)

	log.Printf("Healthcare API listening on %s", addr)
	log.Printf("  [Booking]  Patients API         → POST/GET  /patients")
	log.Printf("  [Booking]  Treatments API        → GET        /treatments/search  /treatments/{id}")
	log.Printf("  [Booking]  Providers API         → GET        /providers  /providers/{id}")
	log.Printf("  [Booking]  Availability API      → GET        /providers/{id}/availability")
	log.Printf("  [Booking]  Pricing API           → GET        /pricing/estimate")
	log.Printf("  [Booking]  Bookings API          → POST/GET   /bookings  /bookings/{id}")
	log.Printf("  [Booking]  Notifications API     → POST       /notifications/confirmation")
	log.Printf("  [Admin]    Provider Mgmt         → POST/PUT/PATCH/DELETE /providers")
	log.Printf("  [Admin]    Schedule Mgmt         → POST/DELETE/PATCH /providers/{id}/availability/{slotId}")
	log.Printf("  [Admin]    Treatment Mgmt        → POST/PUT/DELETE /treatments")
	log.Printf("  [Admin]    Provider Dashboard    → GET /providers/{id}/bookings")
	log.Printf("  [Admin]    Booking Status        → PATCH /bookings/{id}/status")
	log.Printf("  [Admin]    Reports               → GET /providers/{id}/reports/summary")

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
