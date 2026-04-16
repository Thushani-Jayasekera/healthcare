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
	"github.com/healthcare/booking/internal/treatment"
	"github.com/healthcare/booking/pkg/middleware"
)

// All 7 APIs are served from a single port.
// Each API still has its own OpenAPI spec; they all share the same base URL.
//
//   http://localhost:8080
//     POST   /patients
//     GET    /patients/{patientId}
//     GET    /treatments/search
//     GET    /treatments/{treatmentId}
//     GET    /providers
//     GET    /providers/{providerId}
//     GET    /providers/{providerId}/availability
//     GET    /pricing/estimate
//     POST   /bookings
//     GET    /bookings/{bookingId}
//     POST   /bookings/{bookingId}/cancel
//     POST   /bookings/{bookingId}/reschedule
//     POST   /notifications/confirmation

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
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

	// ------------------------------------------------------------------ //
	// Single server
	// ------------------------------------------------------------------ //
	handler := middleware.Chain(mux,
		middleware.Logger,
		middleware.CORS,
		middleware.JSON,
	)

	log.Printf("Healthcare API listening on %s", addr)
	log.Printf("  Patients API       → POST/GET  /patients")
	log.Printf("  Treatments API     → GET        /treatments/search  /treatments/{id}")
	log.Printf("  Providers API      → GET        /providers  /providers/{id}")
	log.Printf("  Availability API   → GET        /providers/{id}/availability")
	log.Printf("  Pricing API        → GET        /pricing/estimate")
	log.Printf("  Bookings API       → POST/GET   /bookings  /bookings/{id}")
	log.Printf("  Notifications API  → POST       /notifications/confirmation")

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
