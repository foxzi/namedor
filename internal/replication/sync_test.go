package replication

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"namedot/internal/config"
	dbm "namedot/internal/db"
)

func setupTestClient(t testing.TB, masterURL string) (*SyncClient, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	if err := dbm.AutoMigrate(db); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	cfg := &config.Config{
		RESTListen: "localhost:8080",
		Replication: config.ReplicationConfig{
			Mode:            "slave",
			MasterURL:       masterURL,
			APIToken:        "test-token",
			SyncIntervalSec: 60,
		},
	}

	client := NewSyncClient(cfg, db)
	return client, db
}

func TestNewSyncClient(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	cfg := &config.Config{
		Replication: config.ReplicationConfig{
			MasterURL: "http://master:8080",
		},
	}

	client := NewSyncClient(cfg, db)
	if client == nil {
		t.Fatal("Expected client, got nil")
	}
	if client.cfg != cfg {
		t.Error("Config not properly set")
	}
	if client.db != db {
		t.Error("Database not properly set")
	}
	if client.client == nil {
		t.Error("HTTP client not initialized")
	}
}

func TestFetchFromMaster_Success(t *testing.T) {
	// Create mock master server
	master := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify endpoint
		if r.URL.Path != "/sync/export" {
			t.Errorf("Expected path /sync/export, got %s", r.URL.Path)
		}

		// Verify authentication
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("Expected Bearer test-token, got %s", auth)
		}

		// Return mock data
		data := SyncData{
			Zones: []dbm.Zone{
				{Name: "test.com"},
			},
			Templates: []dbm.Template{
				{Name: "template1", Description: "Test template"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}))
	defer master.Close()

	client, _ := setupTestClient(t, master.URL)

	ctx := context.Background()
	data, err := client.FetchFromMaster(ctx)
	if err != nil {
		t.Fatalf("FetchFromMaster failed: %v", err)
	}

	if len(data.Zones) != 1 {
		t.Errorf("Expected 1 zone, got %d", len(data.Zones))
	}
	if data.Zones[0].Name != "test.com" {
		t.Errorf("Expected zone name 'test.com', got '%s'", data.Zones[0].Name)
	}

	if len(data.Templates) != 1 {
		t.Errorf("Expected 1 template, got %d", len(data.Templates))
	}
	if data.Templates[0].Name != "template1" {
		t.Errorf("Expected template name 'template1', got '%s'", data.Templates[0].Name)
	}
}

func TestFetchFromMaster_WithoutAuth(t *testing.T) {
	// Create mock master server
	master := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify no authentication header
		auth := r.Header.Get("Authorization")
		if auth != "" {
			t.Errorf("Expected no auth header, got %s", auth)
		}

		data := SyncData{}
		json.NewEncoder(w).Encode(data)
	}))
	defer master.Close()

	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	cfg := &config.Config{
		RESTListen: "localhost:8080",
		Replication: config.ReplicationConfig{
			MasterURL: master.URL,
			// No APIToken set
		},
	}

	client := NewSyncClient(cfg, db)
	ctx := context.Background()
	_, err := client.FetchFromMaster(ctx)
	if err != nil {
		t.Fatalf("FetchFromMaster failed: %v", err)
	}
}

func TestFetchFromMaster_ServerError(t *testing.T) {
	// Create mock master server that returns error
	master := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer master.Close()

	client, _ := setupTestClient(t, master.URL)

	ctx := context.Background()
	_, err := client.FetchFromMaster(ctx)
	if err == nil {
		t.Error("Expected error for server error, got nil")
	}
	if err != nil && err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

func TestFetchFromMaster_Unauthorized(t *testing.T) {
	// Create mock master server that requires auth
	master := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	}))
	defer master.Close()

	client, _ := setupTestClient(t, master.URL)

	ctx := context.Background()
	_, err := client.FetchFromMaster(ctx)
	if err == nil {
		t.Error("Expected error for unauthorized, got nil")
	}
}

func TestFetchFromMaster_InvalidJSON(t *testing.T) {
	// Create mock master server that returns invalid JSON
	master := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer master.Close()

	client, _ := setupTestClient(t, master.URL)

	ctx := context.Background()
	_, err := client.FetchFromMaster(ctx)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestFetchFromMaster_ContextCancellation(t *testing.T) {
	// Create mock master server with delay
	master := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		data := SyncData{}
		json.NewEncoder(w).Encode(data)
	}))
	defer master.Close()

	client, _ := setupTestClient(t, master.URL)

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.FetchFromMaster(ctx)
	if err == nil {
		t.Error("Expected error for cancelled context, got nil")
	}
}

func TestFetchFromMaster_EmptyData(t *testing.T) {
	// Create mock master server with empty data
	master := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := SyncData{
			Zones:     []dbm.Zone{},
			Templates: []dbm.Template{},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer master.Close()

	client, _ := setupTestClient(t, master.URL)

	ctx := context.Background()
	data, err := client.FetchFromMaster(ctx)
	if err != nil {
		t.Fatalf("FetchFromMaster failed: %v", err)
	}

	if len(data.Zones) != 0 {
		t.Errorf("Expected 0 zones, got %d", len(data.Zones))
	}
	if len(data.Templates) != 0 {
		t.Errorf("Expected 0 templates, got %d", len(data.Templates))
	}
}

func TestFetchFromMaster_LargeDataset(t *testing.T) {
	// Create mock master server with large dataset
	master := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zones := make([]dbm.Zone, 100)
		for i := 0; i < 100; i++ {
			zones[i] = dbm.Zone{Name: "zone" + string(rune(i)) + ".com"}
		}

		templates := make([]dbm.Template, 50)
		for i := 0; i < 50; i++ {
			templates[i] = dbm.Template{Name: "template" + string(rune(i))}
		}

		data := SyncData{
			Zones:     zones,
			Templates: templates,
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer master.Close()

	client, _ := setupTestClient(t, master.URL)

	ctx := context.Background()
	data, err := client.FetchFromMaster(ctx)
	if err != nil {
		t.Fatalf("FetchFromMaster failed: %v", err)
	}

	if len(data.Zones) != 100 {
		t.Errorf("Expected 100 zones, got %d", len(data.Zones))
	}
	if len(data.Templates) != 50 {
		t.Errorf("Expected 50 templates, got %d", len(data.Templates))
	}
}

func TestSyncData_Marshaling(t *testing.T) {
	// Test that SyncData can be marshaled and unmarshaled
	original := SyncData{
		Zones: []dbm.Zone{
			{Name: "test.com"},
			{Name: "example.org"},
		},
		Templates: []dbm.Template{
			{Name: "template1", Description: "First"},
			{Name: "template2", Description: "Second"},
		},
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal
	var decoded SyncData
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify
	if len(decoded.Zones) != len(original.Zones) {
		t.Errorf("Zone count mismatch: expected %d, got %d", len(original.Zones), len(decoded.Zones))
	}
	if len(decoded.Templates) != len(original.Templates) {
		t.Errorf("Template count mismatch: expected %d, got %d", len(original.Templates), len(decoded.Templates))
	}
}

func TestFetchFromMaster_FallbackToken(t *testing.T) {
	// Test that APIToken is used when Replication.APIToken is empty
	master := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer fallback-token" {
			t.Errorf("Expected Bearer fallback-token, got %s", auth)
		}
		data := SyncData{}
		json.NewEncoder(w).Encode(data)
	}))
	defer master.Close()

	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	cfg := &config.Config{
		APIToken:   "fallback-token",
		RESTListen: "localhost:8080",
		Replication: config.ReplicationConfig{
			MasterURL: master.URL,
			// Replication.APIToken is empty, should use cfg.APIToken
		},
	}

	client := NewSyncClient(cfg, db)
	ctx := context.Background()
	_, err := client.FetchFromMaster(ctx)
	if err != nil {
		t.Fatalf("FetchFromMaster failed: %v", err)
	}
}

func TestFetchFromMaster_Timeout(t *testing.T) {
	// Create mock master server that hangs
	master := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer master.Close()

	client, _ := setupTestClient(t, master.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.FetchFromMaster(ctx)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

func BenchmarkFetchFromMaster(b *testing.B) {
	master := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := SyncData{
			Zones: []dbm.Zone{
				{Name: "test1.com"},
				{Name: "test2.com"},
			},
			Templates: []dbm.Template{
				{Name: "template1"},
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer master.Close()

	client, _ := setupTestClient(b, master.URL)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.FetchFromMaster(ctx)
	}
}
