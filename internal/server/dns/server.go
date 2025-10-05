package dns

import (
    "context"
    "fmt"
    "log"
    "net"
    "net/netip"
    "strings"
    "time"

    "github.com/miekg/dns"
    "gorm.io/gorm"

    "smaillgeodns/internal/cache"
    "smaillgeodns/internal/config"
    dbm "smaillgeodns/internal/db"
    "smaillgeodns/internal/geoip"
)

type Server struct {
    cfg       *config.Config
    db        *gorm.DB
    udpServer *dns.Server
    tcpServer *dns.Server
    resolver  *dns.Client
    cache     *cache.Cache
    geo       geoip.Provider
    geoStop   func()
    lastRule  string
}

func NewServer(cfg *config.Config, db *gorm.DB) (*Server, error) {
    s := &Server{
        cfg:      cfg,
        db:       db,
        resolver: &dns.Client{Timeout: 2 * time.Second},
        cache:    cache.New(1024),
    }
    // GeoIP provider
    if cfg.GeoIP.Enabled && cfg.GeoIP.MMDBPath != "" {
        prov, stop, err := geoip.NewFromPath(cfg.GeoIP.MMDBPath, time.Duration(cfg.GeoIP.ReloadSec)*time.Second)
        if err != nil {
            log.Printf("GeoIP: %v; disabling GeoDNS", err)
            s.geo = geoip.NewNoop()
        } else {
            s.geo = prov
            s.geoStop = stop
        }
    } else {
        s.geo = geoip.NewNoop()
    }
    return s, nil
}

func (s *Server) Start() error {
    dns.HandleFunc(".", s.serveDNS)
    s.udpServer = &dns.Server{Addr: s.cfg.Listen, Net: "udp", TsigSecret: s.cfg.Update.TSIGSecrets}
    s.tcpServer = &dns.Server{Addr: s.cfg.Listen, Net: "tcp", TsigSecret: s.cfg.Update.TSIGSecrets}

    go func() {
        if err := s.udpServer.ListenAndServe(); err != nil {
            log.Fatalf("failed to start UDP server: %v", err)
        }
    }()
    go func() {
        if err := s.tcpServer.ListenAndServe(); err != nil {
            log.Fatalf("failed to start TCP server: %v", err)
        }
    }()
    return nil
}

func (s *Server) Shutdown() error {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    if s.udpServer != nil {
        _ = s.udpServer.ShutdownContext(ctx)
    }
    if s.tcpServer != nil {
        _ = s.tcpServer.ShutdownContext(ctx)
    }
    if s.geoStop != nil {
        s.geoStop()
    }
    return nil
}

func (s *Server) serveDNS(w dns.ResponseWriter, r *dns.Msg) {
    // Dynamic update
    if r.Opcode == dns.OpcodeUpdate {
        if len(r.Question) > 0 {
            q := r.Question[0]
            log.Printf("DNS UPDATE zone=%s from=%s id=%d", q.Name, w.RemoteAddr(), r.Id)
        } else {
            log.Printf("DNS UPDATE from=%s id=%d", w.RemoteAddr(), r.Id)
        }
        s.handleUpdate(w, r)
        return
    }
    m := new(dns.Msg)
    m.SetReply(r)
    m.Authoritative = true

    if len(r.Question) == 0 {
        _ = w.WriteMsg(m)
        return
    }
    q := r.Question[0]
    // Determine client IP (ECS or remote) for geo and cache scoping
    useECS := false
    if s.cfg != nil {
        useECS = s.cfg.GeoIP.UseECS
    }
    cip := clientIPFrom(r, w, useECS)
    prov := s.geo
    if prov == nil {
        prov = geoip.NewNoop()
    }
    ginfo := prov.Lookup(cip)
    verbose := false
    if s.cfg != nil {
        verbose = s.cfg.Log.DNSVerbose
    }
    geoStr := ""
    if verbose {
        geoStr = fmt.Sprintf(" geo[c=%s,ct=%s,asn=%d]", ginfo.Country, ginfo.Continent, ginfo.ASN)
    }

    // Cache key
    cacheScope := cip.String()
    if !cip.IsValid() { cacheScope = "" }
    key := fmt.Sprintf("%s|%d|%s", strings.ToLower(q.Name), q.Qtype, cacheScope)
    if v, ok := s.cache.Get(key); ok {
        if cached, ok2 := v.(*dns.Msg); ok2 {
            log.Printf("DNS QUERY cache-hit q=%s type=%s from=%s%s id=%d", q.Name, dns.TypeToString[q.Qtype], w.RemoteAddr(), geoStr, r.Id)
            resp := cached.Copy()
            // Update transaction ID and question to match current request
            resp.Id = r.Id
            resp.Question = r.Question
            _ = w.WriteMsg(resp)
            return
        }
    }

    // Resolve locally
    answers, ttl, err := s.lookup(r, q, cip)
    if err == nil && len(answers) > 0 {
        if verbose {
            log.Printf("DNS QUERY q=%s type=%s from=%s ecs=%s%s rule=%s answers=%d ttl=%d id=%d", q.Name, dns.TypeToString[q.Qtype], w.RemoteAddr(), cip, geoStr, s.lastRule, len(answers), ttl, r.Id)
        } else {
            log.Printf("DNS QUERY q=%s type=%s from=%s answers=%d ttl=%d id=%d", q.Name, dns.TypeToString[q.Qtype], w.RemoteAddr(), len(answers), ttl, r.Id)
        }
        m.Answer = answers
        _ = w.WriteMsg(m)
        if ttl > 0 {
            // Store a copy in cache to avoid mutating original
            s.cache.Set(key, m.Copy(), time.Duration(ttl)*time.Second)
        }
        return
    }

    // Forward on miss
    if s.cfg.Forwarder != "" {
        fwd := new(dns.Msg)
        fwd.SetQuestion(dns.Fqdn(q.Name), q.Qtype)
        in, _, ferr := s.resolver.Exchange(fwd, net.JoinHostPort(s.cfg.Forwarder, "53"))
        if ferr == nil && in != nil {
            log.Printf("DNS QUERY forward q=%s type=%s to=%s%s rcode=%d id=%d", q.Name, dns.TypeToString[q.Qtype], s.cfg.Forwarder, geoStr, in.Rcode, r.Id)
            in.Id = r.Id
            _ = w.WriteMsg(in)
            return
        }
    }

    log.Printf("DNS QUERY nxdomain q=%s type=%s from=%s%s id=%d", q.Name, dns.TypeToString[q.Qtype], w.RemoteAddr(), geoStr, r.Id)
    m.Rcode = dns.RcodeNameError
    _ = w.WriteMsg(m)
}

// lookup resolves a question from DB applying Geo selection.
func (s *Server) lookup(r *dns.Msg, q dns.Question, clientIP netip.Addr) (answers []dns.RR, ttl uint32, err error) {
    qname := strings.ToLower(dns.Fqdn(q.Name))
    qtype := dns.TypeToString[q.Qtype]

    // Find the best matching zone suffix
    var zones []dbm.Zone
    if err := s.db.Order("length(name) desc").Find(&zones).Error; err != nil {
        return nil, 0, err
    }
    var zone *dbm.Zone
    for i := range zones {
        name := dns.Fqdn(strings.ToLower(zones[i].Name))
        if strings.HasSuffix(qname, name) {
            zone = &zones[i]
            break
        }
    }
    if zone == nil {
        return nil, 0, fmt.Errorf("no zone")
    }

    // Find RRSet by FQDN name and type
    var set dbm.RRSet
    if err := s.db.Preload("Records").
        Where("zone_id = ? AND name = ? AND type = ?", zone.ID, strings.ToLower(qname), strings.ToUpper(qtype)).
        First(&set).Error; err != nil {
        return nil, 0, err
    }

    // Geo selection
    g := s.geo.Lookup(clientIP)
    recs, rule := selectGeoRecords(set.Records, clientIP, g)
    s.lastRule = rule

    for _, rec := range recs {
        rr, perr := dns.NewRR(fmt.Sprintf("%s %d %s %s", qname, set.TTL, strings.ToUpper(qtype), rec.Data))
        if perr == nil {
            answers = append(answers, rr)
        }
    }
    return answers, set.TTL, nil
}

func clientIPFrom(r *dns.Msg, w dns.ResponseWriter, useECS bool) netip.Addr {
    if useECS {
        if opt := r.IsEdns0(); opt != nil {
            for _, o := range opt.Option {
                if ecs, ok := o.(*dns.EDNS0_SUBNET); ok {
                    var ip net.IP
                    if ecs.Family == 1 { // IPv4
                        ip = ecs.Address.To4()
                    } else {
                        ip = ecs.Address
                    }
                    if ip != nil {
                        a, _ := netip.ParseAddr(ip.String())
                        return a
                    }
                }
            }
        }
    }
    if ra := w.RemoteAddr(); ra != nil {
        host, _, err := net.SplitHostPort(ra.String())
        if err == nil {
            if a, err2 := netip.ParseAddr(host); err2 == nil { return a }
        }
    }
    return netip.Addr{}
}

// remapIP maps an IP from one CIDR into another CIDR with the same prefix length.
// Useful to translate reserved ranges (e.g., 127.0.1.0/24) into TEST-NET for GeoIP lookup.

func selectGeoRecords(recs []dbm.RData, ip netip.Addr, g geoip.Info) ([]dbm.RData, string) {
    if len(recs) == 0 {
        return recs, "none"
    }
    // If no IP, return generic ones or all
    if !ip.IsValid() {
        out := make([]dbm.RData, 0, len(recs))
        for _, r := range recs {
            if r.Country == nil && r.Continent == nil && r.ASN == nil && r.Subnet == nil {
                out = append(out, r)
            }
        }
        if len(out) > 0 {
            return out, "generic"
        }
        return recs, "all"
    }
    // Priority: subnet > asn > country > continent > default
    var subnetMatch, asnMatch, countryMatch, continentMatch, generic []dbm.RData
    for _, r := range recs {
        if r.Subnet != nil {
            if p, err := netip.ParsePrefix(*r.Subnet); err == nil && p.Contains(ip) {
                subnetMatch = append(subnetMatch, r)
                continue
            }
        }
        if r.ASN != nil {
            if g.ASN != 0 && *r.ASN == g.ASN {
                asnMatch = append(asnMatch, r)
                continue
            }
        }
        if r.Country != nil && g.Country != "" && strings.EqualFold(*r.Country, g.Country) {
            countryMatch = append(countryMatch, r)
            continue
        }
        if r.Continent != nil && g.Continent != "" && strings.EqualFold(*r.Continent, g.Continent) {
            continentMatch = append(continentMatch, r)
            continue
        }
        if r.Country == nil && r.Continent == nil && r.ASN == nil && r.Subnet == nil {
            generic = append(generic, r)
        }
    }
    if len(subnetMatch) > 0 {
        return subnetMatch, "subnet"
    }
    if len(asnMatch) > 0 {
        return asnMatch, "asn"
    }
    if len(countryMatch) > 0 {
        return countryMatch, "country"
    }
    if len(continentMatch) > 0 {
        return continentMatch, "continent"
    }
    if len(generic) > 0 {
        return generic, "generic"
    }
    return recs, "all"
}

// handleUpdate processes RFC 2136 dynamic updates (basic ADD/DELETE semantics).
func (s *Server) handleUpdate(w dns.ResponseWriter, r *dns.Msg) {
    // Authorization: TSIG if configured
    if s.cfg.Update.Enabled {
        if s.cfg.Update.RequireTSIG {
            signed := false
            for _, rr := range r.Extra {
                if _, ok := rr.(*dns.TSIG); ok {
                    signed = true
                    break
                }
            }
            if !signed {
                m := new(dns.Msg)
                m.SetReply(r)
                m.Rcode = dns.RcodeNotAuth
                _ = w.WriteMsg(m)
                return
            }
        }
        if len(s.cfg.Update.TSIGSecrets) > 0 {
            if err := w.TsigStatus(); err != nil {
                m := new(dns.Msg)
                m.SetReply(r)
                m.Rcode = dns.RcodeNotAuth
                _ = w.WriteMsg(m)
                return
            }
        }
    } else {
        m := new(dns.Msg)
        m.SetReply(r)
        m.Rcode = dns.RcodeRefused
        _ = w.WriteMsg(m)
        return
    }

    // Zone section must contain one entry specifying the zone
    if len(r.Question) == 0 {
        m := new(dns.Msg)
        m.SetReply(r)
        m.Rcode = dns.RcodeFormatError
        _ = w.WriteMsg(m)
        return
    }
    zname := strings.ToLower(dns.Fqdn(r.Question[0].Name))
    var zone dbm.Zone
    if err := s.db.Where("name = ?", strings.TrimSuffix(zname, ".")).Or("name = ?", zname).First(&zone).Error; err != nil {
        m := new(dns.Msg)
        m.SetReply(r)
        m.Rcode = dns.RcodeRefused
        _ = w.WriteMsg(m)
        return
    }

    // Process updates from the Update section (r.Ns)
    err := s.db.Transaction(func(tx *gorm.DB) error {
        for _, rr := range r.Ns {
            hdr := rr.Header()
            name := strings.ToLower(dns.Fqdn(hdr.Name))
            typ := strings.ToUpper(dns.TypeToString[hdr.Rrtype])
            cls := hdr.Class
            // Restrict to this zone
            if !strings.HasSuffix(name, dns.Fqdn(zone.Name)) {
                return fmt.Errorf("name outside zone: %s", name)
            }
            // Delete all (ANY ANY)
            if cls == dns.ClassANY && hdr.Rrtype == dns.TypeANY {
                if err := tx.Where("zone_id = ? AND name = ?", zone.ID, name).Delete(&dbm.RRSet{}).Error; err != nil {
                    return err
                }
                continue
            }
            // Delete rrset (ANY <type>)
            if cls == dns.ClassANY {
                if err := tx.Where("zone_id = ? AND name = ? AND type = ?", zone.ID, name, typ).Delete(&dbm.RRSet{}).Error; err != nil {
                    return err
                }
                continue
            }
            // Delete specific RR (NONE)
            if cls == dns.ClassNONE {
                var set dbm.RRSet
                if err := tx.Preload("Records").Where("zone_id = ? AND name = ? AND type = ?", zone.ID, name, typ).First(&set).Error; err != nil {
                    continue
                }
                data := rrDataString(rr)
                // delete matching records
                if err := tx.Where("rr_set_id = ? AND data = ?", set.ID, data).Delete(&dbm.RData{}).Error; err != nil {
                    return err
                }
                continue
            }
            // Otherwise: add (IN)
            var set dbm.RRSet
            if err := tx.Where("zone_id = ? AND name = ? AND type = ?", zone.ID, name, typ).First(&set).Error; err != nil {
                ttl := hdr.Ttl
                if ttl == 0 && s.cfg.DefaultTTL > 0 {
                    ttl = s.cfg.DefaultTTL
                }
                set = dbm.RRSet{ZoneID: zone.ID, Name: name, Type: typ, TTL: ttl}
                if err := tx.Create(&set).Error; err != nil {
                    return err
                }
            } else if hdr.Ttl > 0 {
                set.TTL = hdr.Ttl
                if err := tx.Save(&set).Error; err != nil {
                    return err
                }
            } else if set.TTL == 0 && s.cfg.DefaultTTL > 0 {
                set.TTL = s.cfg.DefaultTTL
                if err := tx.Save(&set).Error; err != nil { return err }
            }
            rec := dbm.RData{RRSetID: set.ID, Data: rrDataString(rr)}
            if err := tx.Create(&rec).Error; err != nil {
                return err
            }
        }
        return nil
    })

    m := new(dns.Msg)
    m.SetReply(r)
    if err != nil {
        m.Rcode = dns.RcodeServerFailure
        _ = w.WriteMsg(m)
        return
    }
    // bump SOA serial (best-effort)
    dbm.BumpSOASerialAuto(s.db, zone, s.cfg.AutoSOAOnMissing)

    m.Rcode = dns.RcodeSuccess
    _ = w.WriteMsg(m)
}

// rrDataString extracts the RDATA portion of an RR as a string.
func rrDataString(rr dns.RR) string {
    s := rr.String()
    fields := strings.Fields(s)
    if len(fields) < 5 {
        return s
    }
    return strings.Join(fields[4:], " ")
}
