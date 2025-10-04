package zoneio

import (
    "strings"

    "gorm.io/gorm"

    dbm "smaillgeodns/internal/db"
)

// ImportJSON imports RRsets from src into dst zone.
// mode: upsert | replace
func ImportJSON(db *gorm.DB, dst *dbm.Zone, src *dbm.Zone, mode string, defaultTTL uint32) error {
    return db.Transaction(func(tx *gorm.DB) error {
        if mode == "replace" {
            if err := tx.Where("zone_id = ?", dst.ID).Delete(&dbm.RRSet{}).Error; err != nil {
                return err
            }
        }
        for _, rs := range src.RRSets {
            rs.ZoneID = dst.ID
            rs.Name = strings.ToLower(rs.Name)
            rs.Type = strings.ToUpper(rs.Type)
            if rs.TTL == 0 && defaultTTL > 0 {
                rs.TTL = defaultTTL
            }
            // Upsert by name+type
            var existing dbm.RRSet
            if err := tx.Where("zone_id = ? AND name = ? AND type = ?", dst.ID, rs.Name, rs.Type).First(&existing).Error; err == nil {
                // replace records
                if err := tx.Where("rr_set_id = ?", existing.ID).Delete(&dbm.RData{}).Error; err != nil {
                    return err
                }
                existing.TTL = rs.TTL
                existing.Records = rs.Records
                if err := tx.Save(&existing).Error; err != nil {
                    return err
                }
            } else {
                if err := tx.Create(&rs).Error; err != nil {
                    return err
                }
            }
        }
        return nil
    })
}
