package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"namedot/internal/config"
	dbm "namedot/internal/db"
)

// Aliases for cleaner code
type (
	Zone           = dbm.Zone
	RRSet          = dbm.RRSet
	RData          = dbm.RData
	Template       = dbm.Template
	TemplateRecord = dbm.TemplateRecord
)

func setupTestServer(t *testing.T, cfg *config.Config) (*Server, *gin.Engine) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}

	// Auto-migrate tables
	if err := db.AutoMigrate(
		&Zone{},
		&RRSet{},
		&RData{},
		&Template{},
		&TemplateRecord{},
	); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	mockDNSServer := &mockDNSServer{}
	server := NewServer(cfg, db, mockDNSServer)

	return server, server.r
}

type mockDNSServer struct{}

func (m *mockDNSServer) InvalidateZoneCache() {}

func TestAuthMiddleware(t *testing.T) {
	// Generate bcrypt hash for testing
	hashedToken, err := bcrypt.GenerateFromPassword([]byte("test-token-hash"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate bcrypt hash: %v", err)
	}

	tests := []struct {
		name           string
		apiToken       string
		apiTokenHash   string
		authHeader     string
		expectedStatus int
		description    string
	}{
		{
			name:           "valid hashed token",
			apiToken:       "",
			apiTokenHash:   string(hashedToken),
			authHeader:     "Bearer test-token-hash",
			expectedStatus: http.StatusOK,
			description:    "Should authenticate with valid bcrypt hashed token",
		},
		{
			name:           "invalid hashed token",
			apiToken:       "",
			apiTokenHash:   string(hashedToken),
			authHeader:     "Bearer wrong-token",
			expectedStatus: http.StatusUnauthorized,
			description:    "Should reject invalid token when hash is configured",
		},
		{
			name:           "valid plain token",
			apiToken:       "plain-token-123",
			apiTokenHash:   "",
			authHeader:     "Bearer plain-token-123",
			expectedStatus: http.StatusOK,
			description:    "Should authenticate with valid plain text token (deprecated)",
		},
		{
			name:           "invalid plain token",
			apiToken:       "plain-token-123",
			apiTokenHash:   "",
			authHeader:     "Bearer wrong-token",
			expectedStatus: http.StatusUnauthorized,
			description:    "Should reject invalid plain text token",
		},
		{
			name:           "missing token when hash configured",
			apiToken:       "",
			apiTokenHash:   string(hashedToken),
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			description:    "Should reject request without token when hash is configured",
		},
		{
			name:           "missing token when plain configured",
			apiToken:       "plain-token-123",
			apiTokenHash:   "",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			description:    "Should reject request without token when plain token is configured",
		},
		{
			name:           "no authentication configured - allows all",
			apiToken:       "",
			apiTokenHash:   "",
			authHeader:     "",
			expectedStatus: http.StatusOK,
			description:    "Should allow request when no authentication is configured (current behavior)",
		},
		{
			name:           "no authentication configured with token",
			apiToken:       "",
			apiTokenHash:   "",
			authHeader:     "Bearer any-token",
			expectedStatus: http.StatusOK,
			description:    "Should allow request with any token when no authentication is configured",
		},
		{
			name:           "token without Bearer prefix - edge case",
			apiToken:       "plain-token-123",
			apiTokenHash:   "",
			authHeader:     "different-value",
			expectedStatus: http.StatusUnauthorized,
			description:    "Should reject when token doesn't match (even without Bearer prefix)",
		},
		{
			name:           "token without Bearer prefix matches configured token",
			apiToken:       "plain-token-123",
			apiTokenHash:   "",
			authHeader:     "plain-token-123",
			expectedStatus: http.StatusOK,
			description:    "Edge case: Token without 'Bearer ' prefix passes if it matches configured token",
		},
		{
			name:           "empty bearer token",
			apiToken:       "plain-token-123",
			apiTokenHash:   "",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusUnauthorized,
			description:    "Should reject empty Bearer token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				APIToken:     tt.apiToken,
				APITokenHash: tt.apiTokenHash,
			}

			_, router := setupTestServer(t, cfg)

			// Test endpoint that should be protected
			req := httptest.NewRequest("GET", "/zones", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s\nExpected status %d, got %d\nResponse: %s",
					tt.description, tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestAuthMiddleware_SecurityRecommendation tests the security recommendation:
// When authentication is not configured, it should return an error/warning
// instead of allowing all requests (current behavior is permissive).
func TestAuthMiddleware_SecurityRecommendation(t *testing.T) {
	t.Run("recommendation: reject when no auth configured", func(t *testing.T) {
		cfg := &config.Config{
			APIToken:     "",
			APITokenHash: "",
		}

		_, router := setupTestServer(t, cfg)

		req := httptest.NewRequest("GET", "/zones", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// CURRENT BEHAVIOR: Allows all (status 200)
		if w.Code != http.StatusOK {
			t.Errorf("Current behavior check failed: expected %d, got %d", http.StatusOK, w.Code)
		}

		// RECOMMENDED BEHAVIOR: Should reject (status 401 or 500)
		// This test documents the security concern that when no authentication
		// is configured, the API should either:
		// 1. Require explicit configuration (fail to start)
		// 2. Log a warning and require authentication anyway
		// 3. Have a config flag like "allow_unauthenticated: true"
		//
		// Current implementation allows all requests when auth is not configured,
		// which may be a security risk in production environments.
		t.Log("SECURITY RECOMMENDATION: When api_token and api_token_hash are both empty,")
		t.Log("the server should either:")
		t.Log("  1. Fail to start with an error requiring authentication configuration")
		t.Log("  2. Reject all API requests with 401 Unauthorized")
		t.Log("  3. Require explicit 'allow_unauthenticated: true' config option")
		t.Log("Current behavior: Allows all requests (permissive default)")
	})
}

// TestAuthMiddleware_BothTokensConfigured tests error handling when both
// api_token and api_token_hash are configured (should be prevented in config validation)
func TestAuthMiddleware_BothTokensConfigured(t *testing.T) {
	hashedToken, _ := bcrypt.GenerateFromPassword([]byte("test-token"), bcrypt.DefaultCost)

	cfg := &config.Config{
		APIToken:     "plain-token",
		APITokenHash: string(hashedToken),
	}

	// Note: This scenario should be prevented by config.Validate()
	// Testing the middleware behavior if it somehow gets through

	_, router := setupTestServer(t, cfg)

	t.Run("prefers hash over plain when both configured", func(t *testing.T) {
		// With correct hash token
		req := httptest.NewRequest("GET", "/zones", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected hash to be checked first and succeed, got status %d", w.Code)
		}

		// With plain token only (should fail because hash is checked first and doesn't fallback)
		req = httptest.NewRequest("GET", "/zones", nil)
		req.Header.Set("Authorization", "Bearer plain-token")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Current behavior: Hash is checked first, if it fails, authentication fails
		// There is NO fallback to plain token when hash is configured
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected no fallback to plain token (should be 401), got status %d", w.Code)
		}
	})
}

// TestAuthMiddleware_CaseInsensitiveBearer tests Bearer prefix handling
func TestAuthMiddleware_CaseInsensitiveBearer(t *testing.T) {
	cfg := &config.Config{
		APIToken: "test-token",
	}

	_, router := setupTestServer(t, cfg)

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{"Bearer with capital B", "Bearer test-token", http.StatusOK},
		{"bearer lowercase", "bearer test-token", http.StatusUnauthorized}, // Case sensitive!
		{"BEARER uppercase", "BEARER test-token", http.StatusUnauthorized}, // Case sensitive!
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/zones", nil)
			req.Header.Set("Authorization", tt.authHeader)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}
