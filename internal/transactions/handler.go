package transactions

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Danikarat/tally/internal/auth"
	"github.com/go-chi/chi/v5"
)

// Handler exposes HTTP REST endpoints for transactions.
type Handler struct {
	service   *Service
	jwtSecret string
}

// NewHandler creates a new transactions HTTP handler.
func NewHandler(service *Service, jwtSecret string) *Handler {
	return &Handler{
		service:   service,
		jwtSecret: jwtSecret,
	}
}

// RegisterRoutes registers transaction retrieval and update routes on a Chi router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/v1/transactions", func(r chi.Router) {
		r.Use(auth.RequireAuth(h.jwtSecret))
		r.Get("/", h.handleListTransactions)
		r.Get("/{id}", h.handleGetTransaction)
		r.Patch("/{id}", h.handlePatchTransaction)
	})
}

func (h *Handler) handleListTransactions(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	filter := ListFilter{
		Limit:  20,
		Offset: 0,
	}

	query := r.URL.Query()
	if lStr := query.Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			filter.Limit = l
		}
	}
	if oStr := query.Get("offset"); oStr != "" {
		if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
			filter.Offset = o
		}
	}
	if accID := query.Get("account_id"); accID != "" {
		filter.AccountID = &accID
	}
	if txType := query.Get("type"); txType != "" {
		filter.TransactionType = &txType
	} else if txType := query.Get("transaction_type"); txType != "" {
		filter.TransactionType = &txType
	}

	if fromStr := query.Get("from_date"); fromStr != "" {
		t, err := parseTimestamp(fromStr)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid from_date format, expected RFC3339 or YYYY-MM-DD")
			return
		}
		filter.FromDate = &t
	}

	if toStr := query.Get("to_date"); toStr != "" {
		t, err := parseTimestamp(toStr)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid to_date format, expected RFC3339 or YYYY-MM-DD")
			return
		}
		filter.ToDate = &t
	}

	resp, err := h.service.ListTransactions(r.Context(), userID, filter)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to list transactions")
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleGetTransaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	tx, err := h.service.GetTransactionByID(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, ErrTransactionNotFound) || errors.Is(err, ErrInvalidUUID) {
			h.writeError(w, http.StatusNotFound, "transaction not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "failed to get transaction")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{"transaction": tx})
}

func (h *Handler) handlePatchTransaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")

	var req PatchTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	updated, err := h.service.PatchTransaction(r.Context(), userID, id, req)
	if err != nil {
		if errors.Is(err, ErrTransactionNotFound) || errors.Is(err, ErrInvalidUUID) {
			h.writeError(w, http.StatusNotFound, "transaction not found")
			return
		}
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{"transaction": updated})
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}

func parseTimestamp(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, errors.New("unrecognized date format")
}
