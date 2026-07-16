// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	endpoint := "http://localhost:8080"
	username := "test-user"
	password := "test-pass"

	client := NewClient(endpoint, username, password)

	if client.endpoint != endpoint {
		t.Errorf("Expected endpoint %s, got %s", endpoint, client.endpoint)
	}

	if client.username != username {
		t.Errorf("Expected username %s, got %s", username, client.username)
	}

	if client.password != password {
		t.Errorf("Expected password %s, got %s", password, client.password)
	}

	if client.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}
}

func TestNewClient_trailingSlash(t *testing.T) {
	endpoint := "http://localhost:8080/"
	client := NewClient(endpoint, "user", "pass")

	expected := "http://localhost:8080"
	if client.endpoint != expected {
		t.Errorf("Expected endpoint %s, got %s", expected, client.endpoint)
	}
}

func TestLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/auth/login/LoginLocalUser" {
			t.Errorf("Expected path /auth/login/LoginLocalUser, got %s", r.URL.Path)
		}

		// Verify request body
		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if req["username"] != "test-user" {
			t.Errorf("Expected username 'test-user', got %s", req["username"])
		}

		if req["password"] != "test-pass" {
			t.Errorf("Expected password 'test-pass', got %s", req["password"])
		}

		// Return mock JWT
		resp := LoginResponse{
			Data: LoginResponseData{JWT: "test-jwt-token"},
			Type: "Jwt",
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-user", "test-pass")
	err := client.login(context.Background())

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if client.token != "test-jwt-token" {
		t.Errorf("Expected token 'test-jwt-token', got %s", client.token)
	}
}

func TestListAPIKeys(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// First call is login
		if callCount == 1 {
			if r.URL.Path != "/auth/login/LoginLocalUser" {
				t.Errorf("Expected first call to be login, got %s", r.URL.Path)
			}
			resp := LoginResponse{Data: LoginResponseData{JWT: "test-jwt"}, Type: "Jwt"}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Second call is list keys
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/read/ListApiKeys" {
			t.Errorf("Expected path /read/ListApiKeys, got %s", r.URL.Path)
		}

		// Check authorization header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-jwt" {
			t.Errorf("Expected Authorization header 'Bearer test-jwt', got %s", auth)
		}

		// Return mock response
		keys := []APIKey{
			{
				Key:       "K-test1",
				Secret:    "",
				UserID:    "user-123",
				Name:      "test-key-1",
				CreatedAt: 1700000000000,
				Expires:   0,
			},
			{
				Key:       "K-test2",
				Secret:    "",
				UserID:    "user-123",
				Name:      "test-key-2",
				CreatedAt: 1700000001000,
				Expires:   1800000000000,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(keys)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-user", "test-pass")
	keys, err := client.ListAPIKeys(context.Background())

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}

	if keys[0].Key != "K-test1" {
		t.Errorf("Expected key 'K-test1', got %s", keys[0].Key)
	}

	if keys[1].Name != "test-key-2" {
		t.Errorf("Expected name 'test-key-2', got %s", keys[1].Name)
	}
}

func TestGetAPIKey(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// First call is login
		if callCount == 1 {
			resp := LoginResponse{Data: LoginResponseData{JWT: "test-jwt"}, Type: "Jwt"}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Return list of keys for GetAPIKey to search through.
		keys := []APIKey{
			{
				Key:       "K-test1",
				UserID:    "user-123",
				Name:      "test-key-1",
				CreatedAt: 1700000000000,
				Expires:   0,
			},
			{
				Key:       "K-test2",
				UserID:    "user-123",
				Name:      "test-key-2",
				CreatedAt: 1700000001000,
				Expires:   0,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(keys)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-user", "test-pass")

	// Test finding existing key
	key, err := client.GetAPIKey(context.Background(), "K-test1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if key == nil {
		t.Fatal("Expected key to be returned")
	}

	if key.Key != "K-test1" {
		t.Errorf("Expected key 'K-test1', got %s", key.Key)
	}

	if key.Name != "test-key-1" {
		t.Errorf("Expected name 'test-key-1', got %s", key.Name)
	}
}

func TestGetAPIKey_notFound(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// First call is login
		if callCount == 1 {
			resp := LoginResponse{Data: LoginResponseData{JWT: "test-jwt"}, Type: "Jwt"}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Return empty list
		keys := []APIKey{}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(keys)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-user", "test-pass")

	// Test finding non-existent key
	key, err := client.GetAPIKey(context.Background(), "K-nonexistent")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if key != nil {
		t.Error("Expected nil key for non-existent key")
	}
}

func TestCreateAPIKey(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// First call is login
		if callCount == 1 {
			resp := LoginResponse{Data: LoginResponseData{JWT: "test-jwt"}, Type: "Jwt"}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Second call is CreateAPIKey.
		if callCount == 2 {
			if r.Method != http.MethodPost {
				t.Errorf("Expected POST request, got %s", r.Method)
			}
			if r.URL.Path != "/auth/manage/CreateApiKey" {
				t.Errorf("Expected path /auth/manage/CreateApiKey, got %s", r.URL.Path)
			}

			// Verify request body
			var req CreateAPIKeyRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("Failed to decode request body: %v", err)
			}

			if req.Name != "new-key" {
				t.Errorf("Expected name 'new-key', got %s", req.Name)
			}

			if req.Expires != 0 {
				t.Errorf("Expected expires 0, got %d", req.Expires)
			}

			// Return created key - API only returns key and secret
			response := map[string]string{
				"key":    "K-newkey123",
				"secret": "S-secret456",
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
			return
		}

		// Third call is ListAPIKeys (via GetAPIKey).
		if callCount == 3 {
			if r.URL.Path != "/read/ListApiKeys" {
				t.Errorf("Expected path /read/ListApiKeys, got %s", r.URL.Path)
			}

			// Return the key in the list
			keys := []APIKey{
				{
					Key:       "K-newkey123",
					Secret:    "",
					UserID:    "user-123",
					Name:      "new-key",
					CreatedAt: 1700000000000,
					Expires:   0,
				},
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(keys)
			return
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-user", "test-pass")
	key, err := client.CreateAPIKey(context.Background(), CreateAPIKeyRequest{
		Name:    "new-key",
		Expires: 0,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if key.Key != "K-newkey123" {
		t.Errorf("Expected key 'K-newkey123', got %s", key.Key)
	}

	if key.Secret != "S-secret456" {
		t.Errorf("Expected secret 'S-secret456', got %s", key.Secret)
	}

	if key.Name != "new-key" {
		t.Errorf("Expected name 'new-key', got %s", key.Name)
	}

	if key.UserID != "user-123" {
		t.Errorf("Expected user_id 'user-123', got %s", key.UserID)
	}

	if key.CreatedAt != 1700000000000 {
		t.Errorf("Expected created_at 1700000000000, got %d", key.CreatedAt)
	}
}

func TestDeleteAPIKey(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// First call is login
		if callCount == 1 {
			resp := LoginResponse{Data: LoginResponseData{JWT: "test-jwt"}, Type: "Jwt"}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/auth/manage/DeleteApiKey" {
			t.Errorf("Expected path /auth/manage/DeleteApiKey, got %s", r.URL.Path)
		}

		// Verify request body
		var req DeleteAPIKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if req.Key != "K-test123" {
			t.Errorf("Expected key 'K-test123', got %s", req.Key)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-user", "test-pass")
	err := client.DeleteAPIKey(context.Background(), DeleteAPIKeyRequest{
		Key: "K-test123",
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestClient_tokenRefresh(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// Handle login requests
		if r.URL.Path == "/auth/login/LoginLocalUser" {
			resp := LoginResponse{Data: LoginResponseData{JWT: "test-jwt"}, Type: "Jwt"}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// First API call returns 401 to trigger re-login
		if callCount == 2 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Subsequent calls succeed
		keys := []APIKey{}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(keys)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-user", "test-pass")
	_, err := client.ListAPIKeys(context.Background())

	if err != nil {
		t.Fatalf("Expected no error after token refresh, got %v", err)
	}
}

func TestClient_errorHandling(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// Handle login
		if r.URL.Path == "/auth/login/LoginLocalUser" {
			resp := LoginResponse{Data: LoginResponseData{JWT: "test-jwt"}, Type: "Jwt"}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-user", "test-pass")

	// Test ListAPIKeys error.
	_, err := client.ListAPIKeys(context.Background())
	if err == nil {
		t.Error("Expected error for 500 response")
	}

	// Test CreateAPIKey error.
	_, err = client.CreateAPIKey(context.Background(), CreateAPIKeyRequest{Name: "test"})
	if err == nil {
		t.Error("Expected error for 500 response")
	}

	// Test DeleteAPIKey error.
	err = client.DeleteAPIKey(context.Background(), DeleteAPIKeyRequest{Key: "test"})
	if err == nil {
		t.Error("Expected error for 500 response")
	}
}

func TestClient_loginError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("Invalid credentials"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "bad-user", "bad-pass")
	_, err := client.ListAPIKeys(context.Background())

	if err == nil {
		t.Error("Expected error for failed login")
	}
}
