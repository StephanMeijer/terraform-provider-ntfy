package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewNtfyClient_Timeout(t *testing.T) {
	client := NewNtfyClient("http://localhost", "admin", "pass")
	if client.HTTP.Timeout != 30*time.Second {
		t.Errorf("expected 30s timeout, got %v", client.HTTP.Timeout)
	}
}

func TestAdminRequest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/users" {
			t.Errorf("expected /v1/users, got %s", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		expected := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:pass"))
		if auth != expected {
			t.Errorf("expected auth %q, got %q", expected, auth)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[{"username":"alice","role":"user"}]`)
	}))
	defer server.Close()

	client := NewNtfyClient(server.URL, "admin", "pass")
	body, statusCode, err := client.AdminRequest(context.Background(), http.MethodGet, "/v1/users", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", statusCode)
	}
	if !strings.Contains(string(body), "alice") {
		t.Errorf("expected body to contain 'alice', got %s", string(body))
	}
}

func TestAdminRequest_NoBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "" {
			t.Errorf("expected no Content-Type header for GET with no body, got %q", r.Header.Get("Content-Type"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[]`)
	}))
	defer server.Close()

	client := NewNtfyClient(server.URL, "admin", "pass")
	body, statusCode, err := client.AdminRequest(context.Background(), http.MethodGet, "/v1/users", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", statusCode)
	}
	if string(body) != "[]" {
		t.Errorf("expected empty array body, got %s", string(body))
	}
}

func TestAdminRequest_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"internal"}`)
	}))
	defer server.Close()

	client := NewNtfyClient(server.URL, "admin", "pass")
	body, statusCode, err := client.AdminRequest(context.Background(), http.MethodGet, "/v1/users", nil)
	// The client returns the status code and body without erroring on non-200
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if statusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", statusCode)
	}
	if !strings.Contains(string(body), "internal") {
		t.Errorf("expected body to contain 'internal', got %s", string(body))
	}
}

func TestAdminRequest_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewNtfyClient(server.URL, "admin", "pass")
	client.HTTP.Timeout = 10 * time.Millisecond

	_, _, err := client.AdminRequest(context.Background(), http.MethodGet, "/v1/users", nil)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestUserRequest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		expected := "Basic " + base64.StdEncoding.EncodeToString([]byte("alice:alicepass"))
		if auth != expected {
			t.Errorf("expected auth %q, got %q", expected, auth)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"token":"tk_abc123","label":"test"}`)
	}))
	defer server.Close()

	client := NewNtfyClient(server.URL, "admin", "adminpass")
	body, statusCode, err := client.UserRequest(context.Background(), http.MethodPost, "/v1/account/token", nil, "alice", "alicepass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", statusCode)
	}
	if !strings.Contains(string(body), "tk_abc123") {
		t.Errorf("expected body to contain token, got %s", string(body))
	}
}

func TestUserRequest_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":"unauthorized"}`)
	}))
	defer server.Close()

	client := NewNtfyClient(server.URL, "admin", "adminpass")
	_, statusCode, err := client.UserRequest(context.Background(), http.MethodPost, "/v1/account/token", nil, "alice", "wrongpass")
	// The client returns the status code without erroring on non-200
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if statusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", statusCode)
	}
}

func TestTokenRequest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer mytoken123" {
			t.Errorf("expected Bearer auth, got %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"username":"alice","role":"user"}`)
	}))
	defer server.Close()

	client := NewNtfyClient(server.URL, "admin", "adminpass")
	body, statusCode, err := client.TokenRequest(context.Background(), http.MethodGet, "/v1/account", "mytoken123", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", statusCode)
	}
	if !strings.Contains(string(body), "alice") {
		t.Errorf("expected body to contain 'alice', got %s", string(body))
	}
}

func TestTokenRequest_CustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		customHeader := r.Header.Get("X-Custom")
		if customHeader != "value" {
			t.Errorf("expected X-Custom header 'value', got %q", customHeader)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer server.Close()

	client := NewNtfyClient(server.URL, "admin", "adminpass")
	headers := map[string]string{"X-Custom": "value"}
	_, statusCode, err := client.TokenRequest(context.Background(), http.MethodGet, "/v1/account", "mytoken123", headers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", statusCode)
	}
}

func TestTokenRequest_InvalidToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":"unauthorized"}`)
	}))
	defer server.Close()

	client := NewNtfyClient(server.URL, "admin", "adminpass")
	_, statusCode, err := client.TokenRequest(context.Background(), http.MethodGet, "/v1/account", "badtoken", nil)
	// The client returns the status code without erroring on non-200
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if statusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", statusCode)
	}
}
