-- name: InsertSourceEvent :one
INSERT INTO source_events (
    user_id,
    source_type,
    provider,
    sender_identifier,
    external_reference,
    payload_hash,
    raw_payload,
    parse_status,
    received_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
ON CONFLICT (user_id, payload_hash) DO NOTHING
RETURNING id, user_id, source_type, provider, sender_identifier, external_reference, payload_hash, raw_payload, parse_status, received_at, created_at;

-- name: FindSourceEventByHash :one
SELECT id, user_id, source_type, provider, sender_identifier, external_reference, payload_hash, raw_payload, parse_status, received_at, created_at
FROM source_events
WHERE user_id = $1 AND payload_hash = $2;

-- name: GetSourceEventByID :one
SELECT id, user_id, source_type, provider, sender_identifier, external_reference, payload_hash, raw_payload, parse_status, received_at, created_at
FROM source_events
WHERE id = $1 AND user_id = $2;

-- name: UpdateSourceEventStatus :one
UPDATE source_events
SET parse_status = $3
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, source_type, provider, sender_identifier, external_reference, payload_hash, raw_payload, parse_status, received_at, created_at;

-- name: ListSourceEventsByUser :many
SELECT id, user_id, source_type, provider, sender_identifier, external_reference, payload_hash, raw_payload, parse_status, received_at, created_at
FROM source_events
WHERE user_id = $1
ORDER BY received_at DESC
LIMIT $2 OFFSET $3;
