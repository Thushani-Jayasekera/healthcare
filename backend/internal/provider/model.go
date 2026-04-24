package provider

import (
	"strings"
	"sync"
)

// ---- Domain model ---------------------------------------------------------

type Address struct {
	Street   string `json:"street,omitempty"`
	City     string `json:"city,omitempty"`
	State    string `json:"state,omitempty"`
	Postcode string `json:"postcode,omitempty"`
	Country  string `json:"country,omitempty"`
}

type Provider struct {
	ProviderID           string   `json:"provider_id"`
	Name                 string   `json:"name"`
	Type                 string   `json:"type"` // doctor | hospital | clinic
	Specialty            string   `json:"specialty"`
	Location             Address  `json:"location"`
	Rating               float64  `json:"rating"`
	AcceptingNewPatients bool     `json:"accepting_new_patients"`
	DistanceKm           float64  `json:"distance_km"`
	Qualifications       []string `json:"qualifications,omitempty"`
	Languages            []string `json:"languages,omitempty"`
	Phone                string   `json:"phone,omitempty"`
	Email                string   `json:"email,omitempty"`
	Website              string   `json:"website,omitempty"`
	AcceptedFunds        []string `json:"accepted_funds,omitempty"`
	Services             []string `json:"services,omitempty"`
	ConsultationFee      float64  `json:"consultation_fee,omitempty"`
}

// ---- Response DTO ---------------------------------------------------------

type ListResponse struct {
	Total int         `json:"total"`
	Page  int         `json:"page"`
	Limit int         `json:"limit"`
	Items []*Provider `json:"items"`
}

// ---- In-memory store ------------------------------------------------------

type Store struct {
	mu   sync.RWMutex
	byID map[string]*Provider
	all  []*Provider
}

func NewStore() *Store {
	return &Store{byID: make(map[string]*Provider)}
}

// ---- Request DTOs (admin) -------------------------------------------------

type CreateProviderRequest struct {
	Name                 string   `json:"name"`
	Type                 string   `json:"type"`
	Specialty            string   `json:"specialty"`
	Location             Address  `json:"location"`
	AcceptingNewPatients bool     `json:"accepting_new_patients"`
	Qualifications       []string `json:"qualifications,omitempty"`
	Languages            []string `json:"languages,omitempty"`
	Phone                string   `json:"phone,omitempty"`
	Email                string   `json:"email,omitempty"`
	Website              string   `json:"website,omitempty"`
	AcceptedFunds        []string `json:"accepted_funds,omitempty"`
	Services             []string `json:"services,omitempty"`
	ConsultationFee      float64  `json:"consultation_fee,omitempty"`
}

type UpdateProviderRequest struct {
	Name                 string   `json:"name,omitempty"`
	Type                 string   `json:"type,omitempty"`
	Specialty            string   `json:"specialty,omitempty"`
	Location             *Address `json:"location,omitempty"`
	Rating               *float64 `json:"rating,omitempty"`
	AcceptingNewPatients *bool    `json:"accepting_new_patients,omitempty"`
	DistanceKm           *float64 `json:"distance_km,omitempty"`
	Qualifications       []string `json:"qualifications,omitempty"`
	Languages            []string `json:"languages,omitempty"`
	Phone                string   `json:"phone,omitempty"`
	Email                string   `json:"email,omitempty"`
	Website              string   `json:"website,omitempty"`
	AcceptedFunds        []string `json:"accepted_funds,omitempty"`
	Services             []string `json:"services,omitempty"`
	ConsultationFee      *float64 `json:"consultation_fee,omitempty"`
}

type StatusUpdateRequest struct {
	AcceptingNewPatients bool `json:"accepting_new_patients"`
}

// ---- Store methods --------------------------------------------------------

func (s *Store) Add(p *Provider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[p.ProviderID] = p
	s.all = append(s.all, p)
}

func (s *Store) GetByID(id string) (*Provider, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.byID[id]
	return p, ok
}

func (s *Store) Create(id string, req CreateProviderRequest) *Provider {
	p := &Provider{
		ProviderID:           id,
		Name:                 req.Name,
		Type:                 req.Type,
		Specialty:            req.Specialty,
		Location:             req.Location,
		AcceptingNewPatients: req.AcceptingNewPatients,
		Qualifications:       req.Qualifications,
		Languages:            req.Languages,
		Phone:                req.Phone,
		Email:                req.Email,
		Website:              req.Website,
		AcceptedFunds:        req.AcceptedFunds,
		Services:             req.Services,
		ConsultationFee:      req.ConsultationFee,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[p.ProviderID] = p
	s.all = append(s.all, p)
	return p
}

func (s *Store) Update(id string, req UpdateProviderRequest) (*Provider, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.byID[id]
	if !ok {
		return nil, false
	}
	if req.Name != "" {
		p.Name = req.Name
	}
	if req.Type != "" {
		p.Type = req.Type
	}
	if req.Specialty != "" {
		p.Specialty = req.Specialty
	}
	if req.Location != nil {
		p.Location = *req.Location
	}
	if req.Rating != nil {
		p.Rating = *req.Rating
	}
	if req.AcceptingNewPatients != nil {
		p.AcceptingNewPatients = *req.AcceptingNewPatients
	}
	if req.DistanceKm != nil {
		p.DistanceKm = *req.DistanceKm
	}
	if req.Qualifications != nil {
		p.Qualifications = req.Qualifications
	}
	if req.Languages != nil {
		p.Languages = req.Languages
	}
	if req.Phone != "" {
		p.Phone = req.Phone
	}
	if req.Email != "" {
		p.Email = req.Email
	}
	if req.Website != "" {
		p.Website = req.Website
	}
	if req.AcceptedFunds != nil {
		p.AcceptedFunds = req.AcceptedFunds
	}
	if req.Services != nil {
		p.Services = req.Services
	}
	if req.ConsultationFee != nil {
		p.ConsultationFee = *req.ConsultationFee
	}
	return p, true
}

func (s *Store) UpdateStatus(id string, accepting bool) (*Provider, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.byID[id]
	if !ok {
		return nil, false
	}
	p.AcceptingNewPatients = accepting
	return p, true
}

func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byID[id]; !ok {
		return false
	}
	delete(s.byID, id)
	filtered := s.all[:0]
	for _, p := range s.all {
		if p.ProviderID != id {
			filtered = append(filtered, p)
		}
	}
	s.all = filtered
	return true
}

func (s *Store) List(specialty, location string, acceptingOnly bool, page, limit int) ListResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	spec := strings.ToLower(specialty)
	loc := strings.ToLower(location)

	var matched []*Provider
	for _, p := range s.all {
		if spec != "" && strings.ToLower(p.Specialty) != spec {
			continue
		}
		if loc != "" && !strings.Contains(strings.ToLower(p.Location.City), loc) {
			continue
		}
		if acceptingOnly && !p.AcceptingNewPatients {
			continue
		}
		matched = append(matched, p)
	}

	total := len(matched)
	start := (page - 1) * limit
	if start >= total {
		return ListResponse{Total: total, Page: page, Limit: limit, Items: []*Provider{}}
	}
	end := start + limit
	if end > total {
		end = total
	}
	return ListResponse{Total: total, Page: page, Limit: limit, Items: matched[start:end]}
}

// ---- Seed data ------------------------------------------------------------

func Seed(s *Store) {
	providers := []*Provider{
		{
			ProviderID:           "p2000000-0001-0001-0001-000000000001",
			Name:                 "Dr. Marcus Chen",
			Type:                 "doctor",
			Specialty:            "Orthopedics",
			Location:             Address{Street: "150 Elizabeth St", City: "Sydney", State: "NSW", Postcode: "2000", Country: "AU"},
			Rating:               4.8,
			AcceptingNewPatients: true,
			DistanceKm:           2.1,
			Qualifications:       []string{"MBBS (UNSW)", "FRACS (Ortho)"},
			Languages:            []string{"English", "Mandarin"},
			Phone:                "+61292001001",
			Email:                "dr.chen@sydneyortho.com.au",
			Website:              "https://sydneyortho.com.au/chen",
			AcceptedFunds:        []string{"Medibank", "BUPA", "HCF", "NIB"},
			Services:             []string{"Lumbar Spinal Decompression", "Knee Arthroscopy", "Hip Replacement"},
			ConsultationFee:      280.00,
		},
		{
			ProviderID:           "p2000000-0002-0002-0002-000000000002",
			Name:                 "Dr. Sarah Williams",
			Type:                 "doctor",
			Specialty:            "Orthopedics",
			Location:             Address{Street: "88 Pacific Hwy", City: "Sydney", State: "NSW", Postcode: "2065", Country: "AU"},
			Rating:               4.6,
			AcceptingNewPatients: true,
			DistanceKm:           5.4,
			Qualifications:       []string{"MBBS (USyd)", "FRACS (Ortho)", "Fellow Spine Surgery"},
			Languages:            []string{"English"},
			Phone:                "+61292001002",
			Email:                "s.williams@northsidespine.com.au",
			AcceptedFunds:        []string{"Medibank", "BUPA", "AHM"},
			Services:             []string{"Lumbar Spinal Decompression", "Spinal Fusion"},
			ConsultationFee:      320.00,
		},
		{
			ProviderID:           "p2000000-0003-0003-0003-000000000003",
			Name:                 "Sydney Orthopaedic Hospital",
			Type:                 "hospital",
			Specialty:            "Orthopedics",
			Location:             Address{Street: "200 Macquarie St", City: "Sydney", State: "NSW", Postcode: "2000", Country: "AU"},
			Rating:               4.5,
			AcceptingNewPatients: true,
			DistanceKm:           1.8,
			Qualifications:       []string{"JCI Accredited", "ACHS Certified"},
			Languages:            []string{"English", "Mandarin", "Arabic", "Greek"},
			Phone:                "+61292001003",
			Email:                "bookings@sydortho.com.au",
			Website:              "https://sydortho.com.au",
			AcceptedFunds:        []string{"Medibank", "BUPA", "HCF", "NIB", "AHM", "Westfund"},
			Services:             []string{"All orthopaedic procedures", "Emergency trauma", "Rehabilitation"},
			ConsultationFee:      0,
		},
		{
			ProviderID:           "p2000000-0004-0004-0004-000000000004",
			Name:                 "Dr. James Patel",
			Type:                 "doctor",
			Specialty:            "Cardiology",
			Location:             Address{Street: "45 Park St", City: "Sydney", State: "NSW", Postcode: "2000", Country: "AU"},
			Rating:               4.9,
			AcceptingNewPatients: true,
			DistanceKm:           3.0,
			Qualifications:       []string{"MBBS", "FRACP", "Interventional Cardiology Fellowship"},
			Languages:            []string{"English", "Hindi"},
			Phone:                "+61292001004",
			Email:                "j.patel@heartcare.com.au",
			AcceptedFunds:        []string{"Medibank", "BUPA", "HCF"},
			Services:             []string{"Coronary Angiogram", "Echocardiogram", "Stress Test"},
			ConsultationFee:      350.00,
		},
		{
			ProviderID:           "p2000000-0005-0005-0005-000000000005",
			Name:                 "Dr. Lisa Nguyen",
			Type:                 "doctor",
			Specialty:            "Gastroenterology",
			Location:             Address{Street: "320 Crown St", City: "Sydney", State: "NSW", Postcode: "2010", Country: "AU"},
			Rating:               4.7,
			AcceptingNewPatients: false,
			DistanceKm:           4.2,
			Qualifications:       []string{"MBBS", "FRACP (Gastro)"},
			Languages:            []string{"English", "Vietnamese"},
			Phone:                "+61292001005",
			Email:                "l.nguyen@gutclinic.com.au",
			AcceptedFunds:        []string{"Medibank", "NIB"},
			Services:             []string{"Colonoscopy", "Gastroscopy", "IBD Management"},
			ConsultationFee:      260.00,
		},
	}
	for _, p := range providers {
		s.Add(p)
	}
}
