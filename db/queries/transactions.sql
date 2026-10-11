-- name: InsertTransaction :one
INSERT INTO transactions (
    user_id,
    account_id,
    source_event_id,
    amount_minor_units,
    currency,
    direction,
    transaction_type,
    counterparty,
    counterparty_identifier,
    provider_reference,
    status,
    user_category_id,
    user_notes,
    user_modified,
    event_timestamp
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
)
RETURNING id, user_id, account_id, source_event_id, amount_minor_units, currency, direction, transaction_type, counterparty, counterparty_identifier, provider_reference, status, user_category_id, user_notes, user_modified, event_timestamp, created_at, updated_at;

-- name: FindTransactionByReference :one
SELECT id, user_id, account_id, source_event_id, amount_minor_units, currency, direction, transaction_type, counterparty, counterparty_identifier, provider_reference, status, user_category_id, user_notes, user_modified, event_timestamp, created_at, updated_at
FROM transactions
WHERE user_id = $1 AND provider_reference = $2;

-- name: GetTransactionByID :one
SELECT id, user_id, account_id, source_event_id, amount_minor_units, currency, direction, transaction_type, counterparty, counterparty_identifier, provider_reference, status, user_category_id, user_notes, user_modified, event_timestamp, created_at, updated_at
FROM transactions
WHERE id = $1 AND user_id = $2;

-- name: ListTransactionsByUser :many
SELECT id, user_id, account_id, source_event_id, amount_minor_units, currency, direction, transaction_type, counterparty, counterparty_identifier, provider_reference, status, user_category_id, user_notes, user_modified, event_timestamp, created_at, updated_at
FROM transactions
WHERE user_id = $1
ORDER BY event_timestamp DESC
LIMIT $2 OFFSET $3;

-- name: ListTransactionsByAccount :many
SELECT id, user_id, account_id, source_event_id, amount_minor_units, currency, direction, transaction_type, counterparty, counterparty_identifier, provider_reference, status, user_category_id, user_notes, user_modified, event_timestamp, created_at, updated_at
FROM transactions
WHERE user_id = $1 AND account_id = $2
ORDER BY event_timestamp DESC
LIMIT $3 OFFSET $4;

-- name: UpdateTransactionStatus :one
UPDATE transactions
SET status = $3,
    updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, account_id, source_event_id, amount_minor_units, currency, direction, transaction_type, counterparty, counterparty_identifier, provider_reference, status, user_category_id, user_notes, user_modified, event_timestamp, created_at, updated_at;
