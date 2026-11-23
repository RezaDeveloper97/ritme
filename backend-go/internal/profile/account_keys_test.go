package profile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Every Redis key family that carries one user's data must be purged on account deletion (T-M12-13, M12 re-audit:
// the analysis cache, the engine-input snapshot and the pregnancy-alert digest were missing).
func TestUserKeyPatterns_CoverPerUserCaches(t *testing.T) {
	got := UserKeyPatterns(42)
	for _, want := range []string{
		"cycle-engine:42:*",
		"eng:in:42:*",
		"ana:42:*",
		"preg-alerts:eval:42",
		"idem:42:*",
		"family:ra:42:*",
		"family:ra:*:42:*",
	} {
		assert.Contains(t, got, want)
	}
}

// Patterns are scoped to the one user: no pattern may match another id (4 vs 42, 420).
func TestUserKeyPatterns_ScopedToOneUser(t *testing.T) {
	for _, p := range UserKeyPatterns(4) {
		assert.NotContains(t, p, ":42", p)
		assert.NotRegexp(t, `(^|:)\*:`+"$", p) // never a bare wildcard in the id slot
		assert.Regexp(t, `(^|:)4(:|$)`, p)
	}
}
