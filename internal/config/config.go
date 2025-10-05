package config

import (
    "fmt"
    "os"

    "gopkg.in/yaml.v3"
)

type DBConfig struct {
    Driver string `yaml:"driver"`
    DSN    string `yaml:"dsn"`
}

type GeoIPConfig struct {
    Enabled   bool   `yaml:"enabled"`
    MMDBPath  string `yaml:"mmdb_path"`
    ReloadSec int    `yaml:"reload_sec"`
    UseECS    bool   `yaml:"use_ecs"`
}

type LogConfig struct {
    DNSVerbose bool `yaml:"dns_verbose"`
}

type UpdateConfig struct {
    Enabled     bool              `yaml:"enabled"`
    RequireTSIG bool              `yaml:"require_tsig"`
    TSIGSecrets map[string]string `yaml:"tsig_secrets"`
}

type Config struct {
    Listen       string     `yaml:"listen"`
    Forwarder    string     `yaml:"forwarder"`
    EnableDNSSEC bool       `yaml:"enable_dnssec"`
    APIToken     string     `yaml:"api_token"`
    RESTListen   string     `yaml:"rest_listen"`
    AutoSOAOnMissing bool   `yaml:"auto_soa_on_missing"`
    DefaultTTL   uint32     `yaml:"default_ttl"`

    DB    DBConfig    `yaml:"db"`
    GeoIP GeoIPConfig `yaml:"geoip"`
    Update UpdateConfig `yaml:"update"`
    Log   LogConfig   `yaml:"log"`
}

func Load(path string) (*Config, error) {
    b, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("read config: %w", err)
    }
    var cfg Config
    if err := yaml.Unmarshal(b, &cfg); err != nil {
        return nil, fmt.Errorf("parse yaml: %w", err)
    }
    if cfg.RESTListen == "" {
        cfg.RESTListen = ":8080"
    }
    if cfg.Listen == "" {
        cfg.Listen = ":53"
    }
    return &cfg, nil
}
