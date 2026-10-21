package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckDSN(t *testing.T) {
	ok := []string{
		"ritme:ritme@tcp(127.0.0.1:13317)/ritme_loadtest?parseTime=true",
		"u:p@tcp(localhost:3306)/ritme_local",
	}
	for _, d := range ok {
		_, err := CheckDSN(d)
		assert.NoError(t, err, d)
	}
	bad := []string{
		"u:p@tcp(89.251.8.115:3306)/ritme_loadtest",
		"u:p@tcp(db.internal:3306)/ritme_test_x",
		"u:p@tcp(127.0.0.1:3306)/ritme",
		"u:p@tcp(127.0.0.1:3306)/ritme_stage",
		"u:p@tcp(127.0.0.1:3306)/ritme_test",
		"u:p@tcp(127.0.0.1:3306)/gt_1792_users_ab",
		"not a dsn",
	}
	for _, d := range bad {
		_, err := CheckDSN(d)
		require.Error(t, err, d)
	}
}
