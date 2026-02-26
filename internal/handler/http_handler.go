package handler

import (
	"encoding/json"
	"net/http"

	"Go-Microservice-Template/internal/model"
	"Go-Microservice-Template/internal/service"

	"github.com/rs/zerolog/log"
)

// HTTPHandler handles REST API requests.
type HTTPHandler struct {
	userService service.UserService
}

// NewHTTPHandler creates a new HTTP handler.
func NewHTTPHandler(us service.UserService) *HTTPHandler {
	return &HTTPHandler{userService: us}
}

// ── Health & System Endpoints ─────────────────────────────

// Health returns basic health status.
func (h *HTTPHandler) Health(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "go-microservice-template",
	})
}

// Readiness checks if all dependencies are available.
func (h *HTTPHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	// In production, check DB and Redis connectivity here
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "ready",
	})
}

// Metrics exposes Prometheus metrics (placeholder for prometheus handler).
func (h *HTTPHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	// In production, use promhttp.Handler() instead
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("# Metrics endpoint - integrate with promhttp.Handler()\n"))
}

// Login authenticates a user and returns a JWT.
func (h *HTTPHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.userService.Login(r.Context(), req, "dev-secret-change-in-production", 24)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

// ── Response Helpers ──────────────────────────────────────

type errorResponse struct {
	Error string `json:"error"`
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Error().Err(err).Msg("failed to encode response")
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, errorResponse{Error: message})
}
