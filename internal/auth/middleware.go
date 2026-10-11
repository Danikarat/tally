package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type contextKey string

const (
	userIDContextKey contextKey = "user_id"
	claimsContextKey contextKey = "claims"
)

// ContextWithUserID returns a new context containing the authenticated user ID.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDContextKey, userID)
}

// UserIDFromContext extracts the authenticated user ID from context.
func UserIDFromContext(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(userIDContextKey).(string)
	return val, ok && val != ""
}

// MustUserIDFromContext extracts the user ID or panics if not present (only use behind RequireAuth).
func MustUserIDFromContext(ctx context.Context) string {
	userID, ok := UserIDFromContext(ctx)
	if !ok {
		panic("user_id not found in context: RequireAuth middleware missing")
	}
	return userID
}

// ContextWithClaims returns a new context containing the token claims.
func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	ctx = context.WithValue(ctx, claimsContextKey, claims)
	if claims != nil && claims.UserID != "" {
		ctx = ContextWithUserID(ctx, claims.UserID)
	}
	return ctx
}

// ClaimsFromContext extracts the full claims from context.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*Claims)
	return claims, ok && claims != nil
}

// RequireAuth returns an HTTP middleware that verifies JWT tokens in the Authorization header.
func RequireAuth(jwtSecret string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeMiddlewareError(w, http.StatusUnauthorized, "authorization header is required")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeMiddlewareError(w, http.StatusUnauthorized, "bearer token required")
				return
			}

			tokenString := strings.TrimSpace(parts[1])
			if tokenString == "" {
				writeMiddlewareError(w, http.StatusUnauthorized, "bearer token cannot be empty")
				return
			}

			claims, err := ValidateToken(tokenString, jwtSecret)
			if err != nil {
				writeMiddlewareError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := ContextWithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeMiddlewareError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
