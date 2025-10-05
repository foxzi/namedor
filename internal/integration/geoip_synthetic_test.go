package integration

import (
    "bytes"
    "encoding/json"
    "net"
    "net/http"
    "net/netip"
    "os"
    "path/filepath"
    "testing"
    "time"

    "github.com/maxmind/mmdbwriter"
    "github.com/maxmind/mmdbwriter/mmdbtype"
    "github.com/miekg/dns"

    "smaillgeodns/internal/config"
    "smaillgeodns/internal/db"
    dnssrv "smaillgeodns/internal/server/dns"
    restsrv "smaillgeodns/internal/server/rest"
)

func writeCityMMDB(t *testing.T, path string, entries map[string]struct{ Country, Continent string }) {
    t.Helper()
    w, err := mmdbwriter.New(mmdbwriter.Options{DatabaseType: "GeoLite2-City"})
    if err != nil { t.Fatalf("new writer: %v", err) }
    for cidr, v := range entries {
        _, network, err := net.ParseCIDR(cidr)
        if err != nil { t.Fatalf("parse cidr: %v", err) }
        data := mmdbtype.Map{
            "country":   mmdbtype.Map{"iso_code": mmdbtype.String(v.Country)},
            "continent": mmdbtype.Map{"code": mmdbtype.String(v.Continent)},
        }
        if err := w.Insert(network, data); err != nil { t.Fatalf("insert: %v", err) }
    }
    f, err := os.Create(path)
    if err != nil { t.Fatalf("create city mmdb: %v", err) }
    defer f.Close()
    if _, err := w.WriteTo(f); err != nil { t.Fatalf("write city mmdb: %v", err) }
}

func writeASNMMDB(t *testing.T, path string, entries map[string]uint32) {
    t.Helper()
    w, err := mmdbwriter.New(mmdbwriter.Options{DatabaseType: "GeoLite2-ASN"})
    if err != nil { t.Fatalf("new writer: %v", err) }
    for cidr, asn := range entries {
        _, network, err := net.ParseCIDR(cidr)
        if err != nil { t.Fatalf("parse cidr: %v", err) }
        data := mmdbtype.Map{
            "autonomous_system_number":       mmdbtype.Uint32(asn),
            "autonomous_system_organization": mmdbtype.String("TEST"),
        }
        if err := w.Insert(network, data); err != nil { t.Fatalf("insert: %v", err) }
    }
    f, err := os.Create(path)
    if err != nil { t.Fatalf("create asn mmdb: %v", err) }
    defer f.Close()
    if _, err := w.WriteTo(f); err != nil { t.Fatalf("write asn mmdb: %v", err) }
}

func TestGeoDNS_SyntheticMMDB(t *testing.T) {
    if os.Getenv("GEOIP_SYNTH") != "1" {
        t.Skip("set GEOIP_SYNTH=1 to enable synthetic MMDB test")
    }
    tmp := t.TempDir()
    cityPath := filepath.Join(tmp, "city-ipv4.mmdb")
    asnPath := filepath.Join(tmp, "asn-ipv4.mmdb")
    // Define loopback ranges (requires modified mmdbwriter without 127/8 restriction)
    writeCityMMDB(t, cityPath, map[string]struct{ Country, Continent string }{
        "127.0.1.0/24": {Country: "RU", Continent: "EU"},
        "127.0.2.0/24": {Country: "GB", Continent: "EU"},
    })
    writeASNMMDB(t, asnPath, map[string]uint32{
        "127.0.1.0/24": 65001,
        "127.0.2.0/24": 65002,
    })

    dnsAddr := "127.0.0.1:19056"
    restAddr := "127.0.0.1:18092"
    dbPath := filepath.Join(tmp, "dns.db")
    cfg := &config.Config{
        Listen: dnsAddr, RESTListen: restAddr, APIToken: "devtoken",
        AutoSOAOnMissing: true, DefaultTTL: 60,
        DB: config.DBConfig{Driver: "sqlite", DSN: "file:" + dbPath + "?_foreign_keys=on"},
        GeoIP: config.GeoIPConfig{Enabled: true, MMDBPath: tmp, ReloadSec: 0, UseECS: true},
    }

    gdb, err := db.Open(cfg.DB); if err != nil { t.Fatal(err) }
    if err := db.AutoMigrate(gdb); err != nil { t.Fatal(err) }
    dnsServer, _ := dnssrv.NewServer(cfg, gdb)
    restServer := restsrv.NewServer(cfg, gdb)
    go func() { _ = dnsServer.Start() }()
    go func() { _ = restServer.Start() }()
    if err := waitHTTPReady("http://"+restAddr+"/zones", 5*time.Second); err != nil { t.Fatal(err) }

    // Create zone and rrset with RU, GB, ASN-specific and generic addresses
    type zoneResp struct{ ID uint `json:"id"` }
    zr := zoneResp{}
    reqZ := bytes.NewBufferString(`{"name":"localgeo.test"}`)
    rreq, _ := http.NewRequest("POST", "http://"+restAddr+"/zones", reqZ)
    rreq.Header.Set("Authorization", "Bearer "+cfg.APIToken)
    rreq.Header.Set("Content-Type", "application/json")
    rresp, err := http.DefaultClient.Do(rreq); if err != nil { t.Fatal(err) }
    if rresp.StatusCode != http.StatusCreated { t.Fatalf("zone status %d", rresp.StatusCode) }
    _ = json.NewDecoder(rresp.Body).Decode(&zr); rresp.Body.Close()

    body := `{"name":"svc","type":"A","ttl":60,"records":[
        {"data":"203.0.113.21","subnet":"127.0.1.0/24"},
        {"data":"203.0.113.22","subnet":"127.0.2.0/24"},
        {"data":"203.0.113.10","asn":65001},
        {"data":"203.0.113.14","asn":65002},
        {"data":"203.0.113.11","country":"RU"},
        {"data":"203.0.113.12","country":"GB"},
        {"data":"203.0.113.13"}
    ]}`
    reqR, _ := http.NewRequest("POST", "http://"+restAddr+"/zones/"+itoa(zr.ID)+"/rrsets", bytes.NewBufferString(body))
    reqR.Header.Set("Authorization", "Bearer "+cfg.APIToken)
    reqR.Header.Set("Content-Type", "application/json")
    rresp2, err := http.DefaultClient.Do(reqR); if err != nil { t.Fatal(err) }
    if rresp2.StatusCode != http.StatusCreated { t.Fatalf("rrset status %d", rresp2.StatusCode) }
    rresp2.Body.Close()

    // ECS with 127.0.1.1 (RU by our synthetic DB, ASN 65001) must pick ASN-specific first
    ecsA := netip.MustParseAddr("127.0.1.1")
    wantA := "203.0.113.21"
    assertDNSAnswer(t, dnsAddr, "svc.localgeo.test.", ecsA, wantA)

    // ECS with 127.0.2.1 should match subnet priority
    ecsB := netip.MustParseAddr("127.0.2.1")
    wantB := "203.0.113.22"
    assertDNSAnswer(t, dnsAddr, "svc.localgeo.test.", ecsB, wantB)

    _ = dnsServer.Shutdown()
}

func assertDNSAnswer(t *testing.T, dnsAddr, qname string, ecsIP netip.Addr, want string) {
    t.Helper()
    c := &dns.Client{Timeout: 2 * time.Second}
    m := new(dns.Msg)
    m.SetQuestion(qname, dns.TypeA)
    // ECS option
    opt := &dns.OPT{Hdr: dns.RR_Header{Name: ".", Rrtype: dns.TypeOPT}}
    fam := uint16(1)
    if ecsIP.Is6() { fam = 2 }
    e := &dns.EDNS0_SUBNET{Code: dns.EDNS0SUBNET, Family: fam, SourceNetmask: 24}
    if ecsIP.Is6() { e.SourceNetmask = 56 }
    e.Address = net.ParseIP(ecsIP.String())
    opt.Option = append(opt.Option, e)
    m.Extra = append(m.Extra, opt)
    in, _, err := c.Exchange(m, dnsAddr)
    if err != nil { t.Fatalf("dns exchange: %v", err) }
    if in.Rcode != dns.RcodeSuccess { t.Fatalf("rcode %d", in.Rcode) }
    found := false
    for _, rr := range in.Answer {
        if a, _ := rr.(*dns.A); a != nil && a.A.String() == want { found = true; break }
    }
    if !found { t.Fatalf("want %s, got %#v", want, in.Answer) }
}

func TestGeoDNS_SyntheticMMDB_IPv6(t *testing.T) {
    if os.Getenv("GEOIP_SYNTH") != "1" {
        t.Skip("set GEOIP_SYNTH=1 to enable synthetic MMDB test")
    }
    tmp := t.TempDir()
    cityPath := filepath.Join(tmp, "city-ipv6.mmdb")
    asnPath := filepath.Join(tmp, "asn-ipv6.mmdb")
    // Use documentation ranges 2001:db8::/32 for IPv6 samples
    writeCityMMDB(t, cityPath, map[string]struct{ Country, Continent string }{
        "2001:db8:1::/64": {Country: "RU", Continent: "EU"},
        "2001:db8:2::/64": {Country: "GB", Continent: "EU"},
    })
    writeASNMMDB(t, asnPath, map[string]uint32{
        "2001:db8:1::/64": 65101,
        "2001:db8:2::/64": 65102,
    })

    dnsAddr := "127.0.0.1:19057"
    restAddr := "127.0.0.1:18093"
    dbPath := filepath.Join(tmp, "dns.db")
    cfg := &config.Config{
        Listen: dnsAddr, RESTListen: restAddr, APIToken: "devtoken",
        AutoSOAOnMissing: true, DefaultTTL: 60,
        DB: config.DBConfig{Driver: "sqlite", DSN: "file:" + dbPath + "?_foreign_keys=on"},
        GeoIP: config.GeoIPConfig{Enabled: true, MMDBPath: tmp, ReloadSec: 0, UseECS: true},
    }
    gdb, err := db.Open(cfg.DB); if err != nil { t.Fatal(err) }
    if err := db.AutoMigrate(gdb); err != nil { t.Fatal(err) }
    dnsServer, _ := dnssrv.NewServer(cfg, gdb)
    restServer := restsrv.NewServer(cfg, gdb)
    go func() { _ = dnsServer.Start() }()
    go func() { _ = restServer.Start() }()
    if err := waitHTTPReady("http://"+restAddr+"/zones", 5*time.Second); err != nil { t.Fatal(err) }

    type zoneResp struct{ ID uint `json:"id"` }
    zr := zoneResp{}
    reqZ := bytes.NewBufferString(`{"name":"localgeo6.test"}`)
    rreq, _ := http.NewRequest("POST", "http://"+restAddr+"/zones", reqZ)
    rreq.Header.Set("Authorization", "Bearer "+cfg.APIToken)
    rreq.Header.Set("Content-Type", "application/json")
    rresp, err := http.DefaultClient.Do(rreq); if err != nil { t.Fatal(err) }
    if rresp.StatusCode != http.StatusCreated { t.Fatalf("zone status %d", rresp.StatusCode) }
    _ = json.NewDecoder(rresp.Body).Decode(&zr); rresp.Body.Close()

    // AAAA rrset with subnet/ASN/country/generic
    body := `{"name":"svc","type":"AAAA","ttl":60,"records":[
        {"data":"2001:db8:100::21","subnet":"2001:db8:1::/64"},
        {"data":"2001:db8:100::22","subnet":"2001:db8:2::/64"},
        {"data":"2001:db8:100::10","asn":65101},
        {"data":"2001:db8:100::14","asn":65102},
        {"data":"2001:db8:100::11","country":"RU"},
        {"data":"2001:db8:100::12","country":"GB"},
        {"data":"2001:db8:100::13"}
    ]}`
    reqR, _ := http.NewRequest("POST", "http://"+restAddr+"/zones/"+itoa(zr.ID)+"/rrsets", bytes.NewBufferString(body))
    reqR.Header.Set("Authorization", "Bearer "+cfg.APIToken)
    reqR.Header.Set("Content-Type", "application/json")
    rresp2, err := http.DefaultClient.Do(reqR); if err != nil { t.Fatal(err) }
    if rresp2.StatusCode != http.StatusCreated { t.Fatalf("rrset status %d", rresp2.StatusCode) }
    rresp2.Body.Close()

    assertDNSAnswerAAAA(t, dnsAddr, "svc.localgeo6.test.", netip.MustParseAddr("2001:db8:1::1"), "2001:db8:100::21")
    assertDNSAnswerAAAA(t, dnsAddr, "svc.localgeo6.test.", netip.MustParseAddr("2001:db8:2::1"), "2001:db8:100::22")

    _ = dnsServer.Shutdown()
}

func assertDNSAnswerAAAA(t *testing.T, dnsAddr, qname string, ecsIP netip.Addr, want string) {
    t.Helper()
    c := &dns.Client{Timeout: 2 * time.Second}
    m := new(dns.Msg)
    m.SetQuestion(qname, dns.TypeAAAA)
    // ECS option
    opt := &dns.OPT{Hdr: dns.RR_Header{Name: ".", Rrtype: dns.TypeOPT}}
    fam := uint16(2)
    e := &dns.EDNS0_SUBNET{Code: dns.EDNS0SUBNET, Family: fam, SourceNetmask: 56}
    e.Address = net.ParseIP(ecsIP.String())
    opt.Option = append(opt.Option, e)
    m.Extra = append(m.Extra, opt)
    in, _, err := c.Exchange(m, dnsAddr)
    if err != nil { t.Fatalf("dns exchange: %v", err) }
    if in.Rcode != dns.RcodeSuccess { t.Fatalf("rcode %d", in.Rcode) }
    found := false
    for _, rr := range in.Answer {
        if aaaa, _ := rr.(*dns.AAAA); aaaa != nil && aaaa.AAAA.String() == want { found = true; break }
    }
    if !found { t.Fatalf("want %s, got %#v", want, in.Answer) }
}
