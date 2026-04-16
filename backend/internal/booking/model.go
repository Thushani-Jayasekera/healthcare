package booking

import (
	"sync"
	"time"
)

// ---- Domain model ---------------------------------------------------------

type Booking struct {
	BookingID        string    `json:"booking_id"`
	PatientID        string    `json:"patient_id"`
	ProviderID       string    `json:"provider_id"`
	TreatmentID      string    `json:"treatment_id"`
	SlotID           string    `json:"slot_id"`
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time"`
	Status           string    `json:"status"` // confirmed|pending|cancelled|completed|rescheduled
	Notes            string    `json:"notes,omitempty"`
	ReferralID       string    `json:"referral_id,omitempty"`
	ConfirmationCode string    `json:"confirmation_code"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ---- Request DTOs ---------------------------------------------------------

type CreateRequest struct {
	PatientID   string `json:"patient_id"`
	ProviderID  string `json:"provider_id"`
	TreatmentID string `json:"treatment_id"`
	SlotID      string `json:"slot_id"`
	Notes       string `json:"notes,omitempty"`
	ReferralID  string `json:"referral_id,omitempty"`
}

type CancelRequest struct {
	Reason string `json:"reason"`
}

type RescheduleRequest struct {
	NewSlotID string `json:"new_slot_id"`
	Reason    string `json:"reason,omitempty"`
}

// ---- In-memory store ------------------------------------------------------

type Store struct {
	mu   sync.RWMutex
	byID map[string]*Booking
}

func NewStore() *Store {
	return &Store{byID: make(map[string]*Booking)}
}

func (s *Store) Save(b *Booking) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[b.BookingID] = b
}

func (s *Store) GetByID(id string) (*Booking, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.byID[id]
	return b, ok
}
