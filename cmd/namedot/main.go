package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "os"
    "os/signal"
    "strings"
    "syscall"
    "time"

    "golang.org/x/crypto/bcrypt"

    "namedot/internal/config"
    "namedot/internal/db"
    "namedot/internal/replication"
    dnssrv "namedot/internal/server/dns"
    restsrv "namedot/internal/server/rest"
)

// Version is set via -ldflags "-X main.Version=<version>" during build.
var Version = "dev"

func main() {
    // Normalize GNU-style flags ("--flag") to Go's default ("-flag")
    // to support both -c/--config, -t/--test, -p/--password without extra deps.
    if len(os.Args) > 1 {
        norm := make([]string, 0, len(os.Args))
        norm = append(norm, os.Args[0])
        for i := 1; i < len(os.Args); i++ {
            a := os.Args[i]
            if a == "--" { // end of flags
                norm = append(norm, a)
                // append the rest as-is
                norm = append(norm, os.Args[i+1:]...)
                break
            }
            if strings.HasPrefix(a, "--") {
                a = "-" + strings.TrimPrefix(a, "--")
            }
            norm = append(norm, a)
        }
        os.Args = norm
    }

    var (
        cfgPath  string
        testOnly bool
        password string
        token    string
        showVer  bool
    )

    // Support both short and long variants by binding to the same var
    flag.StringVar(&cfgPath, "c", "", "path to config file (yaml)")
    flag.StringVar(&cfgPath, "config", "", "path to config file (yaml)")
    flag.BoolVar(&testOnly, "t", false, "validate config and exit")
    flag.BoolVar(&testOnly, "test", false, "validate config and exit")
    flag.StringVar(&password, "p", "", "generate bcrypt hash for admin password and exit")
    flag.StringVar(&password, "password", "", "generate bcrypt hash for admin password and exit")
    flag.StringVar(&token, "g", "", "generate bcrypt hash for api token and exit")
    flag.StringVar(&token, "gen-token", "", "generate bcrypt hash for api token and exit")
    flag.BoolVar(&showVer, "v", false, "print version and exit")
    flag.BoolVar(&showVer, "version", false, "print version and exit")
    flag.Parse()

    if showVer {
        fmt.Println(Version)
        return
    }

    // If password flag provided, generate bcrypt and exit
    if password != "" {
        hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
        if err != nil {
            log.Fatalf("error generating bcrypt: %v", err)
        }
        fmt.Printf("Bcrypt hash for '%s':\n%s\n", password, string(hash))
        fmt.Println("\nAdd this to your config.yaml:")
        fmt.Println("admin:")
        fmt.Println("  enabled: true")
        fmt.Println("  username: admin")
        fmt.Printf("  password_hash: \"%s\"\n", string(hash))
        return
    }

    // If token flag provided, generate bcrypt and exit
    if token != "" {
        hash, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
        if err != nil {
            log.Fatalf("error generating bcrypt: %v", err)
        }
        fmt.Printf("Bcrypt hash for API token '%s':\n%s\n", token, string(hash))
        fmt.Println("\nAdd this to your config.yaml:")
        fmt.Printf("api_token_hash: \"%s\"\n", string(hash))
        fmt.Println("\nFor replication slave config:")
        fmt.Println("replication:")
        fmt.Println("  mode: slave")
        fmt.Println("  master_url: \"http://master:8080\"")
        fmt.Printf("  api_token: \"%s\"  # Use plain token for outgoing requests\n", token)
        return
    }

    // Determine config path precedence: -c/--config > env > default
    if cfgPath == "" {
        cfgPath = os.Getenv("SGDNS_CONFIG")
    }
    if cfgPath == "" {
        cfgPath = "config.yaml"
    }

    cfg, err := config.Load(cfgPath)
    if err != nil {
        log.Fatalf("load config: %v", err)
    }

    if testOnly {
        fmt.Printf("Config OK: %s\n", cfgPath)
        return
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

    // Start replication sync worker for slave mode
    if cfg.Replication.Mode == "slave" {
        syncClient := replication.NewSyncClient(cfg, gormDB)
        go func() {
            // Wait a bit for REST server to start
            time.Sleep(2 * time.Second)
            syncClient.StartPeriodicSync(ctx)
        }()
        log.Printf("Slave mode enabled: syncing from %s every %d seconds",
            cfg.Replication.MasterURL, cfg.Replication.SyncIntervalSec)
    } else if cfg.Replication.Mode == "master" {
        log.Println("Master mode enabled: ready to serve replication data")
    }

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
