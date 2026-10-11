package order

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCoverPath(t *testing.T) {
	ns := func(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }
	assert.Equal(t, "shop/a.webp", coverPath(ns("shop/a.webp"), ns(`["shop/b.webp"]`)), "legacy image_path wins")
	assert.Equal(t, "shop/b.webp", coverPath(ns(""), ns(`["shop/b.webp","shop/c.webp"]`)), "commerce products only have the array")
	assert.Equal(t, "", coverPath(ns(""), ns(`[]`)))
	assert.Equal(t, "", coverPath(ns(""), ns(`not json`)))
	assert.Equal(t, "", coverPath(ns(""), ns("")))
}
