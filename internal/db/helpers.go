package db

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Text returns a valid pgtype.Text from a string.
func Text(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

// NullText returns an invalid/null pgtype.Text.
func NullText() pgtype.Text {
	return pgtype.Text{Valid: false}
}

// TextOrNull returns a valid pgtype.Text if s is not empty, otherwise null.
func TextOrNull(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

// Timestamptz returns a valid pgtype.Timestamptz from a time.Time.
func Timestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// NullTimestamptz returns a null pgtype.Timestamptz.
func NullTimestamptz() pgtype.Timestamptz {
	return pgtype.Timestamptz{Valid: false}
}

// ParseUUID parses a UUID string into pgtype.UUID.
func ParseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	err := u.Scan(s)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return u, nil
}

// NullUUID returns an invalid/null pgtype.UUID.
func NullUUID() pgtype.UUID {
	return pgtype.UUID{Valid: false}
}
