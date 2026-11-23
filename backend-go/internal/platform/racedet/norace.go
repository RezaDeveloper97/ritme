//go:build !race

// Package racedet tells tests whether the race detector is on, so latency budgets can skip under -race.
package racedet

// Enabled reports whether the binary was built with -race.
const Enabled = false
