package db

import (
    "time"

    "gorm.io/gorm"
)

type Zone struct {
    ID        uint           `gorm:"primaryKey" json:"id"`
    Name      string         `gorm:"uniqueIndex;size:255" json:"name"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
    RRSets    []RRSet        `json:"rrsets"`
}

type RRSet struct {
    ID        uint           `gorm:"primaryKey" json:"id"`
    ZoneID    uint           `gorm:"index" json:"zone_id"`
    Name      string         `gorm:"index;size:255" json:"name"`
    Type      string         `gorm:"index;size:20" json:"type"`
    TTL       uint32         `json:"ttl"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
    Records   []RData        `json:"records"`
}

type RData struct {
    ID        uint           `gorm:"primaryKey" json:"id"`
    RRSetID   uint           `gorm:"index" json:"rrset_id"`
    Data      string         `gorm:"type:text" json:"data"`
    Country   *string        `gorm:"size:2" json:"country,omitempty"`
    Continent *string        `gorm:"size:2" json:"continent,omitempty"`
    ASN       *int           `json:"asn,omitempty"`
    Subnet    *string        `gorm:"size:64" json:"subnet,omitempty"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Template represents a DNS record template
type Template struct {
    ID          uint             `gorm:"primaryKey" json:"id"`
    Name        string           `gorm:"size:100;not null" json:"name"`
    Description string           `gorm:"type:text" json:"description"`
    CreatedAt   time.Time        `json:"created_at"`
    UpdatedAt   time.Time        `json:"updated_at"`
    DeletedAt   gorm.DeletedAt   `gorm:"index" json:"-"`
    Records     []TemplateRecord `json:"records"`
}

// TemplateRecord represents a DNS record within a template
type TemplateRecord struct {
    ID          uint           `gorm:"primaryKey" json:"id"`
    TemplateID  uint           `gorm:"index;not null" json:"template_id"`
    Name        string         `gorm:"size:255;not null" json:"name"` // Can use placeholders like {domain}
    Type        string         `gorm:"size:20;not null" json:"type"`
    TTL         uint32         `json:"ttl"`
    Data        string         `gorm:"type:text;not null" json:"data"` // Can use placeholders
    Country     *string        `gorm:"size:2" json:"country,omitempty"`
    Continent   *string        `gorm:"size:2" json:"continent,omitempty"`
    ASN         *int           `json:"asn,omitempty"`
    Subnet      *string        `gorm:"size:64" json:"subnet,omitempty"`
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
    DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

