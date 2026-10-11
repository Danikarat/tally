package transactions

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Danikarat/tally/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service provides methods for reading and updating transactions.
type Service struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

// NewService creates a new transactions query and management service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		pool:    pool,
		queries: db.New(pool),
	}
}

// GetTransactionByID retrieves a single transaction belonging to the specified user.
func (s *Service) GetTransactionByID(ctx context.Context, userID, transactionID string) (TransactionResponse, error) {
	userUUID, err := db.ParseUUID(userID)
	if err != nil {
		return TransactionResponse{}, fmt.Errorf("invalid user id: %w", err)
	}

	txUUID, err := db.ParseUUID(transactionID)
	if err != nil {
		return TransactionResponse{}, ErrInvalidUUID
	}

	record, err := s.queries.GetTransactionByID(ctx, db.GetTransactionByIDParams{
		ID:     txUUID,
		UserID: userUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TransactionResponse{}, ErrTransactionNotFound
		}
		return TransactionResponse{}, fmt.Errorf("failed to fetch transaction: %w", err)
	}

	return FromDBModel(record), nil
}

// ListTransactions retrieves a paginated and filtered list of transactions for a user.
func (s *Service) ListTransactions(ctx context.Context, userID string, filter ListFilter) (ListTransactionsResponse, error) {
	userUUID, err := db.ParseUUID(userID)
	if err != nil {
		return ListTransactionsResponse{}, fmt.Errorf("invalid user id: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	var accountUUID pgtype.UUID
	if filter.AccountID != nil && strings.TrimSpace(*filter.AccountID) != "" {
		accountUUID, err = db.ParseUUID(*filter.AccountID)
		if err != nil {
			return ListTransactionsResponse{}, fmt.Errorf("invalid account_id filter: %w", err)
		}
	}

	var fromDate pgtype.Timestamptz
	if filter.FromDate != nil {
		fromDate = db.Timestamptz(*filter.FromDate)
	}

	var toDate pgtype.Timestamptz
	if filter.ToDate != nil {
		toDate = db.Timestamptz(*filter.ToDate)
	}

	var txType pgtype.Text
	if filter.TransactionType != nil && strings.TrimSpace(*filter.TransactionType) != "" {
		txType = db.Text(strings.ToUpper(strings.TrimSpace(*filter.TransactionType)))
	}

	// Total count for pagination metadata
	total, err := s.queries.CountTransactionsFiltered(ctx, db.CountTransactionsFilteredParams{
		UserID:          userUUID,
		AccountID:       accountUUID,
		FromDate:        fromDate,
		ToDate:          toDate,
		TransactionType: txType,
	})
	if err != nil {
		return ListTransactionsResponse{}, fmt.Errorf("failed to count transactions: %w", err)
	}

	rows, err := s.queries.ListTransactionsFiltered(ctx, db.ListTransactionsFilteredParams{
		UserID:          userUUID,
		Limit:           int32(limit),
		Offset:          int32(offset),
		AccountID:       accountUUID,
		FromDate:        fromDate,
		ToDate:          toDate,
		TransactionType: txType,
	})
	if err != nil {
		return ListTransactionsResponse{}, fmt.Errorf("failed to list transactions: %w", err)
	}

	items := make([]TransactionResponse, len(rows))
	for i, r := range rows {
		items[i] = FromDBModel(r)
	}

	return ListTransactionsResponse{
		Data: items,
		Pagination: PaginationMeta{
			Limit:  limit,
			Offset: offset,
			Total:  total,
		},
	}, nil
}

// PatchTransaction updates user-editable classification fields on a transaction.
func (s *Service) PatchTransaction(ctx context.Context, userID, transactionID string, req PatchTransactionRequest) (TransactionResponse, error) {
	userUUID, err := db.ParseUUID(userID)
	if err != nil {
		return TransactionResponse{}, fmt.Errorf("invalid user id: %w", err)
	}

	txUUID, err := db.ParseUUID(transactionID)
	if err != nil {
		return TransactionResponse{}, ErrInvalidUUID
	}

	// Verify transaction exists and belongs to the authenticated user
	_, err = s.queries.GetTransactionByID(ctx, db.GetTransactionByIDParams{
		ID:     txUUID,
		UserID: userUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TransactionResponse{}, ErrTransactionNotFound
		}
		return TransactionResponse{}, fmt.Errorf("failed to check transaction ownership: %w", err)
	}

	var categoryUUID pgtype.UUID
	if req.UserCategoryID != nil {
		if strings.TrimSpace(*req.UserCategoryID) != "" {
			categoryUUID, err = db.ParseUUID(*req.UserCategoryID)
			if err != nil {
				return TransactionResponse{}, fmt.Errorf("invalid user_category_id: %w", err)
			}
		} else {
			categoryUUID = db.NullUUID()
		}
	}

	var userNotes pgtype.Text
	if req.UserNotes != nil {
		userNotes = db.Text(*req.UserNotes)
	}

	updated, err := s.queries.UpdateTransactionUserFields(ctx, db.UpdateTransactionUserFieldsParams{
		ID:             txUUID,
		UserID:         userUUID,
		UserCategoryID: categoryUUID,
		UserNotes:      userNotes,
	})
	if err != nil {
		return TransactionResponse{}, fmt.Errorf("failed to update transaction: %w", err)
	}

	return FromDBModel(updated), nil
}
