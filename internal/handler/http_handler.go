package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"Go-Microservice-Template/internal/model"
	"Go-Microservice-Template/internal/repository"
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

// Register creates a new user account.
func (h *HTTPHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Name == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "email, name, and password are required")
		return
	}

	if len(req.Password) < 8 {
		respondError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	user, err := h.userService.Register(r.Context(), req)
	if err != nil {
		if err == repository.ErrDuplicate {
			respondError(w, http.StatusConflict, "email already registered")
			return
		}
		log.Error().Err(err).Msg("register user failed")
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusCreated, user)
}

// CreateUser creates a new user (admin only).
func (h *HTTPHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req model.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.userService.Register(r.Context(), req)
	if err != nil {
		if err == repository.ErrDuplicate {
			respondError(w, http.StatusConflict, "email already exists")
			return
		}
		log.Error().Err(err).Msg("create user failed")
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusCreated, user)
}

// ListUsers returns a paginated list of users.
func (h *HTTPHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	params := model.DefaultListParams()

	if v := r.URL.Query().Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			params.Page = p
		}
	}
	if v := r.URL.Query().Get("page_size"); v != "" {
		if ps, err := strconv.Atoi(v); err == nil && ps > 0 && ps <= 100 {
			params.PageSize = ps
		}
	}
	if v := r.URL.Query().Get("search"); v != "" {
		params.Search = v
	}
	if v := r.URL.Query().Get("sort_by"); v != "" {
		params.SortBy = v
	}
	if v := r.URL.Query().Get("sort_dir"); v != "" {
		params.SortDir = v
	}

	result, err := h.userService.List(r.Context(), params)
	if err != nil {
		log.Error().Err(err).Msg("list users failed")
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, result)
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
