package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"namedot/internal/config"
	dbm "namedot/internal/db"
)

func setupZoneIOTestServer(t *testing.T) (*Server, *gorm.DB, uint) {
	t.Helper()

cfg := &config.Config{
	APIToken:         "testtoken",
	DefaultTTL:       300,
	SOA:              config.SOAConfig{AutoOnMissing: true},
}

	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	if err := dbm.AutoMigrate(gormDB); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Create a test zone
	zone := Zone{Name: "export.test"}
	if err := gormDB.Create(&zone).Error; err != nil {
		t.Fatalf("create zone: %v", err)
	}

	mockDNS := &mockDNSServer{}
	server := NewServer(cfg, gormDB, mockDNS)

	return server, gormDB, zone.ID
}

func TestExportZone_JSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupRecords   bool
		zoneID         string
		expectedStatus int
		validateResult func(*testing.T, *Zone)
		description    string
	}{
		{
			name:           "export empty zone",
			setupRecords:   false,
			zoneID:         "1",
			expectedStatus: http.StatusOK,
			validateResult: func(t *testing.T, z *Zone) {
				if z.Name != "export.test" {
					t.Errorf("Expected zone name 'export.test', got '%s'", z.Name)
				}
				if len(z.RRSets) != 0 {
					t.Errorf("Expected 0 RRSets, got %d", len(z.RRSets))
				}
			},
			description: "Should export zone with no records",
		},
		{
			name:           "export zone with records",
			setupRecords:   true,
			zoneID:         "1",
			expectedStatus: http.StatusOK,
			validateResult: func(t *testing.T, z *Zone) {
				if len(z.RRSets) == 0 {
					t.Error("Expected RRSets to be included")
				}
				// Check that records are preloaded
				for _, rr := range z.RRSets {
					if len(rr.Records) == 0 {
						t.Error("Expected Records to be preloaded in RRSets")
					}
				}
			},
			description: "Should export zone with RRSets and Records",
		},
		{
			name:           "non-existent zone",
			setupRecords:   false,
			zoneID:         "999",
			expectedStatus: http.StatusNotFound,
			description:    "Should return 404 for non-existent zone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, gormDB, zoneID := setupZoneIOTestServer(t)

			if tt.setupRecords {
				rrset := RRSet{
					ZoneID: zoneID,
					Name:   "www.export.test.",
					Type:   "A",
					TTL:    300,
					Records: []RData{
						{Data: "192.0.2.1"},
					},
				}
				if err := gormDB.Create(&rrset).Error; err != nil {
					t.Fatalf("Failed to create rrset: %v", err)
				}
			}

			requestZoneID := tt.zoneID
			if tt.zoneID == "1" {
				requestZoneID = strconv.FormatUint(uint64(zoneID), 10)
			}

			req := httptest.NewRequest("GET", "/zones/"+requestZoneID+"/export?format=json", nil)
			req.Header.Set("Authorization", "Bearer testtoken")
			w := httptest.NewRecorder()

			server.r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s\nExpected status %d, got %d", tt.description, tt.expectedStatus, w.Code)
			}

			if tt.validateResult != nil {
				var zone Zone
				if err := json.Unmarshal(w.Body.Bytes(), &zone); err != nil {
					t.Fatalf("Failed to parse JSON response: %v", err)
				}
				tt.validateResult(t, &zone)
			}
		})
	}
}

func TestExportZone_BIND(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server, gormDB, zoneID := setupZoneIOTestServer(t)

	// Create some records
	rrsets := []RRSet{
		{
			ZoneID: zoneID,
			Name:   "export.test.",
			Type:   "SOA",
			TTL:    3600,
			Records: []RData{
				{Data: "ns1.export.test. admin.export.test. 1 3600 1800 604800 86400"},
			},
		},
		{
			ZoneID: zoneID,
			Name:   "www.export.test.",
			Type:   "A",
			TTL:    300,
			Records: []RData{
				{Data: "192.0.2.1"},
			},
		},
		{
			ZoneID: zoneID,
			Name:   "export.test.",
			Type:   "MX",
			TTL:    3600,
			Records: []RData{
				{Data: "10 mail.export.test."},
			},
		},
	}

	for _, rr := range rrsets {
		if err := gormDB.Create(&rr).Error; err != nil {
			t.Fatalf("Failed to create rrset: %v", err)
		}
	}

	req := httptest.NewRequest("GET", "/zones/"+strconv.FormatUint(uint64(zoneID), 10)+"/export?format=bind", nil)
	req.Header.Set("Authorization", "Bearer testtoken")
	w := httptest.NewRecorder()

	server.r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	bindZone := w.Body.String()

	// Verify BIND format structure
	if !strings.Contains(bindZone, "$ORIGIN") {
		t.Error("Expected $ORIGIN in BIND zone file")
	}
	if !strings.Contains(bindZone, "SOA") {
		t.Error("Expected SOA record in BIND zone file")
	}
	if !strings.Contains(bindZone, "www") || !strings.Contains(bindZone, "192.0.2.1") {
		t.Error("Expected A record in BIND zone file")
	}
	if !strings.Contains(bindZone, "MX") {
		t.Error("Expected MX record in BIND zone file")
	}
}

func TestExportZone_UnsupportedFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server, _, zoneID := setupZoneIOTestServer(t)

	req := httptest.NewRequest("GET", "/zones/"+strconv.FormatUint(uint64(zoneID), 10)+"/export?format=yaml", nil)
	req.Header.Set("Authorization", "Bearer testtoken")
	w := httptest.NewRecorder()

	server.r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for unsupported format, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if response["error"] != "unsupported format" {
		t.Errorf("Expected error 'unsupported format', got '%v'", response["error"])
	}
}

func TestImportZone_JSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		mode           string
		existingData   bool
		importPayload  string
		expectedStatus int
		validateResult func(*testing.T, *gorm.DB, uint)
		description    string
	}{
		{
			name: "import into empty zone",
			mode: "upsert",
			existingData: false,
			importPayload: `{
				"name":"export.test",
				"rrsets":[
					{
						"name":"www",
						"type":"A",
						"ttl":300,
						"records":[{"data":"192.0.2.1"}]
					}
				]
			}`,
			expectedStatus: http.StatusNoContent,
			validateResult: func(t *testing.T, db *gorm.DB, zoneID uint) {
				var rrsets []RRSet
				if err := db.Preload("Records").Where("zone_id = ?", zoneID).Find(&rrsets).Error; err != nil {
					t.Fatalf("Failed to load rrsets: %v", err)
				}
				if len(rrsets) == 0 {
					t.Error("Expected imported RRSets")
				}
			},
			description: "Should import records into empty zone",
		},
		{
			name: "import replaces existing records in upsert mode",
			mode: "upsert",
			existingData: true,
			importPayload: `{
				"name":"export.test",
				"rrsets":[
					{
						"name":"new",
						"type":"A",
						"ttl":300,
						"records":[{"data":"192.0.2.2"}]
					}
				]
			}`,
			expectedStatus: http.StatusNoContent,
			validateResult: func(t *testing.T, db *gorm.DB, zoneID uint) {
				var rrsets []RRSet
				if err := db.Preload("Records").Where("zone_id = ?", zoneID).Find(&rrsets).Error; err != nil {
					t.Fatalf("Failed to load rrsets: %v", err)
				}
				// Should have the new record
				found := false
				for _, rr := range rrsets {
					if rr.Name == "new.export.test." {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected new imported record")
				}
			},
			description: "Should merge/upsert imported records",
		},
		{
			name: "import geo-aware records",
			mode: "upsert",
			existingData: false,
			importPayload: `{
				"name":"export.test",
				"rrsets":[
					{
						"name":"geo",
						"type":"A",
						"ttl":300,
						"records":[
							{"data":"192.0.2.1","country":"US"},
							{"data":"192.0.2.2","continent":"EU"}
						]
					}
				]
			}`,
			expectedStatus: http.StatusNoContent,
			validateResult: func(t *testing.T, db *gorm.DB, zoneID uint) {
				var rrsets []RRSet
				if err := db.Preload("Records").Where("zone_id = ? AND name = ?", zoneID, "geo.export.test.").Find(&rrsets).Error; err != nil {
					t.Fatalf("Failed to load rrsets: %v", err)
				}
				if len(rrsets) == 0 {
					t.Fatal("Expected geo-aware RRSet")
				}
				if len(rrsets[0].Records) != 2 {
					t.Errorf("Expected 2 records, got %d", len(rrsets[0].Records))
				}
			},
			description: "Should import geo-aware records",
		},
		{
			name: "invalid json",
			mode: "upsert",
			existingData: false,
			importPayload: `{invalid}`,
			expectedStatus: http.StatusBadRequest,
			description: "Should reject invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, gormDB, zoneID := setupZoneIOTestServer(t)

			if tt.existingData {
				rrset := RRSet{
					ZoneID: zoneID,
					Name:   "old.export.test.",
					Type:   "A",
					TTL:    300,
					Records: []RData{
						{Data: "192.0.2.99"},
					},
				}
				if err := gormDB.Create(&rrset).Error; err != nil {
					t.Fatalf("Failed to create existing rrset: %v", err)
				}
			}

			url := "/zones/" + strconv.FormatUint(uint64(zoneID), 10) + "/import?format=json&mode=" + tt.mode
			req := httptest.NewRequest("POST", url, bytes.NewBufferString(tt.importPayload))
			req.Header.Set("Authorization", "Bearer testtoken")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s\nExpected status %d, got %d\nBody: %s",
					tt.description, tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.validateResult != nil {
				tt.validateResult(t, gormDB, zoneID)
			}
		})
	}
}

func TestImportZone_BIND(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server, gormDB, zoneID := setupZoneIOTestServer(t)

	bindZone := `$ORIGIN export.test.
$TTL 3600
@   SOA ns1.export.test. admin.export.test. (
        2024010101  ; serial
        3600        ; refresh
        1800        ; retry
        604800      ; expire
        86400 )     ; minimum

@   NS  ns1.export.test.
@   NS  ns2.export.test.
www A   192.0.2.1
`

	url := "/zones/" + strconv.FormatUint(uint64(zoneID), 10) + "/import?format=bind&mode=upsert"
	req := httptest.NewRequest("POST", url, bytes.NewBufferString(bindZone))
	req.Header.Set("Authorization", "Bearer testtoken")
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	server.r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d\nBody: %s", http.StatusNoContent, w.Code, w.Body.String())
	}

	// Verify imported records
	var rrsets []RRSet
	if err := gormDB.Preload("Records").Where("zone_id = ?", zoneID).Find(&rrsets).Error; err != nil {
		t.Fatalf("Failed to load rrsets: %v", err)
	}

	if len(rrsets) == 0 {
		t.Error("Expected imported RRSets from BIND format")
	}

	// Check for specific records
	types := make(map[string]bool)
	for _, rr := range rrsets {
		types[rr.Type] = true
	}

	expectedTypes := []string{"SOA", "NS", "A"}
	for _, typ := range expectedTypes {
		if !types[typ] {
			t.Errorf("Expected %s record to be imported", typ)
		}
	}
}

func TestImportZone_UnsupportedFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server, _, zoneID := setupZoneIOTestServer(t)

	url := "/zones/" + strconv.FormatUint(uint64(zoneID), 10) + "/import?format=yaml"
	req := httptest.NewRequest("POST", url, bytes.NewBufferString("data"))
	req.Header.Set("Authorization", "Bearer testtoken")
	w := httptest.NewRecorder()

	server.r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for unsupported format, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestImportZone_NonExistentZone(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server, _, _ := setupZoneIOTestServer(t)

	url := "/zones/999/import?format=json"
	req := httptest.NewRequest("POST", url, bytes.NewBufferString(`{"name":"test","rrsets":[]}`))
	req.Header.Set("Authorization", "Bearer testtoken")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d for non-existent zone, got %d", http.StatusNotFound, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if response["error"] != "zone not found" {
		t.Errorf("Expected error 'zone not found', got '%v'", response["error"])
	}
}

func TestImportZone_RestoresNSRecords(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	if err := dbm.AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Setup server with NS auto-creation enabled
	cfg := &config.Config{
		APIToken:   "testtoken",
		DefaultTTL: 3600,
		SOA: config.SOAConfig{
			Primary:       "ns1.{zone}",
			Hostmaster:    "hostmaster.{zone}",
			AutoOnMissing: true,
		},
		NS: config.NSConfig{
			Servers:       []string{"ns1.{zone}", "ns2.{zone}"},
			AutoOnMissing: true,
			TTL:           86400,
		},
	}
	mockDNS := &mockDNSServer{}
	server := NewServer(cfg, db, mockDNS)

	r := gin.New()
	// Simple auth middleware for testing
	auth := func(c *gin.Context) {
		token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if token != cfg.APIToken {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
	r.Use(auth)
	r.POST("/zones", server.createZone)
	r.POST("/zones/:id/import", server.importZone)

	// Step 1: Create zone (NS records should be created automatically)
	createPayload := `{"name":"example.org"}`
	req := httptest.NewRequest("POST", "/zones", bytes.NewBufferString(createPayload))
	req.Header.Set("Authorization", "Bearer testtoken")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status %d for zone creation, got %d", http.StatusCreated, w.Code)
	}

	var zone Zone
	if err := json.Unmarshal(w.Body.Bytes(), &zone); err != nil {
		t.Fatalf("Failed to parse zone response: %v", err)
	}

	// Step 2: Verify NS records exist
	var nsCount int64
	db.Model(&RRSet{}).Where("zone_id = ? AND type = ?", zone.ID, "NS").Count(&nsCount)
	if nsCount != 1 {
		t.Errorf("Expected 1 NS RRSet after zone creation, got %d", nsCount)
	}

	// Step 3: Import zone data with mode=replace WITHOUT NS records
	importPayload := `{
		"name": "example.org",
		"rrsets": [
			{
				"name": "example.org.",
				"type": "A",
				"ttl": 300,
				"records": [{"data": "192.0.2.1"}]
			},
			{
				"name": "www.example.org.",
				"type": "A",
				"ttl": 300,
				"records": [{"data": "192.0.2.2"}]
			}
		]
	}`
	req2 := httptest.NewRequest("POST", "/zones/"+fmt.Sprint(zone.ID)+"/import?format=json&mode=replace", bytes.NewBufferString(importPayload))
	req2.Header.Set("Authorization", "Bearer testtoken")
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusNoContent {
		t.Fatalf("Expected status %d for import, got %d", http.StatusNoContent, w2.Code)
	}

	// Step 4: Verify NS records were restored automatically
	var nsCount2 int64
	db.Model(&RRSet{}).Where("zone_id = ? AND type = ?", zone.ID, "NS").Count(&nsCount2)
	if nsCount2 != 1 {
		t.Errorf("Expected 1 NS RRSet after import, got %d", nsCount2)
	}

	// Step 5: Verify NS records contain correct data
	var nsRRSet RRSet
	if err := db.Preload("Records").Where("zone_id = ? AND type = ?", zone.ID, "NS").First(&nsRRSet).Error; err != nil {
		t.Fatalf("Failed to load NS RRSet: %v", err)
	}

	if len(nsRRSet.Records) != 2 {
		t.Errorf("Expected 2 NS records, got %d", len(nsRRSet.Records))
	}

	expectedNS := map[string]bool{
		"ns1.example.org.": false,
		"ns2.example.org.": false,
	}
	for _, record := range nsRRSet.Records {
		if _, exists := expectedNS[record.Data]; exists {
			expectedNS[record.Data] = true
		}
	}
	for ns, found := range expectedNS {
		if !found {
			t.Errorf("Expected NS record %s not found", ns)
		}
	}

	// Step 6: Verify imported A records exist
	var aCount int64
	db.Model(&RRSet{}).Where("zone_id = ? AND type = ?", zone.ID, "A").Count(&aCount)
	if aCount != 2 {
		t.Errorf("Expected 2 A RRSets after import, got %d", aCount)
	}
}

func TestZoneIO_WithoutAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server, _, zoneID := setupZoneIOTestServer(t)
	zoneIDStr := strconv.FormatUint(uint64(zoneID), 10)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		description    string
	}{
		{
			name:           "export without auth",
			method:         "GET",
			path:           "/zones/" + zoneIDStr + "/export",
			expectedStatus: http.StatusUnauthorized,
			description:    "Should reject unauthenticated export",
		},
		{
			name:           "import without auth",
			method:         "POST",
			path:           "/zones/" + zoneIDStr + "/import",
			expectedStatus: http.StatusUnauthorized,
			description:    "Should reject unauthenticated import",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.method == "POST" {
				req = httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString("{}"))
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			// No Authorization header

			w := httptest.NewRecorder()
			server.r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s\nExpected status %d, got %d", tt.description, tt.expectedStatus, w.Code)
			}
		})
	}
}
