package configcache

import "time"

// NewHubAt is a hub with its own TTL and clock (tests).
func NewHubAt(ttl time.Duration, now func() time.Time) *Hub {
	h := newHub(ttl)
	h.now = now
	return h
}
