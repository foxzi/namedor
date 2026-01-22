package replication

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "strings"
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

// ApplyData applies synced data to local database directly
func (s *SyncClient) ApplyData(data *SyncData) error {
    return s.db.Transaction(func(tx *gorm.DB) error {
        // Import zones
        for _, zone := range data.Zones {
            // Normalize zone name
            zoneName := normalizeFQDN(zone.Name)

            var existingZone dbm.Zone
            err := tx.Where("name = ?", zoneName).First(&existingZone).Error

            if err == gorm.ErrRecordNotFound {
                // Create new zone
                newZone := dbm.Zone{Name: zoneName}
                if err := tx.Create(&newZone).Error; err != nil {
                    return fmt.Errorf("create zone %s: %w", zone.Name, err)
                }
                existingZone = newZone
            } else if err != nil {
                return fmt.Errorf("check zone %s: %w", zone.Name, err)
            }

            // Delete old rrsets and their records for this zone
            var rrsetIDs []uint
            if err := tx.Model(&dbm.RRSet{}).Where("zone_id = ?", existingZone.ID).Pluck("id", &rrsetIDs).Error; err != nil {
                return fmt.Errorf("get rrset ids for zone %s: %w", zone.Name, err)
            }
            if len(rrsetIDs) > 0 {
                if err := tx.Where("rr_set_id IN ?", rrsetIDs).Delete(&dbm.RData{}).Error; err != nil {
                    return fmt.Errorf("delete old records for zone %s: %w", zone.Name, err)
                }
            }
            if err := tx.Where("zone_id = ?", existingZone.ID).Delete(&dbm.RRSet{}).Error; err != nil {
                return fmt.Errorf("delete old rrsets for zone %s: %w", zone.Name, err)
            }

            // Create new rrsets
            for _, rrset := range zone.RRSets {
                normalizedName := normalizeRRSetName(rrset.Name, zoneName)
                newRRSet := dbm.RRSet{
                    ZoneID:  existingZone.ID,
                    Name:    normalizedName,
                    Type:    strings.ToUpper(rrset.Type),
                    TTL:     rrset.TTL,
                    Records: rrset.Records,
                }
                // Clear IDs to avoid conflicts
                for i := range newRRSet.Records {
                    newRRSet.Records[i].ID = 0
                }
                if err := tx.Create(&newRRSet).Error; err != nil {
                    return fmt.Errorf("create rrset %s/%s: %w", zone.Name, rrset.Name, err)
                }
            }
        }

        // Import templates
        for _, tmpl := range data.Templates {
            var existingTmpl dbm.Template
            err := tx.Where("name = ?", tmpl.Name).First(&existingTmpl).Error

            if err == gorm.ErrRecordNotFound {
                newTmpl := dbm.Template{
                    Name:        tmpl.Name,
                    Description: tmpl.Description,
                }
                if err := tx.Create(&newTmpl).Error; err != nil {
                    return fmt.Errorf("create template %s: %w", tmpl.Name, err)
                }
                existingTmpl = newTmpl
            } else if err != nil {
                return fmt.Errorf("check template %s: %w", tmpl.Name, err)
            } else {
                existingTmpl.Description = tmpl.Description
                if err := tx.Save(&existingTmpl).Error; err != nil {
                    return fmt.Errorf("update template %s: %w", tmpl.Name, err)
                }
            }

            // Delete old template records
            if err := tx.Where("template_id = ?", existingTmpl.ID).Delete(&dbm.TemplateRecord{}).Error; err != nil {
                return fmt.Errorf("delete old records for template %s: %w", tmpl.Name, err)
            }

            // Create new template records
            for _, rec := range tmpl.Records {
                newRec := dbm.TemplateRecord{
                    TemplateID: existingTmpl.ID,
                    Name:       rec.Name,
                    Type:       rec.Type,
                    TTL:        rec.TTL,
                    Data:       rec.Data,
                    Country:    rec.Country,
                    Continent:  rec.Continent,
                    ASN:        rec.ASN,
                    Subnet:     rec.Subnet,
                }
                if err := tx.Create(&newRec).Error; err != nil {
                    return fmt.Errorf("create template record for %s: %w", tmpl.Name, err)
                }
            }
        }

        return nil
    })
}

// normalizeFQDN ensures name is lowercase and ends with a dot
func normalizeFQDN(name string) string {
    n := strings.ToLower(strings.TrimSpace(name))
    if n != "" && !strings.HasSuffix(n, ".") {
        n += "."
    }
    return n
}

// normalizeRRSetName normalizes record name for zone
func normalizeRRSetName(name, zoneName string) string {
    n := strings.ToLower(strings.TrimSpace(name))
    zone := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(zoneName)), ".")
    zoneFQDN := zone + "."

    if n == "" || n == "@" {
        return zoneFQDN
    }
    if strings.HasSuffix(n, ".@") {
        n = strings.TrimSuffix(n, ".@")
    }
    if strings.HasSuffix(n, ".") {
        return n
    }
    if n == zone {
        return zoneFQDN
    }
    if strings.HasSuffix(n, "."+zone) {
        return n + "."
    }
    return n + "." + zoneFQDN
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
