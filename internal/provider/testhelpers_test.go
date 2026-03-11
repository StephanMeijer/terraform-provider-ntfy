package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	testUsername   = "testuser"
	testPassword   = "testpass123"
	testTopic      = "alerts"
	testPermission = "read-write"
	testToken      = "tk_testtoken123"
)

// mockHandler maps HTTP method+path to response data
type mockHandler struct {
	responses map[string]mockResponse
}

type mockResponse struct {
	statusCode int
	body       interface{}
}

func (m *mockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.Method + " " + r.URL.Path
	resp, ok := m.responses[key]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.statusCode)
	if resp.body != nil {
		json.NewEncoder(w).Encode(resp.body) //nolint:errcheck
	}
}

// newMockNtfyServer creates a test HTTP server with configurable responses
func newMockNtfyServer(t *testing.T, responses map[string]mockResponse) *httptest.Server {
	t.Helper()
	handler := &mockHandler{responses: responses}
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// newTestNtfyClient creates a NtfyClient pointing at a mock server
func newTestNtfyClient(server *httptest.Server) *NtfyClient {
	return NewNtfyClient(server.URL, testUsername, testPassword)
}
