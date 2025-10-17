package dns

import (
	"sync"
	"time"

	dbm "namedot/internal/db"
)

// ZoneCache provides an in-memory cache of zones sorted by name length.
// This avoids expensive database queries on every DNS lookup.
type ZoneCache struct {
	mu        sync.RWMutex
	zones     []dbm.Zone
	lastFetch time.Time
	ttl       time.Duration
}

// NewZoneCache creates a new zone cache with given TTL
func NewZoneCache(ttl time.Duration) *ZoneCache {
	return &ZoneCache{
		ttl: ttl,
	}
}

// Get returns the cached zones if they're still valid, or empty slice if expired
func (zc *ZoneCache) Get() []dbm.Zone {
	zc.mu.RLock()
	defer zc.mu.RUnlock()

	if time.Since(zc.lastFetch) < zc.ttl && len(zc.zones) > 0 {
		// Return a copy to prevent external mutations
		cp := make([]dbm.Zone, len(zc.zones))
		copy(cp, zc.zones)
		return cp
	}
	return nil
}

// Set updates the cache with new zones
func (zc *ZoneCache) Set(zones []dbm.Zone) {
	zc.mu.Lock()
	defer zc.mu.Unlock()

	// Make a copy to prevent external mutations
	cp := make([]dbm.Zone, len(zones))
	copy(cp, zones)
	zc.zones = cp
	zc.lastFetch = time.Now()
}

// Invalidate clears the cache, forcing a refresh on next Get
func (zc *ZoneCache) Invalidate() {
	zc.mu.Lock()
	defer zc.mu.Unlock()

	zc.zones = nil
	zc.lastFetch = time.Time{}
}

// IsExpired returns true if cache is expired and needs refresh
func (zc *ZoneCache) IsExpired() bool {
	zc.mu.RLock()
	defer zc.mu.RUnlock()

	return len(zc.zones) == 0 || time.Since(zc.lastFetch) >= zc.ttl
}
