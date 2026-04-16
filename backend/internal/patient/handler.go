package patient

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/healthcare/booking/pkg/apierror"
)

type Handler struct {
	store *Store
}

func NewHandler(s *Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /patients", h.CreatePatient)
	mux.HandleFunc("GET /patients/{patientId}", h.GetPatient)
}

// POST /patients
func (h *Handler) CreatePatient(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "Invalid JSON body")
		return
	}

	// Validate required fields
	var fields []apierror.FieldError
	if strings.TrimSpace(req.FirstName) == "" {
		fields = append(fields, apierror.FieldError{Field: "first_name", Message: "required"})
	}
	if strings.TrimSpace(req.LastName) == "" {
		fields = append(fields, apierror.FieldError{Field: "last_name", Message: "required"})
	}
	if strings.TrimSpace(req.DateOfBirth) == "" {
		fields = append(fields, apierror.FieldError{Field: "date_of_birth", Message: "required"})
	}
	if strings.TrimSpace(req.Email) == "" {
		fields = append(fields, apierror.FieldError{Field: "email", Message: "required"})
	}
	if strings.TrimSpace(req.Phone) == "" {
		fields = append(fields, apierror.FieldError{Field: "phone", Message: "required"})
	}
	if len(fields) > 0 {
		apierror.Unprocessable(w, "Validation failed", fields...)
		return
	}

	p := &Patient{
		PatientID:   newUUID(),
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		DateOfBirth: req.DateOfBirth,
		Email:       strings.ToLower(strings.TrimSpace(req.Email)),
		Phone:       req.Phone,
		Gender:      req.Gender,
		Address:     req.Address,
		Insurance:   req.Insurance,
		CreatedAt:   time.Now(),
	}

	result, existed := h.store.Create(p)
	if existed {
		// Return 409 with the existing patient so callers can extract patient_id
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(result)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

// GET /patients/{patientId}
func (h *Handler) GetPatient(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("patientId")
	p, ok := h.store.GetByID(id)
	if !ok {
		apierror.NotFound(w, fmt.Sprintf("Patient %s not found", id))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func newUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
