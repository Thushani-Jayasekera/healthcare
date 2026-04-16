package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/healthcare/booking/internal/availability"
	"github.com/healthcare/booking/internal/booking"
	"github.com/healthcare/booking/internal/notification"
	"github.com/healthcare/booking/internal/patient"
	"github.com/healthcare/booking/internal/pricing"
	"github.com/healthcare/booking/internal/provider"
	"github.com/healthcare/booking/internal/treatment"
	"github.com/healthcare/booking/pkg/middleware"
)

// Service port allocation
//
//   8001 — Patients API
//   8002 — Treatments API
//   8003 — Providers API
//   8004 — Availability API
//   8005 — Bookings API
//   8006 — Pricing API
//   8007 — Notifications API

func main() {
	// ------------------------------------------------------------------ //
	// Initialise stores
	// ------------------------------------------------------------------ //
	patientStore      := patient.NewStore()
	treatmentStore    := treatment.NewStore()
	providerStore     := provider.NewStore()
	availStore        := availability.NewStore()
	bookingStore      := booking.NewStore()

	// ------------------------------------------------------------------ //
	// Seed reference data
	// ------------------------------------------------------------------ //
	treatment.Seed(treatmentStore)
	provider.Seed(providerStore)
	availability.Seed(availStore)

	// ------------------------------------------------------------------ //
	// Build handlers (booking and notification take cross-store deps)
	// ------------------------------------------------------------------ //
	patientHandler      := patient.NewHandler(patientStore)
	treatmentHandler    := treatment.NewHandler(treatmentStore)
	providerHandler     := provider.NewHandler(providerStore)
	availHandler        := availability.NewHandler(availStore)
	bookingHandler      := booking.NewHandler(bookingStore, availStore)
	pricingHandler      := pricing.NewHandler(treatmentStore, providerStore)
	notificationHandler := notification.NewHandler(bookingStore)

	// ------------------------------------------------------------------ //
	// Register service servers
	// ------------------------------------------------------------------ //
	services := []struct {
		name string
		addr string
		mux  *http.ServeMux
	}{
		{"Patients API",      ":8001", patientHandler.Routes()},
		{"Treatments API",    ":8002", treatmentHandler.Routes()},
		{"Providers API",     ":8003", providerHandler.Routes()},
		{"Availability API",  ":8004", availHandler.Routes()},
		{"Bookings API",      ":8005", bookingHandler.Routes()},
		{"Pricing API",       ":8006", pricingHandler.Routes()},
		{"Notifications API", ":8007", notificationHandler.Routes()},
	}

	var wg sync.WaitGroup
	for _, svc := range services {
		svc := svc
		wg.Add(1)
		go func() {
			defer wg.Done()
			handler := middleware.Chain(svc.mux,
				middleware.Logger,
				middleware.CORS,
				middleware.JSON,
			)
			log.Printf("Starting %s on %s", svc.name, svc.addr)
			if err := http.ListenAndServe(svc.addr, handler); err != nil {
				log.Fatalf("%s failed: %v", svc.name, err)
			}
		}()
	}

	wg.Wait()
}
