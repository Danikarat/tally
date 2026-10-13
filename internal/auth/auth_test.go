package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Danikarat/tally/internal/config"
	"github.com/Danikarat/tally/internal/database"
	"github.com/go-chi/chi/v5"
)

func setupTestServer(t *testing.T) (*httptest.Server, *Service, *config.Config) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx, cfg.DB.DSN())
	if err != nil {
		t.Skipf("skipping test: database not reachable at %s (%v)", cfg.DB.DSN(), err)
		return nil, nil, nil
	}

	// Ensure tables are migrated
	if err := database.MigrateUp(cfg.DB.DSN()); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Clean users table before test
	_, _ = pool.Exec(ctx, "DELETE FROM users")

	authService := NewService(pool, cfg.Auth)
	handler := NewHandler(authService, cfg.Auth.JWTSecret)

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	ts := httptest.NewServer(router)
	return ts, authService, cfg
}

func TestAuth_RegisterAndLogin_Lifecycle(t *testing.T) {
	ts, _, _ := setupTestServer(t)
	if ts == nil {
		return
	}
	defer ts.Close()

	client := ts.Client()

	// 1. Successful Registration
	registerBody := map[string]string{
		"email":     "kwame.mensah@example.com",
		"password":  "StrongP@ssw0rd1",
		"full_name": "Kwame Mensah",
	}
	bodyBytes, _ := json.Marshal(registerBody)

	resp, err := client.Post(ts.URL+"/v1/auth/register", "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("failed to send register request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created on registration, got %d", resp.StatusCode)
	}

	var regResponse AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResponse); err != nil {
		t.Fatalf("failed to decode register response: %v", err)
	}

	if regResponse.User.Email != "kwame.mensah@example.com" {
		t.Errorf("expected user email 'kwame.mensah@example.com', got %q", regResponse.User.Email)
	}
	if regResponse.User.FullName != "Kwame Mensah" {
		t.Errorf("expected full name 'Kwame Mensah', got %q", regResponse.User.FullName)
	}
	if regResponse.Token == "" {
		t.Error("expected non-empty auth token")
	}

	// 2. Duplicate Registration -> 409 Conflict
	respDup, err := client.Post(ts.URL+"/v1/auth/register", "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("failed to send duplicate register request: %v", err)
	}
	defer respDup.Body.Close()

	if respDup.StatusCode != http.StatusConflict {
		t.Errorf("expected 409 Conflict for duplicate email, got %d", respDup.StatusCode)
	}

	// 3. Weak Password -> 400 Bad Request
	weakBody, _ := json.Marshal(map[string]string{
		"email":    "other@example.com",
		"password": "weak",
	})
	respWeak, err := client.Post(ts.URL+"/v1/auth/register", "application/json", bytes.NewReader(weakBody))
	if err != nil {
		t.Fatalf("failed to send weak password request: %v", err)
	}
	defer respWeak.Body.Close()

	if respWeak.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for weak password, got %d", respWeak.StatusCode)
	}

	// 4. Invalid Email -> 400 Bad Request
	invalidEmailBody, _ := json.Marshal(map[string]string{
		"email":    "not-an-email",
		"password": "StrongP@ssw0rd1",
	})
	respEmail, err := client.Post(ts.URL+"/v1/auth/register", "application/json", bytes.NewReader(invalidEmailBody))
	if err != nil {
		t.Fatalf("failed to send invalid email request: %v", err)
	}
	defer respEmail.Body.Close()

	if respEmail.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for invalid email, got %d", respEmail.StatusCode)
	}

	// 5. Successful Login -> 200 OK
	loginBody, _ := json.Marshal(map[string]string{
		"email":    "kwame.mensah@example.com",
		"password": "StrongP@ssw0rd1",
	})
	respLogin, err := client.Post(ts.URL+"/v1/auth/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatalf("failed to send login request: %v", err)
	}
	defer respLogin.Body.Close()

	if respLogin.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK on valid login, got %d", respLogin.StatusCode)
	}

	var loginResponse AuthResponse
	if err := json.NewDecoder(respLogin.Body).Decode(&loginResponse); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}

	if loginResponse.User.ID != regResponse.User.ID {
		t.Errorf("expected user ID %q, got %q", regResponse.User.ID, loginResponse.User.ID)
	}
	if loginResponse.Token == "" {
		t.Error("expected non-empty auth token on login")
	}

	// 6. Access protected GET /v1/auth/me with valid token -> 200 OK
	reqMe, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/auth/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+loginResponse.Token)
	respMe, err := client.Do(reqMe)
	if err != nil {
		t.Fatalf("failed to call /v1/auth/me: %v", err)
	}
	defer respMe.Body.Close()

	if respMe.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK on /v1/auth/me with valid token, got %d", respMe.StatusCode)
	}

	// 7. Access protected GET /v1/auth/me without token -> 401 Unauthorized
	reqNoAuth, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/auth/me", nil)
	respNoAuth, err := client.Do(reqNoAuth)
	if err != nil {
		t.Fatalf("failed to call /v1/auth/me without token: %v", err)
	}
	defer respNoAuth.Body.Close()

	if respNoAuth.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized on /v1/auth/me without token, got %d", respNoAuth.StatusCode)
	}

	// 8. Login with Wrong Password -> 401 Unauthorized
	wrongPassBody, _ := json.Marshal(map[string]string{
		"email":    "kwame.mensah@example.com",
		"password": "WrongPassword123!",
	})
	respWrongPass, err := client.Post(ts.URL+"/v1/auth/login", "application/json", bytes.NewReader(wrongPassBody))
	if err != nil {
		t.Fatalf("failed to send wrong password login request: %v", err)
	}
	defer respWrongPass.Body.Close()

	if respWrongPass.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for wrong password, got %d", respWrongPass.StatusCode)
	}

	// 9. Login with Non-Existent Email -> 401 Unauthorized
	nonExistentBody, _ := json.Marshal(map[string]string{
		"email":    "nobody@example.com",
		"password": "StrongP@ssw0rd1",
	})
	respNonExistent, err := client.Post(ts.URL+"/v1/auth/login", "application/json", bytes.NewReader(nonExistentBody))
	if err != nil {
		t.Fatalf("failed to send non-existent email login request: %v", err)
	}
	defer respNonExistent.Body.Close()

	if respNonExistent.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for non-existent email, got %d", respNonExistent.StatusCode)
	}
}
