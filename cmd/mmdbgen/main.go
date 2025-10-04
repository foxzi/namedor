package main

import (
    "flag"
    "fmt"
    "net"
    "os"

    "github.com/maxmind/mmdbwriter"
    "github.com/maxmind/mmdbwriter/mmdbtype"
    "gopkg.in/yaml.v3"
)

type CityEntry struct {
    CIDR      string `yaml:"cidr"`
    Country   string `yaml:"country"`
    Continent string `yaml:"continent"`
}

type ASNEntry struct {
    CIDR string `yaml:"cidr"`
    ASN  uint32 `yaml:"asn"`
    Org  string `yaml:"org"`
}

type Spec struct {
    City []CityEntry `yaml:"city"`
    ASN  []ASNEntry  `yaml:"asn"`
}

func loadSpec(path string) (*Spec, error) {
    b, err := os.ReadFile(path)
    if err != nil { return nil, err }
    var s Spec
    if err := yaml.Unmarshal(b, &s); err != nil { return nil, err }
    return &s, nil
}

func writeCity(out string, entries []CityEntry) error {
    if out == "" || len(entries) == 0 { return nil }
    w, err := mmdbwriter.New(mmdbwriter.Options{DatabaseType: "GeoLite2-City"})
    if err != nil { return err }
    for _, e := range entries {
        _, nw, err := net.ParseCIDR(e.CIDR)
        if err != nil { return fmt.Errorf("city %s: %w", e.CIDR, err) }
        data := mmdbtype.Map{
            "country":   mmdbtype.Map{"iso_code": mmdbtype.String(e.Country)},
            "continent": mmdbtype.Map{"code": mmdbtype.String(e.Continent)},
        }
        if err := w.Insert(nw, data); err != nil {
            return fmt.Errorf("city insert %s: %w", e.CIDR, err)
        }
    }
    f, err := os.Create(out)
    if err != nil { return err }
    defer f.Close()
    _, err = w.WriteTo(f)
    return err
}

func writeASN(out string, entries []ASNEntry) error {
    if out == "" || len(entries) == 0 { return nil }
    w, err := mmdbwriter.New(mmdbwriter.Options{DatabaseType: "GeoLite2-ASN"})
    if err != nil { return err }
    for _, e := range entries {
        _, nw, err := net.ParseCIDR(e.CIDR)
        if err != nil { return fmt.Errorf("asn %s: %w", e.CIDR, err) }
        data := mmdbtype.Map{
            "autonomous_system_number":       mmdbtype.Uint32(e.ASN),
            "autonomous_system_organization": mmdbtype.String(e.Org),
        }
        if err := w.Insert(nw, data); err != nil {
            return fmt.Errorf("asn insert %s: %w", e.CIDR, err)
        }
    }
    f, err := os.Create(out)
    if err != nil { return err }
    defer f.Close()
    _, err = w.WriteTo(f)
    return err
}

func main() {
    in := flag.String("in", "", "YAML spec with city/asn entries")
    cityOut := flag.String("city-out", "", "Output path for City MMDB (optional)")
    asnOut := flag.String("asn-out", "", "Output path for ASN MMDB (optional)")
    flag.Parse()

    if *in == "" {
        fmt.Fprintln(os.Stderr, "-in spec.yaml is required")
        os.Exit(2)
    }
    spec, err := loadSpec(*in)
    if err != nil {
        fmt.Fprintln(os.Stderr, "load spec:", err)
        os.Exit(1)
    }
    if *cityOut == "" && *asnOut == "" {
        fmt.Fprintln(os.Stderr, "provide at least one of -city-out or -asn-out")
        os.Exit(2)
    }
    if err := writeCity(*cityOut, spec.City); err != nil {
        // mmdbwriter may reject reserved networks; hint about using TEST-NET ranges
        fmt.Fprintln(os.Stderr, "write city:", err)
        fmt.Fprintln(os.Stderr, "hint: use RFC 5737 ranges like 192.0.2.0/24, 198.51.100.0/24, 203.0.113.0/24 for tests")
        os.Exit(1)
    }
    if err := writeASN(*asnOut, spec.ASN); err != nil {
        fmt.Fprintln(os.Stderr, "write asn:", err)
        fmt.Fprintln(os.Stderr, "hint: use RFC 5737 ranges like 192.0.2.0/24, 198.51.100.0/24, 203.0.113.0/24 for tests")
        os.Exit(1)
    }
}
