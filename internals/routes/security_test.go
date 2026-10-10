package routes_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	baseURL   = "http://localhost:8000"
	jwtSecret = "my-super-secret-key"
)

func TestSecurityAndAPIs(t *testing.T) {
	// 1. Health check
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		t.Skipf("Server not running on %s, skipping live integration test: %v", baseURL, err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 on /health, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 2. Unauthenticated admin requests should be rejected with 401
	adminEndpoints := []string{
		"/api/admin/dashboard/stats",
		"/api/admin/users",
		"/api/admin/teams",
		"/api/admin/payments",
		"/api/admin/admins",
	}

	for _, ep := range adminEndpoints {
		r, err := http.Get(baseURL + ep)
		if err != nil {
			t.Fatalf("Failed to call %s: %v", ep, err)
		}
		if r.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized on unauthenticated %s, got %d", ep, r.StatusCode)
		}
		r.Body.Close()
	}

	// 3. Admin Dev Login
	loginBody, _ := json.Marshal(map[string]string{
		"email": "transfinitte@gmail.com",
	})
	loginResp, err := http.Post(baseURL+"/api/admin/auth/dev-login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatalf("Failed to login as admin: %v", err)
	}
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("Dev login failed with status %d", loginResp.StatusCode)
	}

	var loginData struct {
		Success bool `json:"success"`
		Data    struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.NewDecoder(loginResp.Body).Decode(&loginData)
	loginResp.Body.Close()

	if loginData.Data.Token == "" {
		t.Fatalf("Did not receive JWT token from dev login")
	}
	adminToken := loginData.Data.Token

	// 4. Authenticated admin requests with Admin JWT should succeed (200 OK)
	client := &http.Client{Timeout: 30 * time.Second}
	for _, ep := range adminEndpoints {
		req, _ := http.NewRequest(http.MethodGet, baseURL+ep, nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		res, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to call %s with admin token: %v", ep, err)
		}
		if res.StatusCode != http.StatusOK {
			t.Errorf("Expected 200 on %s with admin token, got %d", ep, res.StatusCode)
		}
		res.Body.Close()
	}

	// 5. Test NORMAL USER JWT against admin endpoints!
	// Normal user tokens have sub (user uuid), email, exp, iat, but NOT type="admin" or role="admin"
	userClaims := jwt.MapClaims{
		"sub":   uuid.New().String(),
		"email": "normaluser@example.com",
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}
	normalUserToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, userClaims).SignedString([]byte(jwtSecret))
	if err != nil {
		t.Fatalf("Failed to sign normal user token: %v", err)
	}

	for _, ep := range adminEndpoints {
		req, _ := http.NewRequest(http.MethodGet, baseURL+ep, nil)
		req.Header.Set("Authorization", "Bearer "+normalUserToken)
		res, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to call %s with normal user token: %v", ep, err)
		}
		// A normal user should NOT be able to access ANY admin endpoint!
		// It should be 401 or 403.
		if res.StatusCode == http.StatusOK {
			t.Errorf("SECURITY VULNERABILITY: Normal user was able to access %s! Status: %d", ep, res.StatusCode)
		} else {
			t.Logf("Correctly blocked normal user from %s: status %d", ep, res.StatusCode)
		}
		res.Body.Close()
	}
}
