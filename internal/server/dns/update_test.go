package dns

import (
    "net"
    "testing"

    "github.com/miekg/dns"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"

    "smaillgeodns/internal/config"
    dbm "smaillgeodns/internal/db"
)

type fakeWriter struct{ msg *dns.Msg }

func (f *fakeWriter) LocalAddr() net.Addr  { return &net.UDPAddr{IP: net.IPv4(127,0,0,1)} }
func (f *fakeWriter) RemoteAddr() net.Addr { return &net.UDPAddr{IP: net.IPv4(127,0,0,1)} }
func (f *fakeWriter) WriteMsg(m *dns.Msg) error { f.msg = m; return nil }
func (f *fakeWriter) Write(b []byte) (int, error) { return len(b), nil }
func (f *fakeWriter) Close() error                { return nil }
func (f *fakeWriter) TsigStatus() error         { return nil }
func (f *fakeWriter) TsigTimersOnly(bool)       {}
func (f *fakeWriter) Hijack()                   {}

func testDB(t *testing.T) *gorm.DB {
    t.Helper()
    db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
    if err != nil { t.Fatalf("open db: %v", err) }
    if err := dbm.AutoMigrate(db); err != nil { t.Fatalf("migrate: %v", err) }
    return db
}

func TestDynamicUpdate_AddAndDelete(t *testing.T) {
    db := testDB(t)
    // Create zone with SOA (for serial bump)
    z := dbm.Zone{Name: "adddel.test"}
    if err := db.Create(&z).Error; err != nil { t.Fatalf("create zone: %v", err) }
    soa := dbm.RRSet{ZoneID: z.ID, Name: "adddel.test.", Type: "SOA", TTL: 3600,
        Records: []dbm.RData{{Data: "ns1.example.com. hostmaster.example.com. 2025010101 7200 3600 1209600 300"}}}
    if err := db.Create(&soa).Error; err != nil { t.Fatalf("create soa: %v", err) }

    cfg := &config.Config{Update: config.UpdateConfig{Enabled: true, RequireTSIG: false}, DefaultTTL: 300}
    srv, err := NewServer(cfg, db)
    if err != nil { t.Fatalf("server: %v", err) }

    // Build dynamic update to add A record
    add := new(dns.Msg)
    add.SetUpdate("adddel.test.")
    rr, _ := dns.NewRR("www.adddel.test. 0 IN A 192.0.2.10")
    add.Ns = append(add.Ns, rr)

    fw := &fakeWriter{}
    srv.handleUpdate(fw, add)
    if fw.msg == nil || fw.msg.Rcode != dns.RcodeSuccess { t.Fatalf("add rcode=%d", fw.msg.Rcode) }

    // Verify record exists
    var set dbm.RRSet
    if err := db.Preload("Records").Where("zone_id = ? AND name = ? AND type = ?", z.ID, "www.adddel.test.", "A").First(&set).Error; err != nil {
        t.Fatalf("missing A set: %v", err)
    }
    if len(set.Records) != 1 || set.Records[0].Data != "192.0.2.10" { t.Fatalf("unexpected records: %+v", set.Records) }
    if set.TTL != 300 { t.Fatalf("expected TTL=300 from DefaultTTL, got %d", set.TTL) }

    // Build dynamic update to delete specific RR (NONE)
    delRR, _ := dns.NewRR("www.adddel.test. 0 NONE A 192.0.2.10")
    del := new(dns.Msg)
    del.SetUpdate("adddel.test.")
    del.Ns = append(del.Ns, delRR)
    fw2 := &fakeWriter{}
    srv.handleUpdate(fw2, del)
    if fw2.msg == nil || fw2.msg.Rcode != dns.RcodeSuccess { t.Fatalf("delete rcode=%d", fw2.msg.Rcode) }

    var cnt int64
    if err := db.Model(&dbm.RData{}).Where("rr_set_id = ?", set.ID).Count(&cnt).Error; err != nil { t.Fatalf("count: %v", err) }
    if cnt != 0 { t.Fatalf("expected 0 records, got %d", cnt) }
}

func TestDynamicUpdate_DefaultTTLZero_NoOverride(t *testing.T) {
    db := testDB(t)
    // Create zone with SOA
    z := dbm.Zone{Name: "defaultttl.test"}
    if err := db.Create(&z).Error; err != nil { t.Fatalf("create zone: %v", err) }
    soa := dbm.RRSet{ZoneID: z.ID, Name: "defaultttl.test.", Type: "SOA", TTL: 3600,
        Records: []dbm.RData{{Data: "ns1.example.com. hostmaster.example.com. 2025010101 7200 3600 1209600 300"}}}
    if err := db.Create(&soa).Error; err != nil { t.Fatalf("create soa: %v", err) }

    cfg := &config.Config{Update: config.UpdateConfig{Enabled: true, RequireTSIG: false}, DefaultTTL: 0}
    srv, err := NewServer(cfg, db)
    if err != nil { t.Fatalf("server: %v", err) }

    add := new(dns.Msg)
    add.SetUpdate("defaultttl.test.")
    rr, _ := dns.NewRR("www.defaultttl.test. 0 IN A 192.0.2.11")
    add.Ns = append(add.Ns, rr)
    fw := &fakeWriter{}
    srv.handleUpdate(fw, add)
    if fw.msg == nil || fw.msg.Rcode != dns.RcodeSuccess { t.Fatalf("add rcode=%d", fw.msg.Rcode) }

    var set dbm.RRSet
    if err := db.Preload("Records").Where("zone_id = ? AND name = ? AND type = ?", z.ID, "www.defaultttl.test.", "A").First(&set).Error; err != nil {
        t.Fatalf("missing A set: %v", err)
    }
    if set.TTL != 0 { t.Fatalf("expected TTL=0 preserved when DefaultTTL=0, got %d", set.TTL) }
}
