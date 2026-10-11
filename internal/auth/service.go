package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Danikarat/tally/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrEmailTaken         = errors.New("email address is already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrAccountInactive    = errors.New("account has been disabled")
)

// User represents the public user model (excludes password hash).
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// RegisterRequest holds registration inputs.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

// LoginRequest holds login credentials.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse holds the authenticated response.
type AuthResponse struct {
	User      User      `json:"user"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Service provides authentication and user management methods.
type Service struct {
	db  *pgxpool.Pool
	cfg config.AuthConfig
}

// NewService creates a new authentication service.
func NewService(db *pgxpool.Pool, cfg config.AuthConfig) *Service {
	return &Service{
		db:  db,
		cfg: cfg,
	}
}

// Register creates a new user account with Argon2id password hashing and returns an auth token.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.FullName = strings.TrimSpace(req.FullName)

	if err := ValidateEmail(req.Email); err != nil {
		return nil, err
	}
	if err := ValidatePassword(req.Password); err != nil {
		return nil, err
	}

	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	var user User
	query := `
		INSERT INTO users (email, password_hash, full_name, is_active)
		VALUES ($1, $2, $3, true)
		RETURNING id, email, full_name, is_active, created_at
	`

	err = s.db.QueryRow(ctx, query, req.Email, hashedPassword, req.FullName).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&user.IsActive,
		&user.CreatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("failed to create user in database: %w", err)
	}

	token, expiresAt, err := GenerateToken(user.ID, user.Email, s.cfg.JWTSecret, s.cfg.TokenDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &AuthResponse{
		User:      user,
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

// Login validates user credentials and returns an auth token.
func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if req.Email == "" || req.Password == "" {
		return nil, ErrInvalidCredentials
	}

	var user User
	var passwordHash string

	query := `
		SELECT id, email, password_hash, full_name, is_active, created_at
		FROM users
		WHERE email = $1
	`

	err := s.db.QueryRow(ctx, query, req.Email).Scan(
		&user.ID,
		&user.Email,
		&passwordHash,
		&user.FullName,
		&user.IsActive,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	if !user.IsActive {
		return nil, ErrAccountInactive
	}

	match, err := VerifyPassword(req.Password, passwordHash)
	if err != nil || !match {
		return nil, ErrInvalidCredentials
	}

	token, expiresAt, err := GenerateToken(user.ID, user.Email, s.cfg.JWTSecret, s.cfg.TokenDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &AuthResponse{
		User:      user,
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}
