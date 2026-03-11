package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type NtfyClient struct {
	URL      string
	Username string
	Password string
	HTTP     *http.Client
}

// API request/response types

type apiUserAddOrUpdateRequest struct {
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	Tier     string `json:"tier,omitempty"`
}

type apiUserDeleteRequest struct {
	Username string `json:"username"`
}

type apiUserResponse struct {
	Username string                  `json:"username"`
	Role     string                  `json:"role"`
	Grants   []*apiUserGrantResponse `json:"grants,omitempty"`
}

type apiUserGrantResponse struct {
	Topic      string `json:"topic"`
	Permission string `json:"permission"`
}

type apiAccessAllowRequest struct {
	Username   string `json:"username"`
	Topic      string `json:"topic"`
	Permission string `json:"permission"`
}

type apiAccessResetRequest struct {
	Username string `json:"username"`
	Topic    string `json:"topic"`
}

type apiAccountTokenIssueRequest struct {
	Label   *string `json:"label"`
	Expires *int64  `json:"expires"`
}

type apiAccountTokenResponse struct {
	Token   string `json:"token"`
	Label   string `json:"label,omitempty"`
	Expires int64  `json:"expires,omitempty"`
}

func NewNtfyClient(url, username, password string) *NtfyClient {
	return &NtfyClient{
		URL:      url,
		Username: username,
		Password: password,
		HTTP:     &http.Client{Timeout: 30 * time.Second},
	}
}

// AdminRequest performs a request with admin Basic Auth credentials.
// Used for /v1/users, /v1/users/access endpoints.
func (c *NtfyClient) AdminRequest(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.URL+path, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.Username, c.Password)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request to %s %s failed: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// UserRequest performs a request with user-specific Basic Auth credentials.
// Used for POST /v1/account/token.
func (c *NtfyClient) UserRequest(ctx context.Context, method, path string, body interface{}, username, password string) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.URL+path, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(username, password)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request to %s %s failed: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// TokenRequest performs a request with Bearer token auth.
// Used for GET /v1/account and token delete operations.
func (c *NtfyClient) TokenRequest(ctx context.Context, method, path string, token string, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.URL+path, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request to %s %s failed: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBody, resp.StatusCode, nil
}
