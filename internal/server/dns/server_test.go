package dns

import (
    "net"
    "net/netip"
    "testing"
    "time"

    "github.com/miekg/dns"

    "smaillgeodns/internal/cache"
    dbm "smaillgeodns/internal/db"
    "smaillgeodns/internal/geoip"
)

func TestSelectGeoRecords(t *testing.T) {
    ip := netip.MustParseAddr("203.0.113.5")
    recs := []dbm.RData{
        {Data: "192.0.2.1"},
        {Data: "192.0.2.2", Subnet: strPtr("203.0.113.0/24")},
        {Data: "192.0.2.3", Country: strPtr("US")},
    }
    out := selectGeoRecords(recs, ip, geoip.Info{})
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
    s.cache.Set("www.example.com.|1", old, time.Minute)

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
