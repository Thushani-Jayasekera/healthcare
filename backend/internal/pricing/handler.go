package pricing

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/healthcare/booking/internal/provider"
	"github.com/healthcare/booking/internal/treatment"
	"github.com/healthcare/booking/pkg/apierror"
)

// PricingEstimate is the response body.
type PricingEstimate struct {
	TreatmentID     string    `json:"treatment_id"`
	ProviderID      string    `json:"provider_id"`
	Currency        string    `json:"currency"`
	GrossFee        float64   `json:"gross_fee"`
	MedicareRebate  float64   `json:"medicare_rebate"`
	InsuranceBenefit float64  `json:"insurance_benefit"`
	EstimatedGap    float64   `json:"estimated_gap"`
	OutOfPocket     float64   `json:"out_of_pocket"`
	Disclaimer      string    `json:"disclaimer"`
	ValidUntil      time.Time `json:"valid_until"`
}

// medicareSchedule maps procedure code → rebate amount (AUD).
var medicareSchedule = map[string]float64{
	"MBS-51011": 420.00,
	"MBS-49112": 380.00,
	"MBS-38200": 290.00,
	"MBS-32090": 150.00,
	"MBS-49518": 520.00,
}

// insuranceBenefitByLevel maps policy level → fraction of gap covered.
var insuranceBenefitByLevel = map[string]float64{
	"basic":    0.10,
	"bronze":   0.20,
	"silver":   0.35,
	"gold":     0.50,
	"platinum": 0.70,
}

type Handler struct {
	treatments *treatment.Store
	providers  *provider.Store
}

func NewHandler(t *treatment.Store, p *provider.Store) *Handler {
	return &Handler{treatments: t, providers: p}
}

func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /pricing/estimate", h.GetEstimate)
	return mux
}

// GET /pricing/estimate?treatment_id=&provider_id=&patient_id=
func (h *Handler) GetEstimate(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	treatmentID := q.Get("treatment_id")
	providerID := q.Get("provider_id")

	if treatmentID == "" || providerID == "" {
		apierror.BadRequest(w, "'treatment_id' and 'provider_id' are required")
		return
	}

	t, ok := h.treatments.GetByID(treatmentID)
	if !ok {
		apierror.NotFound(w, "Treatment not found")
		return
	}

	p, ok := h.providers.GetByID(providerID)
	if !ok {
		apierror.NotFound(w, "Provider not found")
		return
	}

	// Gross fee: provider consultation fee + treatment base cost (heuristic)
	grossFee := p.ConsultationFee + float64(t.DurationMin)*10.00

	// Medicare rebate: use the first procedure code's rebate if available
	var medicareRebate float64
	if len(t.ProcedureCodes) > 0 {
		medicareRebate = medicareSchedule[t.ProcedureCodes[0]]
	}

	gap := grossFee - medicareRebate
	if gap < 0 {
		gap = 0
	}

	// Insurance benefit: not applied without patient context — zero by default.
	// A richer implementation would accept patient_id and look up the fund.
	insuranceBenefit := 0.0

	outOfPocket := gap - insuranceBenefit
	if outOfPocket < 0 {
		outOfPocket = 0
	}

	est := PricingEstimate{
		TreatmentID:      treatmentID,
		ProviderID:       providerID,
		Currency:         "AUD",
		GrossFee:         grossFee,
		MedicareRebate:   medicareRebate,
		InsuranceBenefit: insuranceBenefit,
		EstimatedGap:     gap,
		OutOfPocket:      outOfPocket,
		Disclaimer:       "This is an estimate only. Final costs depend on clinical decisions made during the procedure.",
		ValidUntil:       time.Now().Add(24 * time.Hour),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(est)
}
