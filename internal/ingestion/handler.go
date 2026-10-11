package ingestion

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Danikarat/tally/internal/auth"
	"github.com/go-chi/chi/v5"
)

// Handler handles HTTP requests for transaction ingestion.
type Handler struct {
	service   *Service
	jwtSecret string
}

// NewHandler returns a new ingestion HTTP handler.
func NewHandler(service *Service, jwtSecret string) *Handler {
	return &Handler{
		service:   service,
		jwtSecret: jwtSecret,
	}
}

// RegisterRoutes registers the ingestion routes on a Chi router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/v1/transactions", func(r chi.Router) {
		r.Use(auth.RequireAuth(h.jwtSecret))
		r.Post("/ingest", h.handleIngest)
	})
}

func (h *Handler) handleIngest(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	details, err := ValidateIngestRequest(req)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmptyBatch), errors.Is(err, ErrBatchTooLarge):
			h.writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrValidationFail):
			h.writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"error":   "validation failed",
				"details": details,
			})
		default:
			h.writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	resp, err := h.service.IngestCandidates(r.Context(), userID, req)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to ingest candidates")
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}
