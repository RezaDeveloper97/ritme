//go:build race

package racedet

// Enabled reports whether the binary was built with -race (timing budgets are meaningless then: 5–10× slower).
const Enabled = true
