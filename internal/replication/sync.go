package replication

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "time"

    "gorm.io/gorm"

    "namedot/internal/config"
    dbm "namedot/internal/db"
)

// SyncData matches the structure in rest/server.go
type SyncData struct {
    Zones     []dbm.Zone     `json:"zones"`
    Templates []dbm.Template `json:"templates"`
}

// SyncClient handles replication from master to slave
type SyncClient struct {
    cfg    *config.Config
    db     *gorm.DB
    client *http.Client
}

// NewSyncClient creates a new sync client
func NewSyncClient(cfg *config.Config, db *gorm.DB) *SyncClient {
    return &SyncClient{
        cfg: cfg,
        db:  db,
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

// FetchFromMaster fetches data from master server
func (s *SyncClient) FetchFromMaster(ctx context.Context) (*SyncData, error) {
    url := s.cfg.Replication.MasterURL + "/sync/export"

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }

    // Add authentication token
    token := s.cfg.Replication.APIToken
    if token == "" {
        token = s.cfg.APIToken
    }
    if token != "" {
        req.Header.Set("Authorization", "Bearer "+token)
    }

    resp, err := s.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("master returned status %d: %s", resp.StatusCode, string(body))
    }

    var data SyncData
    if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }

    return &data, nil
}

// ApplyData applies synced data to local database
func (s *SyncClient) ApplyData(data *SyncData) error {
    // Use the same import logic as syncImport endpoint
    url := "http://" + s.cfg.RESTListen + "/sync/import"

    jsonData, err := json.Marshal(data)
    if err != nil {
        return fmt.Errorf("marshal data: %w", err)
    }

    req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
    if err != nil {
        return fmt.Errorf("create request: %w", err)
    }

    req.Header.Set("Content-Type", "application/json")
    if s.cfg.APIToken != "" {
        req.Header.Set("Authorization", "Bearer "+s.cfg.APIToken)
    }

    resp, err := s.client.Do(req)
    if err != nil {
        return fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("import failed with status %d: %s", resp.StatusCode, string(body))
    }

    return nil
}

// SyncOnce performs a single synchronization from master
func (s *SyncClient) SyncOnce(ctx context.Context) error {
    log.Println("Starting sync from master...")

    data, err := s.FetchFromMaster(ctx)
    if err != nil {
        return fmt.Errorf("fetch from master: %w", err)
    }

    log.Printf("Fetched %d zones and %d templates from master", len(data.Zones), len(data.Templates))

    if err := s.ApplyData(data); err != nil {
        return fmt.Errorf("apply data: %w", err)
    }

    log.Println("Sync completed successfully")
    return nil
}

// StartPeriodicSync starts periodic synchronization in background
func (s *SyncClient) StartPeriodicSync(ctx context.Context) {
    interval := time.Duration(s.cfg.Replication.SyncIntervalSec) * time.Second
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    log.Printf("Starting periodic sync every %v", interval)

    // Initial sync
    if err := s.SyncOnce(ctx); err != nil {
        log.Printf("Initial sync failed: %v", err)
    }

    for {
        select {
        case <-ctx.Done():
            log.Println("Stopping periodic sync")
            return
        case <-ticker.C:
            if err := s.SyncOnce(ctx); err != nil {
                log.Printf("Periodic sync failed: %v", err)
            }
        }
    }
}
