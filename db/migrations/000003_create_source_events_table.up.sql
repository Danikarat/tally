CREATE TABLE IF NOT EXISTS source_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source_type VARCHAR(20) NOT NULL CHECK (source_type IN ('SMS', 'STATEMENT', 'EMAIL', 'MANUAL')),
    provider VARCHAR(50) NOT NULL,
    sender_identifier VARCHAR(100),
    external_reference VARCHAR(100),
    payload_hash VARCHAR(64) NOT NULL,
    raw_payload TEXT,
    parse_status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (parse_status IN ('PENDING', 'PARSED', 'FAILED', 'DUPLICATE_IGNORED', 'REVIEW_REQUIRED', 'NON_FINANCIAL')),
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_source_events_user_payload_hash UNIQUE (user_id, payload_hash)
);

CREATE INDEX IF NOT EXISTS idx_source_events_user_received ON source_events(user_id, received_at DESC);
CREATE INDEX IF NOT EXISTS idx_source_events_user_status ON source_events(user_id, parse_status);
