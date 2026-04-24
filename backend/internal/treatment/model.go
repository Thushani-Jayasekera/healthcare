package treatment

import (
	"strings"
	"sync"
	"time"
)

// ---- Domain model ---------------------------------------------------------

type Treatment struct {
	TreatmentID        string    `json:"treatment_id"`
	Name               string    `json:"name"`
	Specialty          string    `json:"specialty"`
	Description        string    `json:"description"`
	DurationMin        int       `json:"duration_min"`
	ProcedureCodes     []string  `json:"procedure_codes"`
	Prerequisites      []string  `json:"prerequisites"`
	RecoveryDays       int       `json:"typical_recovery_days"`
	RelatedSpecialties []string  `json:"related_specialties"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// ---- Search response DTO --------------------------------------------------

type SearchResponse struct {
	Total int          `json:"total"`
	Page  int          `json:"page"`
	Limit int          `json:"limit"`
	Items []*Treatment `json:"items"`
}

// ---- Request DTOs (admin) -------------------------------------------------

type CreateTreatmentRequest struct {
	Name               string   `json:"name"`
	Specialty          string   `json:"specialty"`
	Description        string   `json:"description"`
	DurationMin        int      `json:"duration_min"`
	ProcedureCodes     []string `json:"procedure_codes,omitempty"`
	Prerequisites      []string `json:"prerequisites,omitempty"`
	RecoveryDays       int      `json:"typical_recovery_days,omitempty"`
	RelatedSpecialties []string `json:"related_specialties,omitempty"`
}

type UpdateTreatmentRequest struct {
	Name               string   `json:"name,omitempty"`
	Specialty          string   `json:"specialty,omitempty"`
	Description        string   `json:"description,omitempty"`
	DurationMin        *int     `json:"duration_min,omitempty"`
	ProcedureCodes     []string `json:"procedure_codes,omitempty"`
	Prerequisites      []string `json:"prerequisites,omitempty"`
	RecoveryDays       *int     `json:"typical_recovery_days,omitempty"`
	RelatedSpecialties []string `json:"related_specialties,omitempty"`
}

// ---- In-memory store ------------------------------------------------------

type Store struct {
	mu   sync.RWMutex
	byID map[string]*Treatment
	all  []*Treatment
}

func NewStore() *Store {
	return &Store{byID: make(map[string]*Treatment)}
}

func (s *Store) Add(t *Treatment) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[t.TreatmentID] = t
	s.all = append(s.all, t)
}

func (s *Store) GetByID(id string) (*Treatment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.byID[id]
	return t, ok
}

func (s *Store) Create(id string, req CreateTreatmentRequest) *Treatment {
	t := &Treatment{
		TreatmentID:        id,
		Name:               req.Name,
		Specialty:          req.Specialty,
		Description:        req.Description,
		DurationMin:        req.DurationMin,
		ProcedureCodes:     req.ProcedureCodes,
		Prerequisites:      req.Prerequisites,
		RecoveryDays:       req.RecoveryDays,
		RelatedSpecialties: req.RelatedSpecialties,
		UpdatedAt:          time.Now(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[t.TreatmentID] = t
	s.all = append(s.all, t)
	return t
}

func (s *Store) Update(id string, req UpdateTreatmentRequest) (*Treatment, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.byID[id]
	if !ok {
		return nil, false
	}
	if req.Name != "" {
		t.Name = req.Name
	}
	if req.Specialty != "" {
		t.Specialty = req.Specialty
	}
	if req.Description != "" {
		t.Description = req.Description
	}
	if req.DurationMin != nil {
		t.DurationMin = *req.DurationMin
	}
	if req.ProcedureCodes != nil {
		t.ProcedureCodes = req.ProcedureCodes
	}
	if req.Prerequisites != nil {
		t.Prerequisites = req.Prerequisites
	}
	if req.RecoveryDays != nil {
		t.RecoveryDays = *req.RecoveryDays
	}
	if req.RelatedSpecialties != nil {
		t.RelatedSpecialties = req.RelatedSpecialties
	}
	t.UpdatedAt = time.Now()
	return t, true
}

func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byID[id]; !ok {
		return false
	}
	delete(s.byID, id)
	filtered := s.all[:0]
	for _, t := range s.all {
		if t.TreatmentID != id {
			filtered = append(filtered, t)
		}
	}
	s.all = filtered
	return true
}

func (s *Store) Search(condition, specialty string, page, limit int) SearchResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cond := strings.ToLower(condition)
	spec := strings.ToLower(specialty)

	var matched []*Treatment
	for _, t := range s.all {
		if spec != "" && strings.ToLower(t.Specialty) != spec {
			continue
		}
		if cond != "" {
			haystack := strings.ToLower(t.Name + " " + t.Description + " " + t.Specialty)
			if !strings.Contains(haystack, cond) {
				continue
			}
		}
		matched = append(matched, t)
	}

	total := len(matched)
	start := (page - 1) * limit
	if start >= total {
		return SearchResponse{Total: total, Page: page, Limit: limit, Items: []*Treatment{}}
	}
	end := start + limit
	if end > total {
		end = total
	}
	return SearchResponse{Total: total, Page: page, Limit: limit, Items: matched[start:end]}
}

// ---- Seed data ------------------------------------------------------------

func Seed(s *Store) {
	now := time.Now()
	treatments := []*Treatment{
		{
			TreatmentID:        "t1000000-0001-0001-0001-000000000001",
			Name:               "Lumbar Spinal Decompression",
			Specialty:          "Orthopedics",
			Description:        "Surgical procedure to relieve pressure on spinal nerves caused by herniated discs, bone spurs, or chronic back pain.",
			DurationMin:        90,
			ProcedureCodes:     []string{"MBS-51011"},
			Prerequisites:      []string{"GP referral required", "MRI within 12 months"},
			RecoveryDays:       21,
			RelatedSpecialties: []string{"Neurosurgery", "Physiotherapy"},
			UpdatedAt:          now,
		},
		{
			TreatmentID:        "t1000000-0002-0002-0002-000000000002",
			Name:               "Knee Arthroscopy",
			Specialty:          "Orthopedics",
			Description:        "Minimally invasive surgery to diagnose and treat knee joint problems including meniscus tears.",
			DurationMin:        60,
			ProcedureCodes:     []string{"MBS-49112"},
			Prerequisites:      []string{"GP referral required", "X-ray or MRI recommended"},
			RecoveryDays:       14,
			RelatedSpecialties: []string{"Sports Medicine", "Physiotherapy"},
			UpdatedAt:          now,
		},
		{
			TreatmentID:        "t1000000-0003-0003-0003-000000000003",
			Name:               "Coronary Angiogram",
			Specialty:          "Cardiology",
			Description:        "Imaging procedure using contrast dye and X-rays to visualise coronary arteries.",
			DurationMin:        45,
			ProcedureCodes:     []string{"MBS-38200"},
			Prerequisites:      []string{"Cardiologist referral", "Blood tests within 30 days"},
			RecoveryDays:       1,
			RelatedSpecialties: []string{"Interventional Cardiology"},
			UpdatedAt:          now,
		},
		{
			TreatmentID:        "t1000000-0004-0004-0004-000000000004",
			Name:               "Colonoscopy",
			Specialty:          "Gastroenterology",
			Description:        "Endoscopic examination of the large intestine for polyps, inflammation, or cancer screening.",
			DurationMin:        30,
			ProcedureCodes:     []string{"MBS-32090"},
			Prerequisites:      []string{"GP referral required", "Bowel preparation the day before"},
			RecoveryDays:       1,
			RelatedSpecialties: []string{"Oncology"},
			UpdatedAt:          now,
		},
		{
			TreatmentID:        "t1000000-0005-0005-0005-000000000005",
			Name:               "Hip Replacement",
			Specialty:          "Orthopedics",
			Description:        "Total hip arthroplasty to replace a damaged hip joint with a prosthetic implant.",
			DurationMin:        120,
			ProcedureCodes:     []string{"MBS-49518"},
			Prerequisites:      []string{"Orthopaedic referral", "Pre-operative bloods and ECG"},
			RecoveryDays:       42,
			RelatedSpecialties: []string{"Anaesthesiology", "Physiotherapy"},
			UpdatedAt:          now,
		},
	}
	for _, t := range treatments {
		s.Add(t)
	}
}
