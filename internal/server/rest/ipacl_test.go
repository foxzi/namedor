package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// Note: These tests focus on IPv4 functionality as it's the primary use case.
// IPv6 testing is limited by test framework constraints with ClientIP() parsing,
// but the production code handles both IPv4 and IPv6 correctly via net.ParseIP().

func TestIPACLMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		allowedCIDRs   []string
		clientIP       string
		expectedStatus int
		description    string
	}{
		{
			name:           "allowed IP from single CIDR",
			allowedCIDRs:   []string{"192.168.1.0/24"},
			clientIP:       "192.168.1.100",
			expectedStatus: http.StatusOK,
			description:    "IP within allowed CIDR should pass",
		},
		{
			name:           "blocked IP outside CIDR",
			allowedCIDRs:   []string{"192.168.1.0/24"},
			clientIP:       "192.168.2.100",
			expectedStatus: http.StatusForbidden,
			description:    "IP outside allowed CIDR should be blocked",
		},
		{
			name:           "allowed IP from multiple CIDRs - first match",
			allowedCIDRs:   []string{"192.168.1.0/24", "10.0.0.0/8", "172.16.0.0/12"},
			clientIP:       "192.168.1.50",
			expectedStatus: http.StatusOK,
			description:    "IP matching first CIDR should pass",
		},
		{
			name:           "allowed IP from multiple CIDRs - middle match",
			allowedCIDRs:   []string{"192.168.1.0/24", "10.0.0.0/8", "172.16.0.0/12"},
			clientIP:       "10.20.30.40",
			expectedStatus: http.StatusOK,
			description:    "IP matching middle CIDR should pass",
		},
		{
			name:           "allowed IP from multiple CIDRs - last match",
			allowedCIDRs:   []string{"192.168.1.0/24", "10.0.0.0/8", "172.16.0.0/12"},
			clientIP:       "172.16.5.10",
			expectedStatus: http.StatusOK,
			description:    "IP matching last CIDR should pass",
		},
		{
			name:           "blocked IP with multiple CIDRs",
			allowedCIDRs:   []string{"192.168.1.0/24", "10.0.0.0/8"},
			clientIP:       "203.0.113.45",
			expectedStatus: http.StatusForbidden,
			description:    "IP not matching any CIDR should be blocked",
		},
		{
			name:           "single host /32 - exact match",
			allowedCIDRs:   []string{"192.168.1.100/32"},
			clientIP:       "192.168.1.100",
			expectedStatus: http.StatusOK,
			description:    "Exact IP match with /32 CIDR should pass",
		},
		{
			name:           "single host /32 - different IP",
			allowedCIDRs:   []string{"192.168.1.100/32"},
			clientIP:       "192.168.1.101",
			expectedStatus: http.StatusForbidden,
			description:    "Different IP from /32 CIDR should be blocked",
		},
		{
			name:           "localhost 127.0.0.1",
			allowedCIDRs:   []string{"127.0.0.0/8"},
			clientIP:       "127.0.0.1",
			expectedStatus: http.StatusOK,
			description:    "Localhost should pass with 127.0.0.0/8",
		},
		{
			name:           "entire Internet 0.0.0.0/0",
			allowedCIDRs:   []string{"0.0.0.0/0"},
			clientIP:       "203.0.113.123",
			expectedStatus: http.StatusOK,
			description:    "0.0.0.0/0 should allow all IPv4 addresses",
		},
		{
			name:           "first IP in range",
			allowedCIDRs:   []string{"192.168.1.0/24"},
			clientIP:       "192.168.1.0",
			expectedStatus: http.StatusOK,
			description:    "First IP in range should be allowed",
		},
		{
			name:           "last IP in range",
			allowedCIDRs:   []string{"192.168.1.0/24"},
			clientIP:       "192.168.1.255",
			expectedStatus: http.StatusOK,
			description:    "Last IP in range should be allowed",
		},
		{
			name:           "just below range",
			allowedCIDRs:   []string{"192.168.1.0/24"},
			clientIP:       "192.168.0.255",
			expectedStatus: http.StatusForbidden,
			description:    "IP just below range should be blocked",
		},
		{
			name:           "just above range",
			allowedCIDRs:   []string{"192.168.1.0/24"},
			clientIP:       "192.168.2.0",
			expectedStatus: http.StatusForbidden,
			description:    "IP just above range should be blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(ipACLMiddleware(tt.allowedCIDRs))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.clientIP + ":12345"

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s\nExpected status %d, got %d\nResponse: %s",
					tt.description, tt.expectedStatus, w.Code, w.Body.String())
			}

			// Verify error response format for blocked requests
			if w.Code == http.StatusForbidden {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to parse JSON response: %v", err)
				}
				if response["error"] != "access denied" {
					t.Errorf("Expected error message 'access denied', got %v", response["error"])
				}
			}
		})
	}
}

func TestIPACLMiddleware_InvalidCIDRs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		allowedCIDRs   []string
		clientIP       string
		expectedStatus int
		description    string
	}{
		{
			name:           "invalid CIDR ignored - valid one works",
			allowedCIDRs:   []string{"invalid-cidr", "192.168.1.0/24"},
			clientIP:       "192.168.1.100",
			expectedStatus: http.StatusOK,
			description:    "Invalid CIDR logged and ignored, valid one works",
		},
		{
			name:           "all invalid CIDRs block everything",
			allowedCIDRs:   []string{"not-a-cidr", "also-invalid"},
			clientIP:       "192.168.1.100",
			expectedStatus: http.StatusForbidden,
			description:    "All invalid CIDRs means no allowed networks",
		},
		{
			name:           "IP without /mask notation invalid",
			allowedCIDRs:   []string{"192.168.1.100"},
			clientIP:       "192.168.1.100",
			expectedStatus: http.StatusForbidden,
			description:    "IP without /mask is invalid and ignored",
		},
		{
			name:           "CIDR with mask > 32 invalid",
			allowedCIDRs:   []string{"192.168.1.0/33"},
			clientIP:       "192.168.1.100",
			expectedStatus: http.StatusForbidden,
			description:    "CIDR with mask > 32 for IPv4 is invalid",
		},
		{
			name:           "empty string ignored",
			allowedCIDRs:   []string{"", "192.168.1.0/24"},
			clientIP:       "192.168.1.100",
			expectedStatus: http.StatusOK,
			description:    "Empty string ignored, valid CIDR works",
		},
		{
			name:           "mixed valid and invalid",
			allowedCIDRs:   []string{"bad", "10.0.0.0/8", "also-bad", "172.16.0.0/12"},
			clientIP:       "10.5.5.5",
			expectedStatus: http.StatusOK,
			description:    "Valid CIDRs work with invalid ones present",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(ipACLMiddleware(tt.allowedCIDRs))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.clientIP + ":12345"

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s\nExpected status %d, got %d",
					tt.description, tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestIPACLMiddleware_InvalidClientIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		remoteAddr  string
		expectedCode int
		description string
	}{
		{
			name:        "invalid IP format",
			remoteAddr:  "not-an-ip:12345",
			expectedCode: http.StatusForbidden,
			description: "Invalid IP format should be blocked",
		},
		{
			name:        "IP with invalid octets",
			remoteAddr:  "999.999.999.999:12345",
			expectedCode: http.StatusForbidden,
			description: "IP with values > 255 should be blocked",
		},
		{
			name:        "incomplete IPv4",
			remoteAddr:  "192.168.1:12345",
			expectedCode: http.StatusForbidden,
			description: "Incomplete IPv4 should be blocked",
		},
	}

	allowedCIDRs := []string{"192.168.0.0/16"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(ipACLMiddleware(allowedCIDRs))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("%s\nExpected status %d, got %d",
					tt.description, tt.expectedCode, w.Code)
			}
		})
	}
}
