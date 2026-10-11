package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequireAuth_MiddlewareEdgeCases(t *testing.T) {
	jwtSecret := "super-secret-key-32-bytes-minimum-length"
	userID := "123e4567-e89b-12d3-a456-426614174000"
	email := "test@example.com"

	// Create a test handler protected by RequireAuth
	protectedHandler := RequireAuth(jwtSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, ok := UserIDFromContext(r.Context())
		if !ok || ctxUserID != userID {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		mustID := MustUserIDFromContext(r.Context())
		if mustID != userID {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		claims, ok := ClaimsFromContext(r.Context())
		if !ok || claims.Email != email {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	validToken, _, err := GenerateToken(userID, email, jwtSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate valid token: %v", err)
	}

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "Valid Token",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Missing Authorization Header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Malformed Header Without Bearer Prefix",
			authHeader:     validToken,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Basic Auth Scheme",
			authHeader:     "Basic dXNlcjpwYXNz",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Empty Bearer Token",
			authHeader:     "Bearer   ",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Garbage Token String",
			authHeader:     "Bearer not.a.valid.jwt.token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Tampered Signature",
			authHeader:     "Bearer " + validToken + "corrupted",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rec := httptest.NewRecorder()

			protectedHandler.ServeHTTP(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}
		})
	}

	// Test Expired Token
	t.Run("Expired Token", func(t *testing.T) {
		expiredToken, _, err := GenerateToken(userID, email, jwtSecret, -5*time.Minute)
		if err != nil {
			t.Fatalf("failed to generate expired token: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+expiredToken)
		rec := httptest.NewRecorder()

		protectedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401 for expired token, got %d", rec.Code)
		}
	})

	// Test Token Signed With Different Secret
	t.Run("Wrong Secret Token", func(t *testing.T) {
		wrongSecretToken, _, err := GenerateToken(userID, email, "different-secret-key-at-least-32-chars", 1*time.Hour)
		if err != nil {
			t.Fatalf("failed to generate wrong secret token: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+wrongSecretToken)
		rec := httptest.NewRecorder()

		protectedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401 for token with wrong secret, got %d", rec.Code)
		}
	})
}
