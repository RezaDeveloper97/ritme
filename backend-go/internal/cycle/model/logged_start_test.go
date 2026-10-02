package model

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

func TestPromoteLoggedStart(t *testing.T) {
	d := civildate.MustParse
	confirmed := History{ID: 1, PeriodStart: d("2026-08-16"), PeriodEnd: d("2026-08-20"), IsConfirmed: true, Source: SourceUserLogged}
	logged := History{ID: 2, PeriodStart: d("2026-09-23"), Source: SourceUserLogged}
	backDated := History{ID: 3, PeriodStart: d("2026-09-01"), PeriodEnd: d("2026-09-01"), Source: SourceUserLogged}

	h := []History{logged, confirmed}
	PromoteLoggedStart(h)
	assert.True(t, h[0].IsConfirmed, "the newest logged start anchors the engine")

	h = []History{confirmed, backDated, {ID: 4, PeriodStart: d("2026-09-14"), IsConfirmed: true, Source: SourceUserLogged}}
	PromoteLoggedStart(h)
	assert.False(t, h[1].IsConfirmed, "an older unconfirmed row stays unconfirmed")

	h = []History{confirmed, {ID: 5, PeriodStart: d("2026-09-20"), IsEstimated: true, Source: "onboarding_estimate"}}
	PromoteLoggedStart(h)
	assert.False(t, h[1].IsConfirmed, "an onboarding estimate is not promoted")

	PromoteLoggedStart(nil) // no rows: no panic
}
