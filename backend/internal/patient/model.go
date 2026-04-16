package patient

import (
	"sync"
	"time"
)

// ---- Domain models --------------------------------------------------------

type Address struct {
	Street   string `json:"street,omitempty"`
	City     string `json:"city,omitempty"`
	State    string `json:"state,omitempty"`
	Postcode string `json:"postcode,omitempty"`
	Country  string `json:"country,omitempty"`
}

type InsuranceInfo struct {
	FundName    string `json:"fund_name,omitempty"`
	MemberID    string `json:"member_id,omitempty"`
	PolicyLevel string `json:"policy_level,omitempty"`
}

type Patient struct {
	PatientID   string         `json:"patient_id"`
	FirstName   string         `json:"first_name"`
	LastName    string         `json:"last_name"`
	DateOfBirth string         `json:"date_of_birth"`
	Email       string         `json:"email"`
	Phone       string         `json:"phone"`
	Gender      string         `json:"gender,omitempty"`
	Address     *Address       `json:"address,omitempty"`
	Insurance   *InsuranceInfo `json:"insurance,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

// ---- Request / response DTOs ----------------------------------------------

type CreateRequest struct {
	FirstName   string         `json:"first_name"`
	LastName    string         `json:"last_name"`
	DateOfBirth string         `json:"date_of_birth"`
	Email       string         `json:"email"`
	Phone       string         `json:"phone"`
	Gender      string         `json:"gender,omitempty"`
	Address     *Address       `json:"address,omitempty"`
	Insurance   *InsuranceInfo `json:"insurance,omitempty"`
}

// ---- In-memory store ------------------------------------------------------

type Store struct {
	mu         sync.RWMutex
	byID       map[string]*Patient
	emailIndex map[string]string // email → patient_id
}

func NewStore() *Store {
	return &Store{
		byID:       make(map[string]*Patient),
		emailIndex: make(map[string]string),
	}
}

// Save inserts or replaces a patient. Callers must hold the write lock.
func (s *Store) save(p *Patient) {
	s.byID[p.PatientID] = p
	s.emailIndex[p.Email] = p.PatientID
}

func (s *Store) GetByID(id string) (*Patient, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.byID[id]
	return p, ok
}

func (s *Store) GetByEmail(email string) (*Patient, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.emailIndex[email]
	if !ok {
		return nil, false
	}
	return s.byID[id], true
}

// Create inserts a new patient if the email is not taken.
// Returns (patient, false, nil) on success.
// Returns (existing, true, nil) if the email already exists (caller emits 409).
func (s *Store) Create(p *Patient) (*Patient, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, exists := s.emailIndex[p.Email]; exists {
		return s.byID[id], true
	}
	s.save(p)
	return p, false
}
