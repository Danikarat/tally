package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handler handles HTTP requests for authentication.
type Handler struct {
	service   *Service
	jwtSecret string
}

// NewHandler returns a new authentication handler.
func NewHandler(service *Service, jwtSecret string) *Handler {
	return &Handler{
		service:   service,
		jwtSecret: jwtSecret,
	}
}

// RegisterRoutes mounts authentication endpoints on a Chi router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/v1/auth", func(r chi.Router) {
		r.Post("/register", h.handleRegister)
		r.Post("/login", h.handleLogin)

		// Authenticated endpoints
		r.Group(func(r chi.Router) {
			r.Use(RequireAuth(h.jwtSecret))
			r.Get("/me", h.handleGetMe)
		})
	})
}

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	resp, err := h.service.Register(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailTaken):
			h.writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, ErrInvalidEmail),
			errors.Is(err, ErrPasswordTooShort),
			errors.Is(err, ErrPasswordNeedsUpper),
			errors.Is(err, ErrPasswordNeedsLower),
			errors.Is(err, ErrPasswordNeedsDigit):
			h.writeError(w, http.StatusBadRequest, err.Error())
		default:
			h.writeError(w, http.StatusInternalServerError, "failed to register user")
		}
		return
	}

	h.writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	resp, err := h.service.Login(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrAccountInactive) {
			h.writeError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "failed to authenticate")
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleGetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.service.GetUserByID(r.Context(), userID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "user not found")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{"user": user})
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}
