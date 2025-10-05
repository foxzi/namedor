package geoip

import (
    "errors"
    "fmt"
    "log"
    "net/netip"
    "os"
    "path/filepath"
    "strings"
    "sync/atomic"
    "time"

    geoip2 "github.com/oschwald/geoip2-golang"
)

type Info struct {
    Country   string
    Continent string
    ASN       int
}

type Provider interface {
    Lookup(ip netip.Addr) Info
}

type noop struct{}

func (noop) Lookup(ip netip.Addr) Info { return Info{} }

func NewNoop() Provider { return noop{} }

// MaxMind provider that can load City/ASN DBs for IPv4 and IPv6, with hot-reload.
type maxmind struct {
    path string // file or directory

    city4 atomic.Value // *geoip2.Reader
    city6 atomic.Value // *geoip2.Reader
    asn4  atomic.Value // *geoip2.Reader
    asn6  atomic.Value // *geoip2.Reader
}

// NewFromPath loads GeoIP databases. If path is a directory, loads all .mmdb files inside.
// Otherwise treats path as a single City DB usable for both families.
func NewFromPath(path string, reload time.Duration) (Provider, func(), error) {
    m := &maxmind{path: path}
    load := func() error {
        // Close previous
        for _, v := range []atomic.Value{m.city4, m.city6, m.asn4, m.asn6} {
            if r, ok := v.Load().(*geoip2.Reader); ok && r != nil {
                _ = r.Close()
            }
        }

        fi, err := os.Stat(path)
        if err != nil {
            return fmt.Errorf("stat %s: %w", path, err)
        }
        if fi.IsDir() {
            entries, err := os.ReadDir(path)
            if err != nil { return err }
            for _, e := range entries {
                if e.IsDir() { continue }
                if !strings.HasSuffix(strings.ToLower(e.Name()), ".mmdb") { continue }
                full := filepath.Join(path, e.Name())
                r, err := geoip2.Open(full)
                if err != nil { continue }
                t := strings.ToLower(r.Metadata().DatabaseType)
                name := strings.ToLower(e.Name())
                // Detect IPv6 hint in filename: ipv6, -6, _6, -v6, .6.mmdb
                is6Hint := strings.Contains(name, "ipv6") || strings.Contains(name, "-6") ||
                           strings.Contains(name, "_6") || strings.Contains(name, "-v6") ||
                           strings.HasSuffix(name, "6.mmdb")
                isASN := strings.Contains(t, "asn") || strings.Contains(name, "asn")
                if isASN {
                    if is6Hint {
                        m.asn6.Store(r)
                        log.Printf("GeoIP: loaded ASN IPv6 DB %s", full)
                    } else {
                        m.asn4.Store(r)
                        log.Printf("GeoIP: loaded ASN IPv4 DB %s", full)
                    }
                } else {
                    if is6Hint {
                        m.city6.Store(r)
                        log.Printf("GeoIP: loaded City IPv6 DB %s", full)
                    } else {
                        m.city4.Store(r)
                        log.Printf("GeoIP: loaded City IPv4 DB %s", full)
                    }
                }
            }
            // Universal MMDB (IPVersion=6) supports both IPv4+IPv6
            // Use loaded DBs as fallback for missing IP version
            if m.city4.Load() == nil && m.city6.Load() != nil {
                m.city4.Store(m.city6.Load())
                log.Printf("GeoIP: using IPv6 City DB as fallback for IPv4")
            }
            if m.city6.Load() == nil && m.city4.Load() != nil {
                m.city6.Store(m.city4.Load())
                log.Printf("GeoIP: using IPv4 City DB as fallback for IPv6")
            }
            if m.asn4.Load() == nil && m.asn6.Load() != nil {
                m.asn4.Store(m.asn6.Load())
                log.Printf("GeoIP: using IPv6 ASN DB as fallback for IPv4")
            }
            if m.asn6.Load() == nil && m.asn4.Load() != nil {
                m.asn6.Store(m.asn4.Load())
                log.Printf("GeoIP: using IPv4 ASN DB as fallback for IPv6")
            }
            // if none loaded, error
            if m.city4.Load() == nil && m.city6.Load() == nil && m.asn4.Load() == nil && m.asn6.Load() == nil {
                return errors.New("no geoip databases loaded")
            }
        } else {
            r, err := geoip2.Open(path)
            if err != nil { return fmt.Errorf("open %s: %w", path, err) }
            // Use as both city4 and city6
            m.city4.Store(r)
            m.city6.Store(r)
            log.Printf("GeoIP: loaded City DB %s for IPv4/IPv6", path)
        }
        return nil
    }

    if err := load(); err != nil {
        // degrade to noop if cannot load but return error for logging upstream
        return NewNoop(), func() {}, err
    }
    stop := make(chan struct{})
    go func() {
        if reload <= 0 { return }
        ticker := time.NewTicker(reload)
        defer ticker.Stop()
        for {
            select {
            case <-ticker.C:
                _ = load()
            case <-stop:
                // best-effort close handled on next load call; nothing to do
                return
            }
        }
    }()
    return m, func() { close(stop) }, nil
}

func (m *maxmind) readerFor(ip netip.Addr, which string) *geoip2.Reader {
    v6 := ip.Is6()
    switch which {
    case "city":
        if v6 {
            if r, _ := m.city6.Load().(*geoip2.Reader); r != nil { return r }
            if r, _ := m.city4.Load().(*geoip2.Reader); r != nil { return r }
        } else {
            if r, _ := m.city4.Load().(*geoip2.Reader); r != nil { return r }
            if r, _ := m.city6.Load().(*geoip2.Reader); r != nil { return r }
        }
    case "asn":
        if v6 {
            if r, _ := m.asn6.Load().(*geoip2.Reader); r != nil { return r }
            if r, _ := m.asn4.Load().(*geoip2.Reader); r != nil { return r }
        } else {
            if r, _ := m.asn4.Load().(*geoip2.Reader); r != nil { return r }
            if r, _ := m.asn6.Load().(*geoip2.Reader); r != nil { return r }
        }
    }
    return nil
}

func (m *maxmind) Lookup(ip netip.Addr) Info {
    var info Info
    nip := ip.AsSlice()
    if r := m.readerFor(ip, "city"); r != nil {
        if rec, err := r.City(nip); err == nil && rec != nil {
            info.Country = rec.Country.IsoCode
            info.Continent = rec.Continent.Code
        }
    }
    if r := m.readerFor(ip, "asn"); r != nil {
        if rec, err := r.ASN(nip); err == nil && rec != nil {
            info.ASN = int(rec.AutonomousSystemNumber)
        }
    }
    return info
}
