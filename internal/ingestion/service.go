package ingestion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Danikarat/tally/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service coordinates candidate validation and database persistence.
type Service struct {
	pool *pgxpool.Pool
}

// NewService creates a new ingestion service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// IngestCandidates processes a batch of transaction candidates for an authenticated user.
func (s *Service) IngestCandidates(ctx context.Context, userID string, req IngestRequest) (IngestResponse, error) {
	userUUID, err := db.ParseUUID(userID)
	if err != nil {
		return IngestResponse{}, fmt.Errorf("invalid authenticated user id: %w", err)
	}

	res := IngestResponse{
		TotalReceived: len(req.Candidates),
		Items:         make([]IngestItemResult, 0, len(req.Candidates)),
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return IngestResponse{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	qtx := db.New(tx)

	for i, candidate := range req.Candidates {
		itemRes := s.processCandidate(ctx, qtx, userUUID, i, candidate)
		switch itemRes.Status {
		case StatusIngested:
			res.IngestedCount++
		case StatusDuplicateIgnored:
			res.DuplicateCount++
		case StatusFailed:
			res.FailedCount++
		}
		res.Items = append(res.Items, itemRes)
	}

	if err := tx.Commit(ctx); err != nil {
		return IngestResponse{}, fmt.Errorf("failed to commit batch transaction: %w", err)
	}

	return res, nil
}

func (s *Service) processCandidate(ctx context.Context, q *db.Queries, userUUID pgtype.UUID, idx int, c TransactionCandidate) IngestItemResult {
	payloadHash := s.computePayloadHash(c)

	// 1. Check if source event already exists (deduplication)
	existingEvent, err := q.FindSourceEventByHash(ctx, db.FindSourceEventByHashParams{
		UserID:      userUUID,
		PayloadHash: payloadHash,
	})
	if err == nil {
		srcIDStr := uuidToString(existingEvent.ID)
		var txIDStr *string
		existingTx, txErr := q.FindTransactionBySourceEventID(ctx, db.FindTransactionBySourceEventIDParams{
			UserID:        userUUID,
			SourceEventID: existingEvent.ID,
		})
		if txErr == nil {
			tid := uuidToString(existingTx.ID)
			txIDStr = &tid
		}

		return IngestItemResult{
			Index:         idx,
			Status:        StatusDuplicateIgnored,
			TransactionID: txIDStr,
			SourceEventID: &srcIDStr,
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		errStr := fmt.Sprintf("failed checking duplicate event: %v", err)
		return IngestItemResult{
			Index:  idx,
			Status: StatusFailed,
			Error:  &errStr,
		}
	}

	// 2. Resolve Financial Account
	accountID, err := s.resolveAccountID(ctx, q, userUUID, c)
	if err != nil {
		errStr := fmt.Sprintf("failed resolving financial account: %v", err)
		return IngestItemResult{
			Index:  idx,
			Status: StatusFailed,
			Error:  &errStr,
		}
	}

	// 3. Insert Source Event (with ON CONFLICT DO NOTHING)
	receivedAt := c.EventTimestamp
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}

	srcEvent, err := q.InsertSourceEvent(ctx, db.InsertSourceEventParams{
		UserID:            userUUID,
		SourceType:        strings.ToUpper(strings.TrimSpace(c.SourceType)),
		Provider:          c.Provider,
		SenderIdentifier:  db.TextOrNull(valueOrEmpty(c.SenderIdentifier)),
		ExternalReference: db.TextOrNull(valueOrEmpty(c.ExternalReference)),
		PayloadHash:       payloadHash,
		RawPayload:        db.TextOrNull(valueOrEmpty(c.RawPayload)),
		ParseStatus:       "PARSED",
		ReceivedAt:        db.Timestamptz(receivedAt),
	})
	if err != nil {
		// If ON CONFLICT DO NOTHING returned no rows, it's a conflict
		if errors.Is(err, pgx.ErrNoRows) {
			existingEvt, fErr := q.FindSourceEventByHash(ctx, db.FindSourceEventByHashParams{
				UserID:      userUUID,
				PayloadHash: payloadHash,
			})
			if fErr == nil {
				srcIDStr := uuidToString(existingEvt.ID)
				var txIDStr *string
				existingTx, txErr := q.FindTransactionBySourceEventID(ctx, db.FindTransactionBySourceEventIDParams{
					UserID:        userUUID,
					SourceEventID: existingEvt.ID,
				})
				if txErr == nil {
					tid := uuidToString(existingTx.ID)
					txIDStr = &tid
				}
				return IngestItemResult{
					Index:         idx,
					Status:        StatusDuplicateIgnored,
					TransactionID: txIDStr,
					SourceEventID: &srcIDStr,
				}
			}
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation fallback
			srcIDStr := ""
			return IngestItemResult{
				Index:         idx,
				Status:        StatusDuplicateIgnored,
				SourceEventID: &srcIDStr,
			}
		}

		errStr := fmt.Sprintf("failed inserting source event: %v", err)
		return IngestItemResult{
			Index:  idx,
			Status: StatusFailed,
			Error:  &errStr,
		}
	}

	// 4. Insert Normalized Transaction
	txRecord, err := q.InsertTransaction(ctx, db.InsertTransactionParams{
		UserID:                 userUUID,
		AccountID:              accountID,
		SourceEventID:          srcEvent.ID,
		AmountMinorUnits:       c.AmountMinorUnits,
		Currency:               strings.ToUpper(strings.TrimSpace(c.Currency)),
		Direction:              strings.ToUpper(strings.TrimSpace(c.Direction)),
		TransactionType:        strings.ToUpper(strings.TrimSpace(c.TransactionType)),
		Counterparty:           db.TextOrNull(valueOrEmpty(c.Counterparty)),
		CounterpartyIdentifier: db.TextOrNull(valueOrEmpty(c.CounterpartyIdentifier)),
		ProviderReference:      db.TextOrNull(valueOrEmpty(c.ProviderReference)),
		Status:                 "COMPLETED",
		UserCategoryID:         db.NullUUID(),
		UserNotes:              db.NullText(),
		UserModified:           false,
		EventTimestamp:         db.Timestamptz(c.EventTimestamp),
	})
	if err != nil {
		errStr := fmt.Sprintf("failed inserting transaction: %v", err)
		return IngestItemResult{
			Index:  idx,
			Status: StatusFailed,
			Error:  &errStr,
		}
	}

	txIDStr := uuidToString(txRecord.ID)
	srcIDStr := uuidToString(srcEvent.ID)

	return IngestItemResult{
		Index:         idx,
		Status:        StatusIngested,
		TransactionID: &txIDStr,
		SourceEventID: &srcIDStr,
	}
}

func (s *Service) resolveAccountID(ctx context.Context, q *db.Queries, userUUID pgtype.UUID, c TransactionCandidate) (pgtype.UUID, error) {
	// If explicit AccountID provided, verify and use it
	if c.AccountID != nil && *c.AccountID != "" {
		accUUID, err := db.ParseUUID(*c.AccountID)
		if err == nil {
			acc, err := q.GetFinancialAccountByID(ctx, db.GetFinancialAccountByIDParams{
				ID:     accUUID,
				UserID: userUUID,
			})
			if err == nil && acc.IsActive {
				return acc.ID, nil
			}
		}
	}

	// Try user default account
	defAcc, err := q.GetDefaultFinancialAccount(ctx, userUUID)
	if err == nil && defAcc.IsActive {
		return defAcc.ID, nil
	}

	// Check if any active account exists for the user
	accounts, err := q.ListFinancialAccountsByUser(ctx, userUUID)
	if err == nil && len(accounts) > 0 {
		return accounts[0].ID, nil
	}

	// Auto-provision a primary account for the provider
	providerName := c.Provider
	if strings.EqualFold(providerName, "MTN") {
		providerName = "MTN MoMo"
	} else if strings.EqualFold(providerName, "TELECEL") {
		providerName = "Telecel Cash"
	} else if strings.EqualFold(providerName, "AT") || strings.EqualFold(providerName, "AT_MONEY") {
		providerName = "AT Money"
	}

	currency := strings.ToUpper(strings.TrimSpace(c.Currency))
	if len(currency) != 3 {
		currency = "GHS"
	}

	newAcc, err := q.CreateFinancialAccount(ctx, db.CreateFinancialAccountParams{
		UserID:                  userUUID,
		AccountType:             "MOMO",
		ProviderName:            providerName,
		AccountIdentifierMasked: db.Text("Primary"),
		Currency:                currency,
		BalanceMinorUnits:       0,
		IsDefault:               true,
		IsActive:                true,
	})
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("failed to auto-create default account: %w", err)
	}

	return newAcc.ID, nil
}

func (s *Service) computePayloadHash(c TransactionCandidate) string {
	if c.PayloadHash != nil && strings.TrimSpace(*c.PayloadHash) != "" {
		return strings.TrimSpace(*c.PayloadHash)
	}

	h := sha256.New()
	if c.RawPayload != nil && strings.TrimSpace(*c.RawPayload) != "" {
		h.Write([]byte(*c.RawPayload))
	} else {
		composite := fmt.Sprintf("%s|%s|%s|%d|%s|%s",
			c.SourceType,
			c.Provider,
			c.EventTimestamp.UTC().Format(time.RFC3339),
			c.AmountMinorUnits,
			valueOrEmpty(c.ProviderReference),
			valueOrEmpty(c.ExternalReference),
		)
		h.Write([]byte(composite))
	}

	return hex.EncodeToString(h.Sum(nil))
}

func valueOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	src := u.Bytes
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		src[0:4], src[4:6], src[6:8], src[8:10], src[10:16])
}
