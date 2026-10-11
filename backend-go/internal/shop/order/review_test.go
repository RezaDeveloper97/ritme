package order

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthorName_InitialOnly(t *testing.T) {
	for _, tc := range []struct {
		name sql.NullString
		want string
	}{
		{sql.NullString{String: "sara karimi", Valid: true}, "S."},
		{sql.NullString{String: "  سارا", Valid: true}, "س."},
		{sql.NullString{String: "", Valid: true}, "Ritme user"},
		{sql.NullString{}, "Ritme user"},
		{sql.NullString{String: "0912", Valid: true}, "Ritme user"},
	} {
		assert.Equal(t, tc.want, authorName(tc.name, "Ritme user"), tc.name.String)
	}
}

func TestReviewCopy_EveryLanguage(t *testing.T) {
	for _, l := range []string{"fa", "en"} {
		for _, k := range []string{"errors.not_reviewable", "errors.already_reviewed", "messages.review_submitted", "labels.review_author"} {
			assert.NotEqual(t, "shop_order."+k, T(k, l), l+" "+k)
		}
	}
}
