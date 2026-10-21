package passport

import "time"

// SetClientClock replaces the wall clock of v's client cache (tests drive the 5-minute refresh).
func SetClientClock(v *Verifier, now func() time.Time) { v.clients.now = now }

// WaitClientRefresh blocks until a background client refresh started by v has finished.
func WaitClientRefresh(v *Verifier) {
	for v.clients.refreshing.Load() {
		time.Sleep(time.Millisecond)
	}
}
