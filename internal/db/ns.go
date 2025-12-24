package db

import (
	"strings"

	"gorm.io/gorm"
)

// resolveNSName resolves NS server name with {zone} placeholder.
func resolveNSName(input, zone string) string {
	v := strings.TrimSpace(input)
	if v == "" {
		return ""
	}
	z := strings.TrimSuffix(strings.ToLower(zone), ".")
	v = strings.ReplaceAll(v, "{zone}", z)
	v = strings.ToLower(strings.TrimSpace(v))
	if !strings.HasSuffix(v, ".") {
		v += "."
	}
	return v
}

// EnsureDefaultNS creates default NS records for a zone if none exist.
// servers is a list of NS server names (can include {zone} placeholder).
// Returns true if NS records were created, false if they already existed.
func EnsureDefaultNS(db *gorm.DB, zone Zone, servers []string, ttl uint32) bool {
	if len(servers) == 0 {
		return false
	}

	// Check if NS records already exist for this zone
	var count int64
	db.Model(&RRSet{}).Where("zone_id = ? AND type = ?", zone.ID, "NS").Count(&count)
	if count > 0 {
		return false // NS records already exist
	}

	// Create NS RRSet with all servers
	zname := strings.TrimSuffix(strings.ToLower(zone.Name), ".")
	origin := zname + "."

	if ttl == 0 {
		ttl = 86400 // Default TTL: 24 hours
	}

	records := make([]RData, 0, len(servers))
	for _, srv := range servers {
		nsName := resolveNSName(srv, zname)
		if nsName != "" {
			records = append(records, RData{Data: nsName})
		}
	}

	if len(records) == 0 {
		return false
	}

	rrset := RRSet{
		ZoneID:  zone.ID,
		Name:    origin,
		Type:    "NS",
		TTL:     ttl,
		Records: records,
	}

	if err := db.Create(&rrset).Error; err != nil {
		return false
	}

	return true
}
