package notification

import "time"

// ---- Domain model ---------------------------------------------------------

type Receipt struct {
	NotificationID string    `json:"notification_id"`
	BookingID      string    `json:"booking_id"`
	ChannelsSent   []string  `json:"channels_sent"`
	Status         string    `json:"status"` // queued | delivered | failed
	QueuedAt       time.Time `json:"queued_at"`
}

// ---- Request DTO ----------------------------------------------------------

type SendRequest struct {
	BookingID     string   `json:"booking_id"`
	Channels      []string `json:"channels"`
	OverrideEmail string   `json:"override_email,omitempty"`
	OverridePhone string   `json:"override_phone,omitempty"`
}
