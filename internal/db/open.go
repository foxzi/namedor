package db

import (
    "fmt"

    "gorm.io/driver/mysql"
    "gorm.io/driver/postgres"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"

    "namedot/internal/config"
)

func Open(cfg config.DBConfig) (*gorm.DB, error) {
    return OpenWithDebug(cfg, false)
}

func OpenWithDebug(cfg config.DBConfig, debug bool) (*gorm.DB, error) {
    // Configure GORM logger based on debug flag
    var logLevel logger.LogLevel
    if debug {
        logLevel = logger.Info // Show all queries including successful ones
    } else {
        logLevel = logger.Silent // Don't log anything in production
    }

    gormCfg := &gorm.Config{
        Logger: logger.Default.LogMode(logLevel),
    }

    switch cfg.Driver {
    case "postgres", "postgresql":
        return gorm.Open(postgres.Open(cfg.DSN), gormCfg)
    case "mysql":
        return gorm.Open(mysql.Open(cfg.DSN), gormCfg)
    case "sqlite", "sqlite3", "":
        dsn := cfg.DSN
        if dsn == "" {
            dsn = "file:namedot.db?_foreign_keys=on"
        }
        return gorm.Open(sqlite.Open(dsn), gormCfg)
    default:
        return nil, fmt.Errorf("unsupported db driver: %s", cfg.Driver)
    }
}

func AutoMigrate(db *gorm.DB) error {
    return db.AutoMigrate(&Zone{}, &RRSet{}, &RData{}, &Template{}, &TemplateRecord{})
}

