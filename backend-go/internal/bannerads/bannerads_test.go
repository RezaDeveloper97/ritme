package bannerads

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTargets(t *testing.T) {
	assert.True(t, Targets(sql.NullString{}, "cycle"), "NULL = every mode")
	assert.True(t, Targets(sql.NullString{String: "[]", Valid: true}, ""), "[] = every mode")
	m := sql.NullString{String: `["pregnancy","ttc"]`, Valid: true}
	assert.True(t, Targets(m, "pregnancy"))
	assert.False(t, Targets(m, "cycle"))
	assert.False(t, Targets(m, ""), "unknown mode only matches untargeted ads")
}

func TestSlots(t *testing.T) {
	assert.True(t, IsSlot("ad_strip"))
	assert.False(t, IsSlot("home_top"))
}
