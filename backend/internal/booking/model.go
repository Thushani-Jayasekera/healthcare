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

type StatusUpdateRequest struct {
	Status string `json:"status"` // completed | no_show
	Notes  string `json:"notes,omitempty"`
}

// ---- Response DTO ---------------------------------------------------------

type ListResponse struct {
	Total int        `json:"total"`
	Page  int        `json:"page"`
	Limit int        `json:"limit"`
	Items []*Booking `json:"items"`
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

func (s *Store) ListByProvider(providerID, status, dateFrom, dateTo string, page, limit int) ListResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var from, to time.Time
	if dateFrom != "" {
		from, _ = time.Parse("2006-01-02", dateFrom)
	}
	if dateTo != "" {
		to, _ = time.Parse("2006-01-02", dateTo)
		to = to.Add(24*time.Hour - time.Second)
	}

	var matched []*Booking
	for _, b := range s.byID {
		if b.ProviderID != providerID {
			continue
		}
		if status != "" && b.Status != status {
			continue
		}
		if !from.IsZero() && b.StartTime.Before(from) {
			continue
		}
		if !to.IsZero() && b.StartTime.After(to) {
			continue
		}
		matched = append(matched, b)
	}

	total := len(matched)
	start := (page - 1) * limit
	if start >= total {
		return ListResponse{Total: total, Page: page, Limit: limit, Items: []*Booking{}}
	}
	end := start + limit
	if end > total {
		end = total
	}
	return ListResponse{Total: total, Page: page, Limit: limit, Items: matched[start:end]}
}

func (s *Store) UpdateStatus(id, status, notes string) (*Booking, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.byID[id]
	if !ok {
		return nil, false
	}
	b.Status = status
	if notes != "" {
		b.Notes = notes
	}
	b.UpdatedAt = time.Now()
	return b, true
}
