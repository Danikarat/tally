package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Danikarat/tally/internal/config"
)

func TestServer_HealthCheck(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
		Port:        8080,
	}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	srv := New(cfg, logger, nil)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", contentType)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", body["status"])
	}
	if body["environment"] != "test" {
		t.Errorf("expected environment 'test', got %q", body["environment"])
	}
}

func TestServer_NotFound(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
		Port:        8080,
	}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	srv := New(cfg, logger, nil)

	req := httptest.NewRequest(http.MethodGet, "/unknown-endpoint", nil)
	rec := httptest.NewRecorder()

	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 Not Found, got %d", rec.Code)
	}
}
