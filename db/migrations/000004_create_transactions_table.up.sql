CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES financial_accounts(id) ON DELETE CASCADE,
    source_event_id UUID REFERENCES source_events(id) ON DELETE SET NULL,
    amount_minor_units BIGINT NOT NULL CHECK (amount_minor_units >= 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'GHS' CHECK (length(currency) = 3),
    direction VARCHAR(10) NOT NULL CHECK (direction IN ('INBOUND', 'OUTBOUND')),
    transaction_type VARCHAR(30) NOT NULL CHECK (transaction_type IN ('EXPENSE', 'INCOME', 'TRANSFER', 'REFUND', 'FEE', 'CASH_WITHDRAWAL', 'UNKNOWN')),
    counterparty VARCHAR(255),
    counterparty_identifier VARCHAR(100),
    provider_reference VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'COMPLETED' CHECK (status IN ('COMPLETED', 'PENDING', 'FAILED', 'REVERSED')),
    user_category_id UUID,
    user_notes TEXT,
    user_modified BOOLEAN NOT NULL DEFAULT false,
    event_timestamp TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_timestamp ON transactions(user_id, event_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_user_account ON transactions(user_id, account_id);
CREATE INDEX IF NOT EXISTS idx_transactions_user_reference ON transactions(user_id, provider_reference) WHERE provider_reference IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_transactions_source_event ON transactions(source_event_id);
