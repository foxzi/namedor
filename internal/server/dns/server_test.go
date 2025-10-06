package dns

import (
    "net"
    "net/netip"
    "testing"
    "time"

    "github.com/miekg/dns"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"

    "namedot/internal/cache"
    "namedot/internal/config"
    dbm "namedot/internal/db"
    "namedot/internal/geoip"
)

func TestSelectGeoRecords(t *testing.T) {
    ip := netip.MustParseAddr("203.0.113.5")
    recs := []dbm.RData{
        {Data: "192.0.2.1"},
        {Data: "192.0.2.2", Subnet: strPtr("203.0.113.0/24")},
        {Data: "192.0.2.3", Country: strPtr("US")},
    }
    out, rule := selectGeoRecords(recs, ip, geoip.Info{})
    if rule != "subnet" {
        t.Fatalf("expected rule subnet, got %s", rule)
    }
    if len(out) != 1 || out[0].Data != "192.0.2.2" {
        t.Fatalf("expected subnet match, got %#v", out)
    }
}

func strPtr(s string) *string { return &s }

// cacheWriter verifies that cached response gets current query ID
type cacheWriter struct{ wrote *uint16 }

func (cw *cacheWriter) WriteMsg(m *dns.Msg) error { if cw.wrote != nil { *cw.wrote = m.Id }; return nil }
func (cw *cacheWriter) LocalAddr() net.Addr       { return &net.UDPAddr{} }
func (cw *cacheWriter) RemoteAddr() net.Addr      { return &net.UDPAddr{} }
func (cw *cacheWriter) Write(b []byte) (int, error) { return len(b), nil }
func (cw *cacheWriter) Close() error              { return nil }
func (cw *cacheWriter) TsigStatus() error         { return nil }
func (cw *cacheWriter) TsigTimersOnly(bool)       {}
func (cw *cacheWriter) Hijack()                   {}

func TestCacheResponse_UsesCurrentID(t *testing.T) {
    s := &Server{cache: cache.New(10)}
    // Prepare cached message with old ID
    old := new(dns.Msg)
    old.SetReply(&dns.Msg{MsgHdr: dns.MsgHdr{Id: 111}})
    old.Question = []dns.Question{{Name: "www.example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET}}
    s.cache.Set("www.example.com.|1|", old, time.Minute)

    // Incoming query with new ID
    req := new(dns.Msg)
    req.Id = 222
    req.Question = []dns.Question{{Name: "www.example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET}}

    var got uint16
    cw := &cacheWriter{wrote: &got}
    s.serveDNS(cw, req)
    if got != 222 {
        t.Fatalf("cached response used wrong ID: got %d want 222", got)
    }
}

func TestLookup_CNAME_Fallback(t *testing.T) {
    // Setup in-memory DB and server
    db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
    if err != nil { t.Fatalf("open db: %v", err) }
    if err := db.AutoMigrate(&dbm.Zone{}, &dbm.RRSet{}, &dbm.RData{}); err != nil { t.Fatalf("migrate: %v", err) }

    cfg := &config.Config{Listen: ":0", RESTListen: ":0", Performance: config.PerformanceConfig{CacheSize: 0, ForwarderTimeoutSec: 1}, GeoIP: config.GeoIPConfig{Enabled: false}}
    s, err := NewServer(cfg, db)
    if err != nil { t.Fatalf("new server: %v", err) }

    // Create zone and CNAME at foo.example.com.
    z := dbm.Zone{Name: "example.com"}
    if err := db.Create(&z).Error; err != nil { t.Fatalf("create zone: %v", err) }
    cname := dbm.RRSet{ZoneID: z.ID, Name: "foo.example.com.", Type: "CNAME", TTL: 300, Records: []dbm.RData{{Data: "bar.example.net."}}}
    if err := db.Create(&cname).Error; err != nil { t.Fatalf("create cname: %v", err) }

    // Query A foo.example.com. should return CNAME rrset
    q := dns.Question{Name: "foo.example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET}
    msg := new(dns.Msg)
    ans, ttl, err := s.lookup(msg, q, netip.Addr{})
    if err != nil { t.Fatalf("lookup err: %v", err) }
    if ttl != 300 { t.Fatalf("ttl want 300 got %d", ttl) }
    if len(ans) == 0 { t.Fatalf("no answers") }
    if ans[0].Header().Rrtype != dns.TypeCNAME { t.Fatalf("want CNAME got %s", dns.TypeToString[ans[0].Header().Rrtype]) }
}
