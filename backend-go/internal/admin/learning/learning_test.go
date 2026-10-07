package learning

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReviewState(t *testing.T) {
	at := time.Date(2026, 10, 7, 10, 0, 0, 0, time.UTC)
	reviewed := sql.NullTime{Time: at, Valid: true}
	same := sql.NullTime{Time: at, Valid: true}
	later := sql.NullTime{Time: at.Add(time.Second), Valid: true}

	assert.Equal(t, ReviewPending, ReviewState("pending", sql.NullTime{}, later))
	assert.Equal(t, ReviewApproved, ReviewState("approved", reviewed, same), "edited in the same second counts as reviewed")
	assert.Equal(t, ReviewChanged, ReviewState("approved", reviewed, later))
	assert.Equal(t, ReviewChanged, ReviewState("approved", sql.NullTime{}, later), "approved without a date is re-reviewed")
	assert.Equal(t, ReviewFlagged, ReviewState("flagged", reviewed, later))
	assert.Equal(t, ReviewPending, ReviewState("bogus", reviewed, same))
}

func TestAggregate(t *testing.T) {
	pairs := []pairKey{{1, 10}, {1, 11}, {2, 10}}
	sums := map[pairKey]int64{{1, 10}: 200, {1, 11}: 50, {2, 10}: 0}
	lessons := map[uint64]int64{1: 2, 2: 0}
	per, total := aggregate(pairs, sums, lessons)
	assert.Equal(t, 2, per[1].Students)
	assert.Equal(t, 63, per[1].Percent())
	assert.Equal(t, 1, per[1].Completed)
	assert.Equal(t, 0, per[2].Percent(), "a course without published lessons is 0 %")
	assert.Equal(t, 3, total.Students)
	assert.Equal(t, 42, total.Percent())
	assert.Equal(t, 0, usage{}.Percent())
}
