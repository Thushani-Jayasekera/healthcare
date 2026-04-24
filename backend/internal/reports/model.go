package reports

// ---- Response DTOs --------------------------------------------------------

type BookingStatusCounts struct {
	Confirmed   int `json:"confirmed"`
	Completed   int `json:"completed"`
	Cancelled   int `json:"cancelled"`
	Rescheduled int `json:"rescheduled"`
	NoShow      int `json:"no_show"`
	Pending     int `json:"pending"`
}

type ProviderSummary struct {
	ProviderID     string              `json:"provider_id"`
	PeriodFrom     string              `json:"period_from"`
	PeriodTo       string              `json:"period_to"`
	TotalBookings  int                 `json:"total_bookings"`
	ByStatus       BookingStatusCounts `json:"by_status"`
	AvailableSlots int                 `json:"available_slots"`
	BookedSlots    int                 `json:"booked_slots"`
	UtilizationPct float64             `json:"utilization_pct"`
}
