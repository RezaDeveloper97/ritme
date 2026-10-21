package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryName(t *testing.T) {
	for query, want := range map[string]string{
		"-- name: GetPeriod :one\nSELECT 1":         "GetPeriod",
		"\n  -- name: ListLogs :many\nSELECT 1":     "ListLogs",
		"-- name: DeleteEstimatesExcept :exec\r\nx": "DeleteEstimatesExcept",
		"-- name:\nSELECT 1":                        QueryRaw,
		"-- name: \nSELECT 1":                       QueryRaw,
		"SELECT 1":                                  QueryRaw,
		"/* name: X */ SELECT 1":                    QueryRaw,
		"":                                          QueryRaw,
	} {
		assert.Equal(t, want, QueryName(query), "%q", query)
	}
}
