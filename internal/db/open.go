package db

import (
    "fmt"

    "gorm.io/driver/mysql"
    "gorm.io/driver/postgres"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"

    "smaillgeodns/internal/config"
)

func Open(cfg config.DBConfig) (*gorm.DB, error) {
    switch cfg.Driver {
    case "postgres", "postgresql":
        return gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{})
    case "mysql":
        return gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
    case "sqlite", "sqlite3", "":
        dsn := cfg.DSN
        if dsn == "" {
            dsn = "file:smaillgeodns.db?_foreign_keys=on"
        }
        return gorm.Open(sqlite.Open(dsn), &gorm.Config{})
    default:
        return nil, fmt.Errorf("unsupported db driver: %s", cfg.Driver)
    }
}

func AutoMigrate(db *gorm.DB) error {
    return db.AutoMigrate(&Zone{}, &RRSet{}, &RData{}, &Template{}, &TemplateRecord{})
}

