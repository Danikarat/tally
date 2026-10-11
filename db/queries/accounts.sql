-- name: CreateFinancialAccount :one
INSERT INTO financial_accounts (
    user_id,
    account_type,
    provider_name,
    account_identifier_masked,
    currency,
    balance_minor_units,
    is_default,
    is_active
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING id, user_id, account_type, provider_name, account_identifier_masked, currency, balance_minor_units, is_default, is_active, created_at, updated_at;

-- name: GetFinancialAccountByID :one
SELECT id, user_id, account_type, provider_name, account_identifier_masked, currency, balance_minor_units, is_default, is_active, created_at, updated_at
FROM financial_accounts
WHERE id = $1 AND user_id = $2;

-- name: ListFinancialAccountsByUser :many
SELECT id, user_id, account_type, provider_name, account_identifier_masked, currency, balance_minor_units, is_default, is_active, created_at, updated_at
FROM financial_accounts
WHERE user_id = $1 AND is_active = true
ORDER BY is_default DESC, created_at ASC;

-- name: GetDefaultFinancialAccount :one
SELECT id, user_id, account_type, provider_name, account_identifier_masked, currency, balance_minor_units, is_default, is_active, created_at, updated_at
FROM financial_accounts
WHERE user_id = $1 AND is_default = true AND is_active = true
LIMIT 1;

-- name: FindAccountByProviderAndMaskedIdentifier :one
SELECT id, user_id, account_type, provider_name, account_identifier_masked, currency, balance_minor_units, is_default, is_active, created_at, updated_at
FROM financial_accounts
WHERE user_id = $1 AND provider_name = $2 AND account_identifier_masked = $3
LIMIT 1;

-- name: UpdateAccountBalance :one
UPDATE financial_accounts
SET balance_minor_units = $3,
    updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, account_type, provider_name, account_identifier_masked, currency, balance_minor_units, is_default, is_active, created_at, updated_at;
