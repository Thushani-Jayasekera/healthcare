package availability

import (
	"fmt"
	"sync"
	"time"
)

// ---- Domain model ---------------------------------------------------------

type Slot struct {
	SlotID     string    `json:"slot_id"`
	ProviderID string    `json:"provider_id"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Status     string    `json:"status"` // available | held | booked
}

// ---- Response DTO ---------------------------------------------------------

type AvailabilityResponse struct {
	ProviderID        string  `json:"provider_id"`
	DateFrom          string  `json:"date_from"`
	DateTo            string  `json:"date_to"`
	Slots             []*Slot `json:"slots"`
	NextAvailableDate *string `json:"next_available_date"`
}

// ---- Request DTOs (admin) -------------------------------------------------

type CreateSlotRequest struct {
	StartTime string `json:"start_time"` // RFC3339
	EndTime   string `json:"end_time"`   // RFC3339
}

type BatchCreateSlotsRequest struct {
	Slots []CreateSlotRequest `json:"slots"`
}

type UpdateSlotRequest struct {
	Status string `json:"status"` // available | blocked
}

// ---- In-memory store ------------------------------------------------------

type Store struct {
	mu         sync.Mutex
	byID       map[string]*Slot   // slot_id → Slot
	byProvider map[string][]*Slot // provider_id → []Slot
	counter    int
}

func NewStore() *Store {
	return &Store{
		byID:       make(map[string]*Slot),
		byProvider: make(map[string][]*Slot),
	}
}

func (s *Store) Add(slot *Slot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[slot.SlotID] = slot
	s.byProvider[slot.ProviderID] = append(s.byProvider[slot.ProviderID], slot)
}

func (s *Store) CreateSlot(providerID, slotID, startRFC, endRFC string) (*Slot, error) {
	start, err := time.Parse(time.RFC3339, startRFC)
	if err != nil {
		return nil, fmt.Errorf("invalid start_time: %w", err)
	}
	end, err := time.Parse(time.RFC3339, endRFC)
	if err != nil {
		return nil, fmt.Errorf("invalid end_time: %w", err)
	}
	if !end.After(start) {
		return nil, fmt.Errorf("end_time must be after start_time")
	}
	slot := &Slot{
		SlotID:     slotID,
		ProviderID: providerID,
		StartTime:  start,
		EndTime:    end,
		Status:     "available",
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[slot.SlotID] = slot
	s.byProvider[slot.ProviderID] = append(s.byProvider[slot.ProviderID], slot)
	return slot, nil
}

func (s *Store) DeleteSlot(slotID string) (bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sl, ok := s.byID[slotID]
	if !ok {
		return false, false
	}
	if sl.Status == "booked" {
		return true, false // found but blocked
	}
	delete(s.byID, slotID)
	filtered := s.byProvider[sl.ProviderID][:0]
	for _, existing := range s.byProvider[sl.ProviderID] {
		if existing.SlotID != slotID {
			filtered = append(filtered, existing)
		}
	}
	s.byProvider[sl.ProviderID] = filtered
	return true, true
}

func (s *Store) UpdateSlotStatus(slotID, status string) (*Slot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sl, ok := s.byID[slotID]
	if !ok {
		return nil, false
	}
	sl.Status = status
	return sl, true
}

func (s *Store) nextSlotID() string {
	s.counter++
	return genSlotID(s.counter)
}

func (s *Store) GetByID(id string) (*Slot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sl, ok := s.byID[id]
	return sl, ok
}

// BookSlot atomically marks a slot as booked. Returns false if not available.
func (s *Store) BookSlot(slotID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sl, ok := s.byID[slotID]
	if !ok || sl.Status != "available" {
		return false
	}
	sl.Status = "booked"
	return true
}

// FreeSlot returns a slot to available (used on booking cancellation).
func (s *Store) FreeSlot(slotID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sl, ok := s.byID[slotID]; ok {
		sl.Status = "available"
	}
}

// Query returns slots for a provider within [from, to] (date inclusive).
// Only slots with status == "available" are included.
func (s *Store) Query(providerID, dateFrom, dateTo string) (slots []*Slot, nextAvail *string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	from, _ := time.Parse("2006-01-02", dateFrom)
	to, _ := time.Parse("2006-01-02", dateTo)
	to = to.Add(24*time.Hour - time.Second) // inclusive end of day

	provSlots := s.byProvider[providerID]
	for _, sl := range provSlots {
		if sl.Status != "available" {
			continue
		}
		if sl.StartTime.Before(from) || sl.StartTime.After(to) {
			// Look for next available beyond range
			if sl.StartTime.After(to) && nextAvail == nil {
				d := sl.StartTime.Format("2006-01-02")
				nextAvail = &d
			}
			continue
		}
		slots = append(slots, sl)
	}
	if slots == nil {
		slots = []*Slot{}
	}
	return
}

// ---- Seed data ------------------------------------------------------------

func Seed(s *Store) {
	base := time.Date(2026, 5, 1, 9, 0, 0, 0, time.FixedZone("AEST", 10*3600))

	providers := []string{
		"p2000000-0001-0001-0001-000000000001",
		"p2000000-0002-0002-0002-000000000002",
		"p2000000-0003-0003-0003-000000000003",
		"p2000000-0004-0004-0004-000000000004",
	}

	s.mu.Lock()
	for _, pid := range providers {
		for day := 0; day < 30; day++ {
			d := base.AddDate(0, 0, day)
			if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
				continue
			}
			for _, hour := range []int{9, 10, 11, 14, 15, 16} {
				start := time.Date(d.Year(), d.Month(), d.Day(), hour, 0, 0, 0, d.Location())
				s.counter++
				slot := &Slot{
					SlotID:     genSlotID(s.counter),
					ProviderID: pid,
					StartTime:  start,
					EndTime:    start.Add(60 * time.Minute),
					Status:     "available",
				}
				s.byID[slot.SlotID] = slot
				s.byProvider[slot.ProviderID] = append(s.byProvider[slot.ProviderID], slot)
			}
		}
	}
	s.mu.Unlock()
}

func genSlotID(n int) string {
	return fmt.Sprintf("s%07d-0000-0000-0000-%012d", n, n)
}
