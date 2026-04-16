# Backend Design — Healthcare Specialist Booking

## Service Architecture (Go-style modular monolith, splittable to microservices)

```
cmd/
  server/          → main entrypoint, wires all services
internal/
  patient/         → service + repository + handler
  treatment/       → service + repository + handler
  provider/        → service + repository + handler
  availability/    → service + repository + handler
  booking/         → service + repository + handler
  pricing/         → service + repository + handler
  notification/    → service + dispatcher + handler
pkg/
  middleware/      → auth, logging, rate-limit
  errors/          → typed error responses
  db/              → postgres pool, migrations
```

---

## Resource Model & Relationships

```
Patient ──────────────────────────────────────────────────────┐
  patient_id (PK)                                              │
  email (UNIQUE)                                               │
  insurance_fund_id (FK → InsuranceFund)                       │
                                                               │
Treatment ───────────────────────────────────────────────┐    │
  treatment_id (PK)                                       │    │
  specialty                                               │    │
  duration_min                                            │    │
                                                         ▼    ▼
Provider ─────────────────────────────────────► Booking ◄─────
  provider_id (PK)                                booking_id (PK)
  type (doctor|hospital|clinic)                   patient_id (FK)
  specialty                                       provider_id (FK)
  location_id (FK → Location)                     treatment_id (FK)
                                                  slot_id (FK)
ProviderSlot ◄──────────────────────────────────►  status
  slot_id (PK)                                    confirmation_code
  provider_id (FK)
  start_time
  end_time
  status (available|held|booked)
```

---

## API Endpoints

| Method | Path                                  | OperationId                | Service            |
|--------|---------------------------------------|----------------------------|--------------------|
| POST   | /patients                             | createPatient              | PatientService     |
| GET    | /patients/{patientId}                 | getPatient                 | PatientService     |
| GET    | /treatments/search                    | searchTreatments           | TreatmentService   |
| GET    | /treatments/{treatmentId}             | getTreatment               | TreatmentService   |
| GET    | /providers                            | listProviders              | ProviderService    |
| GET    | /providers/{providerId}               | getProvider                | ProviderService    |
| GET    | /providers/{providerId}/availability  | getProviderAvailability    | AvailabilityService|
| GET    | /pricing/estimate                     | getPricingEstimate         | PricingService     |
| POST   | /bookings                             | createBooking              | BookingService     |
| GET    | /bookings/{bookingId}                 | getBooking                 | BookingService     |
| POST   | /bookings/{bookingId}/cancel          | cancelBooking              | BookingService     |
| POST   | /bookings/{bookingId}/reschedule      | rescheduleBooking          | BookingService     |
| POST   | /notifications/confirmation           | sendBookingConfirmation    | NotificationService|

---

## Key Design Decisions

### Slot locking
`ProviderSlot.status` uses optimistic locking via a `version` column.
`createBooking` runs an atomic `UPDATE slots SET status='booked', version=version+1
WHERE slot_id=$1 AND status='available'`.
If `rowsAffected == 0` → return 409.

### Patient deduplication
`createPatient` returns 409 with the existing `patient_id` embedded when the
email already exists. The Arazzo workflow treats 409 identically to 201 so
returning patients are handled without a separate lookup step.

### Pricing is stateless
`PricingService` computes estimates on the fly from:
  - `ProviderFee` table (provider × treatment base fee)
  - `MedicareSchedule` table (MBS item codes → rebate amount)
  - `InsuranceFundBenefit` table (fund × policy level → benefit cap)
No pricing record is persisted; estimates carry a `valid_until` TTL.

### Notification delivery
`NotificationService` publishes a `BookingConfirmedEvent` to an internal
message queue (e.g. Redis Streams or NATS). Separate email and SMS worker
goroutines consume the queue, keeping the HTTP response non-blocking (202).

### Separate APIs, no shared DB coupling
Each service owns its own DB tables. Cross-service data (e.g. provider name
in booking confirmation) is passed by the orchestration layer (Arazzo/agent),
not via foreign key joins across service boundaries. This allows independent
deployment when split to microservices.

---

## Error Response Contract

All errors follow the same envelope:

```json
{
  "error":   "snake_case_code",
  "message": "Human-readable description",
  "fields":  [{ "field": "email", "message": "must be a valid email address" }]
}
```

HTTP status mapping:
- 400 → bad_request (missing/invalid query params)
- 404 → not_found
- 409 → conflict (duplicate patient, slot taken)
- 422 → validation_error (request body fails schema)
- 500 → internal_error
- 503 → service_unavailable (circuit breaker open)

---

## Auth Model (outline)

- Internal service calls: mTLS + shared JWT service token
- Patient-facing calls: Bearer JWT (issued by Auth service, not modelled here)
- Booking confirmation: booking_id + confirmation_code acts as a public receipt
  token for read-only access without login (share-by-link pattern)
