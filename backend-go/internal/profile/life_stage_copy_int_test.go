package profile_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// B-N2-03: the calm pregnancy exit reads its admin-edited copy (pregnancy_setup / loss_exit); every text is null
// until an admin writes one, inactive or unapproved rows are ignored, and a guest gets the 401 contract.
func TestLifeStage_LossCopy(t *testing.T) {
	e := setup(t)
	_, tok := e.freshUser(t, "09120000231")

	nulls := map[string]any{
		"title": nil, "body": nil, "confirm": nil, "cancel": nil,
		"done_title": nil, "done_body": nil, "done_action": nil,
	}
	assert.Equal(t, nulls, obData(t, e.do(t, "GET", "/api/v1/profile/life-stage/loss-copy", tok, "")))

	// An unapproved draft is not served.
	_, err := e.db.Exec(`INSERT INTO message_contents (` + "`group`" + `, item_key, locale, label, payload, is_active, is_approved, sort_order, created_at, updated_at)
		VALUES ('pregnancy_setup', 'loss_exit', 'fa', 'draft', '{"title":"draft"}', 1, 0, 0, NOW(), NOW())`)
	require.NoError(t, err)
	assert.Equal(t, nulls, obData(t, e.do(t, "GET", "/api/v1/profile/life-stage/loss-copy", tok, "")))

	_, err = e.db.Exec(`UPDATE message_contents SET is_approved = 1, payload = '{"title":"کنارت هستیم","body":"","confirm":"پایان حالت بارداری","extra":"x"}'
		WHERE ` + "`group`" + ` = 'pregnancy_setup' AND item_key = 'loss_exit'`)
	require.NoError(t, err)
	want := map[string]any{
		"title": "کنارت هستیم", "body": nil, "confirm": "پایان حالت بارداری", "cancel": nil,
		"done_title": nil, "done_body": nil, "done_action": nil,
	}
	assert.Equal(t, want, obData(t, e.do(t, "GET", "/api/v1/profile/life-stage/loss-copy", tok, "")))

	r := e.do(t, "GET", "/api/v1/profile/life-stage/loss-copy", "nope", "")
	assert.Equal(t, 401, r.status)
	assert.Equal(t, "unauthenticated", r.body["error_code"])
}
