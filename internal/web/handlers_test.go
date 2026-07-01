package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dereknola/FogCast/internal/config"
	"github.com/dereknola/FogCast/internal/session"
)

func TestStateEndpoint(t *testing.T) {
	server := NewServer(config.Config{StaticDir: "static"}, session.NewManager())

	request := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var state session.State
	if err := json.NewDecoder(response.Body).Decode(&state); err != nil {
		t.Fatalf("decode state: %v", err)
	}

	if state.Mask.Width != 512 || state.Mask.Height != 512 {
		t.Fatalf("expected default 512x512 mask, got %dx%d", state.Mask.Width, state.Mask.Height)
	}
}

func TestRootEndpoint(t *testing.T) {
	server := NewServer(config.Config{StaticDir: "static"}, session.NewManager())

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
}
