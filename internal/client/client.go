// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// Client is a Komodo API client.
type Client struct {
	endpoint   string
	username   string
	password   string
	httpClient *http.Client

	// JWT token management
	mu    sync.RWMutex
	token string
}

// NewClient creates a new Komodo API client.
func NewClient(endpoint, username, password string) *Client {
	return &Client{
		endpoint:   strings.TrimSuffix(endpoint, "/"),
		username:   username,
		password:   password,
		httpClient: http.DefaultClient,
	}
}

// LoginResponse represents the response from the login endpoint.
type LoginResponse struct {
	Data LoginResponseData `json:"data"`
	Type string            `json:"type"`
}

// LoginResponseData contains the JWT from a login response.
type LoginResponseData struct {
	JWT string `json:"jwt"`
}

// ApiKey represents a Komodo API key.
type ApiKey struct {
	Key       string `json:"key"`
	Secret    string `json:"secret"`
	UserID    string `json:"user_id"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"created_at"`
	Expires   int64  `json:"expires"`
}

// CreateApiKeyRequest represents the request to create an API key.
type CreateApiKeyRequest struct {
	Name    string `json:"name"`
	Expires int64  `json:"expires"`
}

// DeleteApiKeyRequest represents the request to delete an API key.
type DeleteApiKeyRequest struct {
	Key string `json:"key"`
}

// ListApiKeysRequest represents the request to list API keys.
type ListApiKeysRequest struct {
	Type   string                 `json:"type"`
	Params map[string]interface{} `json:"params"`
}

// login authenticates with the Komodo server and obtains a JWT token.
func (c *Client) login(ctx context.Context) error {
	loginReq := map[string]string{
		"username": c.username,
		"password": c.password,
	}

	jsonData, err := json.Marshal(loginReq)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/auth/login/LoginLocalUser", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("failed to decode login response: %w", err)
	}

	c.mu.Lock()
	c.token = loginResp.Data.JWT
	c.mu.Unlock()

	return nil
}

// getToken returns the current JWT token, logging in if necessary.
func (c *Client) getToken(ctx context.Context) (string, error) {
	c.mu.RLock()
	token := c.token
	c.mu.RUnlock()

	if token == "" {
		if err := c.login(ctx); err != nil {
			return "", err
		}
		c.mu.RLock()
		token = c.token
		c.mu.RUnlock()
	}

	return token, nil
}

// doRequest makes an authenticated HTTP request to the Komodo API.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, err
	}

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	// If we get a 401, try to re-login and retry once
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()

		c.mu.Lock()
		c.token = ""
		c.mu.Unlock()

		token, err = c.getToken(ctx)
		if err != nil {
			return nil, err
		}

		// Recreate request body if needed
		if body != nil {
			jsonData, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			reqBody = bytes.NewBuffer(jsonData)
		}

		req, err = http.NewRequestWithContext(ctx, method, c.endpoint+path, reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to execute request: %w", err)
		}
	}

	return resp, nil
}

// ListApiKeys lists all API keys for the authenticated user.
func (c *Client) ListApiKeys(ctx context.Context) ([]ApiKey, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/read/ListApiKeys", struct{}{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var keys []ApiKey
	if err := json.NewDecoder(resp.Body).Decode(&keys); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return keys, nil
}

// GetApiKey gets information about a specific API key by its key ID.
func (c *Client) GetApiKey(ctx context.Context, keyID string) (*ApiKey, error) {
	keys, err := c.ListApiKeys(ctx)
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		if key.Key == keyID {
			return &key, nil
		}
	}

	return nil, nil
}

// CreateApiKey creates a new API key.
func (c *Client) CreateApiKey(ctx context.Context, req CreateApiKeyRequest) (*ApiKey, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/auth/manage/CreateApiKey", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var key ApiKey
	if err := json.NewDecoder(resp.Body).Decode(&key); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// The API only returns key and secret, so populate the request values
	key.Name = req.Name
	key.Expires = req.Expires

	// Fetch the full key details to get user_id and created_at
	fullKey, err := c.GetApiKey(ctx, key.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch full key details: %w", err)
	}

	// If we found the full details, merge them with our create response
	if fullKey != nil {
		key.UserID = fullKey.UserID
		key.CreatedAt = fullKey.CreatedAt
		// Use the name and expires from the list if they're populated, otherwise keep request values
		if fullKey.Name != "" {
			key.Name = fullKey.Name
		}
		if fullKey.Expires != 0 {
			key.Expires = fullKey.Expires
		}
	}

	return &key, nil
}

// DeleteApiKey deletes an API key.
func (c *Client) DeleteApiKey(ctx context.Context, req DeleteApiKeyRequest) error {
	resp, err := c.doRequest(ctx, http.MethodPost, "/auth/manage/DeleteApiKey", req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
