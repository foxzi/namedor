package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "smaillgeodns/internal/config"
    "smaillgeodns/internal/db"
    dnssrv "smaillgeodns/internal/server/dns"
    restsrv "smaillgeodns/internal/server/rest"
)

func main() {
    cfgPath := os.Getenv("SGDNS_CONFIG")
    if cfgPath == "" {
        cfgPath = "config.yaml"
    }

    cfg, err := config.Load(cfgPath)
    if err != nil {
        log.Fatalf("load config: %v", err)
    }

    gormDB, err := db.Open(cfg.DB)
    if err != nil {
        log.Fatalf("open db: %v", err)
    }
    if err := db.AutoMigrate(gormDB); err != nil {
        log.Fatalf("migrate db: %v", err)
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    dnsServer, err := dnssrv.NewServer(cfg, gormDB)
    if err != nil {
        log.Fatalf("dns server: %v", err)
    }

    restServer := restsrv.NewServer(cfg, gormDB)

    go func() {
        if err := dnsServer.Start(); err != nil {
            log.Fatalf("dns start: %v", err)
        }
    }()

    go func() {
        if err := restServer.Start(); err != nil {
            log.Fatalf("rest start: %v", err)
        }
    }()

    // Graceful shutdown
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh
    log.Println("Shutting down...")

    shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 5*time.Second)
    defer shutdownCancel()

    _ = restServer.Shutdown(shutdownCtx)
    _ = dnsServer.Shutdown()
}

