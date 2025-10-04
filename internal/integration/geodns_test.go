package integration

import (
    "bytes"
    "encoding/json"
    "net"
    "net/http"
    "os"
    "path/filepath"
    "testing"
    "time"

    "github.com/miekg/dns"

    "smaillgeodns/internal/config"
    "smaillgeodns/internal/db"
    dnssrv "smaillgeodns/internal/server/dns"
    restsrv "smaillgeodns/internal/server/rest"
)

// Test GeoDNS selection using ECS with a known US IP (8.8.8.8).
func TestGeoDNS_WithECS_USCountry(t *testing.T) {
    // Locate geoipdb directory from repo root
    // This test runs from package dir internal/integration
    geoDir := filepath.Clean(filepath.Join("..", "..", "geoipdb"))
    if st, err := os.Stat(geoDir); err != nil || !st.IsDir() {
        t.Skipf("geoipdb directory not found; skipping GeoDNS test")
    }
    // Quick check there is at least one .mmdb file
    files, _ := os.ReadDir(geoDir)
    hasMMDB := false
    for _, f := range files { if !f.IsDir() && filepath.Ext(f.Name()) == ".mmdb" { hasMMDB = true; break } }
    if !hasMMDB { t.Skip("no .mmdb files in geoipdb; skipping") }

    dnsAddr := "127.0.0.1:19054"
    restAddr := "127.0.0.1:18090"

    tmpDB := filepath.Join(t.TempDir(), "geo_integration.db")
    cfg := &config.Config{
        Listen:           dnsAddr,
        Forwarder:        "",
        EnableDNSSEC:     false,
        APIToken:         "devtoken",
        RESTListen:       restAddr,
        AutoSOAOnMissing: true,
        DefaultTTL:       60,
        DB: config.DBConfig{Driver: "sqlite", DSN: "file:" + tmpDB + "?_foreign_keys=on"},
        GeoIP: config.GeoIPConfig{Enabled: true, MMDBPath: geoDir, ReloadSec: 0, UseECS: true},
        Update: config.UpdateConfig{Enabled: false},
    }

    gormDB, err := db.Open(cfg.DB)
    if err != nil { t.Fatalf("open db: %v", err) }
    if err := db.AutoMigrate(gormDB); err != nil { t.Fatalf("migrate: %v", err) }

    dnsServer, err := dnssrv.NewServer(cfg, gormDB)
    if err != nil { t.Fatalf("dns: %v", err) }
    restServer := restsrv.NewServer(cfg, gormDB)

    go func() { _ = dnsServer.Start() }()
    go func() { _ = restServer.Start() }()

    if err := waitHTTPReady("http://"+restAddr+"/zones", 5*time.Second); err != nil {
        t.Fatalf("rest not ready: %v", err)
    }

    // Create zone
    type zoneResp struct{ ID uint `json:"id"` }
    zr := zoneResp{}
    reqBody := bytes.NewBufferString(`{"name":"geodns.test"}`)
    req, _ := http.NewRequest("POST", "http://"+restAddr+"/zones", reqBody)
    req.Header.Set("Authorization", "Bearer "+cfg.APIToken)
    req.Header.Set("Content-Type", "application/json")
    resp, err := http.DefaultClient.Do(req)
    if err != nil { t.Fatalf("create zone: %v", err) }
    if resp.StatusCode != http.StatusCreated { t.Fatalf("create zone status: %d", resp.StatusCode) }
    _ = json.NewDecoder(resp.Body).Decode(&zr)
    resp.Body.Close()

    // Add RRset prioritizing subnet match for ECS 8.8.8.8, plus country-specific and generic
    rrJSON := `{"name":"svc","type":"A","ttl":60,"records":[{"data":"198.51.100.11","country":"US"},{"data":"198.51.100.13","subnet":"8.8.8.0/24"},{"data":"198.51.100.12"}]}`
    req2, _ := http.NewRequest("POST", "http://"+restAddr+"/zones/"+itoa(zr.ID)+"/rrsets", bytes.NewBufferString(rrJSON))
    req2.Header.Set("Authorization", "Bearer "+cfg.APIToken)
    req2.Header.Set("Content-Type", "application/json")
    resp2, err := http.DefaultClient.Do(req2)
    if err != nil { t.Fatalf("create rrset: %v", err) }
    if resp2.StatusCode != http.StatusCreated { t.Fatalf("create rrset status: %d", resp2.StatusCode) }
    resp2.Body.Close()

    // DNS query with ECS=8.8.8.8/24 (US)
    c := &dns.Client{Timeout: 2 * time.Second}
    m := new(dns.Msg)
    m.SetQuestion("svc.geodns.test.", dns.TypeA)
    // add ECS option
    opt := &dns.OPT{Hdr: dns.RR_Header{Name: ".", Rrtype: dns.TypeOPT}}
    ecs := &dns.EDNS0_SUBNET{Code: dns.EDNS0SUBNET, Family: 1, SourceNetmask: 24, SourceScope: 0}
    ecs.Address = net.ParseIP("8.8.8.8")
    opt.Option = append(opt.Option, ecs)
    m.Extra = append(m.Extra, opt)

    in, _, err := c.Exchange(m, dnsAddr)
    if err != nil { t.Fatalf("dns exchange: %v", err) }
    if in.Rcode != dns.RcodeSuccess { t.Fatalf("rcode: %d", in.Rcode) }
    if len(in.Answer) == 0 { t.Fatalf("no answers") }
    // Expect subnet-specific record selected (priority higher than country)
    found := false
    for _, rr := range in.Answer {
        if a, _ := rr.(*dns.A); a != nil && a.A.String() == "198.51.100.13" {
            found = true
            break
        }
    }
    if !found {
        t.Fatalf("expected US-specific A 198.51.100.11, got: %#v", in.Answer)
    }

    _ = dnsServer.Shutdown()
}
