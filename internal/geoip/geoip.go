package geoip

import (
    "net/netip"
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

// MaxMind provider with hot-reload
type maxmind struct {
    dbPath string
    dbCity atomic.Value // *geoip2.Reader
    dbASN  atomic.Value // *geoip2.Reader
}

func NewMaxMind(path string, reload time.Duration) (Provider, func(), error) {
    m := &maxmind{dbPath: path}
    load := func() error {
        reader, err := geoip2.Open(path)
        if err != nil {
            return err
        }
        m.dbCity.Store(reader)
        return nil
    }
    if err := load(); err != nil {
        // return noop if cannot load
        return NewNoop(), func() {}, nil
    }
    stop := make(chan struct{})
    go func() {
        ticker := time.NewTicker(reload)
        defer ticker.Stop()
        for {
            select {
            case <-ticker.C:
                _ = load()
            case <-stop:
                if r, ok := m.dbCity.Load().(*geoip2.Reader); ok && r != nil {
                    r.Close()
                }
                return
            }
        }
    }()
    return m, func() { close(stop) }, nil
}

func (m *maxmind) Lookup(ip netip.Addr) Info {
    var info Info
    if r, ok := m.dbCity.Load().(*geoip2.Reader); ok && r != nil {
        rec, err := r.City(ip.AsSlice())
        if err == nil && rec != nil {
            if rec.Country.IsoCode != "" {
                info.Country = rec.Country.IsoCode
            }
            if rec.Continent.Code != "" {
                info.Continent = rec.Continent.Code
            }
        }
    }
    // ASN DB not wired by default to keep simple
    return info
}
