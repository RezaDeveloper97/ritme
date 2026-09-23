package lang

import "time"

// SetCheckInterval lets tests see storage changes immediately; it returns a restore func.
func SetCheckInterval(d time.Duration) func() {
	old := checkInterval
	checkInterval = d
	return func() { checkInterval = old }
}
