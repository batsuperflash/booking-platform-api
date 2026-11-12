package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/batsuperflash/booking-platform-api/internal/storage"
)

type Handler struct {
	db *storage.Postgres
}

func NewHandler(db *storage.Postgres) *Handler {
	return &Handler{
		db: db,
	}
}

func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.healthHandler)

	mux.HandleFunc("POST /resources", h.createResourceHandler)
	mux.HandleFunc("GET /resources", h.listResourcesHandler)

	mux.HandleFunc("POST /resources/{resource_id}/slots", h.createSlotHandler)
	mux.HandleFunc("GET /resources/{resource_id}/slots", h.listSlotsHandler)
	mux.HandleFunc("GET /resources/{resource_id}/available-slots", h.listAvailableSlotsHandler)

	mux.HandleFunc("POST /bookings", h.createBookingHandler)
	mux.HandleFunc("GET /bookings", h.listBookingsHandler)
	mux.HandleFunc("DELETE /bookings/{booking_id}", h.cancelBookingHandler)

	return mux
}

type HealthResponse struct {
	Status   string `json:"status"`
	App      string `json:"app"`
	Database string `json:"database"`
}

type CreateResourceRequest struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type CreateSlotRequest struct {
	StartsAt string `json:"starts_at"`
	EndsAt   string `json:"ends_at"`
}

type CreateBookingRequest struct {
	SlotID   string `json:"slot_id"`
	UserName string `json:"user_name"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	databaseStatus := "ok"
	statusCode := http.StatusOK

	if err := h.db.Ping(ctx); err != nil {
		databaseStatus = "error"
		statusCode = http.StatusServiceUnavailable
	}

	response := HealthResponse{
		Status:   "ok",
		App:      "booking-platform-api",
		Database: databaseStatus,
	}

	writeJSON(w, statusCode, response)
}

func (h *Handler) createResourceHandler(w http.ResponseWriter, r *http.Request) {
	var request CreateResourceRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	request.Name = strings.TrimSpace(request.Name)
	request.Type = strings.TrimSpace(request.Type)
	request.Description = strings.TrimSpace(request.Description)

	if request.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if request.Type == "" {
		writeError(w, http.StatusBadRequest, "type is required")
		return
	}

	resource, err := h.db.CreateResource(r.Context(), storage.CreateResourceParams{
		Name:        request.Name,
		Type:        request.Type,
		Description: request.Description,
	})
	if err != nil {
		log.Printf("failed to create resource: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create resource")
		return
	}

	writeJSON(w, http.StatusCreated, resource)

}

func (h *Handler) listResourcesHandler(w http.ResponseWriter, r *http.Request) {
	resources, err := h.db.ListResources(r.Context())
	if err != nil {
		log.Printf("failed to list resources: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list resources")
		return
	}
	writeJSON(w, http.StatusOK, resources)
}

func (h *Handler) createSlotHandler(w http.ResponseWriter, r *http.Request) {
	resourceID := r.PathValue("resource_id")
	if strings.TrimSpace(resourceID) == "" {
		writeError(w, http.StatusBadRequest, "resource_id is required")
		return
	}

	var request CreateSlotRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	request.StartsAt = strings.TrimSpace(request.StartsAt)
	request.EndsAt = strings.TrimSpace(request.EndsAt)

	if request.StartsAt == "" {
		writeError(w, http.StatusBadRequest, "starts_at is required")
		return
	}

	if request.EndsAt == "" {
		writeError(w, http.StatusBadRequest, "ends_at is required")
		return
	}

	startsAt, err := time.Parse(time.RFC3339, request.StartsAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "starts_at must be in RFC3339 format")
		return
	}

	endsAt, err := time.Parse(time.RFC3339, request.EndsAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "ends_at must be in RFC3339 format")
		return
	}

	if !endsAt.After(startsAt) {
		writeError(w, http.StatusBadRequest, "ends_at must be after starts_at")
		return
	}

	slot, err := h.db.CreateSlot(r.Context(), storage.CreateSlotParams{
		ResourceID: resourceID,
		StartsAt:   startsAt,
		EndsAt:     endsAt,
	})
	if err != nil {
		log.Printf("failed to create slot: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create slot")
		return
	}
	writeJSON(w, http.StatusCreated, slot)

}

func (h *Handler) listSlotsHandler(w http.ResponseWriter, r *http.Request) {
	resourceID := r.PathValue("resource_id")
	if strings.TrimSpace(resourceID) == "" {
		writeError(w, http.StatusBadRequest, "resource_id is required")
		return
	}

	slots, err := h.db.ListSlotsByResource(r.Context(), resourceID)
	if err != nil {
		log.Printf("failed to list slots: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list slots")
		return
	}

	writeJSON(w, http.StatusOK, slots)
}

func (h *Handler) listAvailableSlotsHandler(w http.ResponseWriter, r *http.Request) {
	resourceID := r.PathValue("resource_id")
	if strings.TrimSpace(resourceID) == "" {
		writeError(w, http.StatusBadRequest, "resource_id is required")
		return
	}

	slots, err := h.db.ListAvailableSlotsByResource(r.Context(), resourceID)
	if err != nil {
		log.Printf("failed to list available slots: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list available slots")
		return
	}

	writeJSON(w, http.StatusOK, slots)
}

func (h *Handler) createBookingHandler(w http.ResponseWriter, r *http.Request) {
	var request CreateBookingRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	request.SlotID = strings.TrimSpace(request.SlotID)
	request.UserName = strings.TrimSpace(request.UserName)

	if request.SlotID == "" {
		writeError(w, http.StatusBadRequest, "slot_id is required")
		return
	}
	if request.UserName == "" {
		writeError(w, http.StatusBadRequest, "user_name is required")
		return
	}

	booking, err := h.db.CreateBooking(r.Context(), storage.CreateBookingParams{
		SlotID:   request.SlotID,
		UserName: request.UserName,
	})
	if err != nil {
		if errors.Is(err, storage.ErrSlotAlreadyBooked) {
			writeError(w, http.StatusConflict, "slot already booked")
			return
		}
		log.Printf("failed to create booking: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create booking")
		return
	}

	writeJSON(w, http.StatusCreated, booking)
}

func (h *Handler) listBookingsHandler(w http.ResponseWriter, r *http.Request) {
	bookings, err := h.db.ListBookings(r.Context())
	if err != nil {
		log.Printf("failed to list bookings: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list bookings")
		return
	}
	writeJSON(w, http.StatusOK, bookings)
}

func (h *Handler) cancelBookingHandler(w http.ResponseWriter, r *http.Request) {
	bookingID := r.PathValue("booking_id")
	if strings.TrimSpace(bookingID) == "" {
		writeError(w, http.StatusBadRequest, "booking_id is required")
		return
	}
	booking, err := h.db.CancelBooking(r.Context(), bookingID)
	if err != nil {
		if errors.Is(err, storage.ErrBookingNotFound) {
			writeError(w, http.StatusNotFound, "booking not found")
			return
		}

		log.Printf("failed to cancel booking: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to cancel booking")
		return
	}

	writeJSON(w, http.StatusOK, booking)
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to encode json response: %v", err)
	}
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, ErrorResponse{
		Error: message,
	})
}
