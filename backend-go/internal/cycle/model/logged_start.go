package model

// SourceUserLogged is the `source` of a row the bleeding log creates (DataSource::USER_LOGGED).
const SourceUserLogged = "user_logged"

// PromoteLoggedStart (B-N3-14b, N3 stage smoke B-2, deviations.md D-55): the period-start reconciliation
// of a bleeding log (internal/healthlog) inserts an unconfirmed `user_logged` row. Only confirmed rows
// anchor the v1.1 engine, so a late user who logs bleeding today kept «تأخیر پریود ۹ روز» on the home hero
// while the calendar and the legacy calculation (profile LMP) already said «روز ۱». When the newest row of
// the history is such a row, it counts as confirmed for the engine; older unconfirmed rows (back-dated
// days between two periods) stay out of the anchors and medians. histories is changed in place.
func PromoteLoggedStart(histories []History) {
	var newest *History
	for i := range histories {
		if h := &histories[i]; newest == nil || h.PeriodStart.After(newest.PeriodStart) {
			newest = h
		}
	}
	if newest != nil && !newest.IsConfirmed && !newest.IsEstimated && newest.Source == SourceUserLogged {
		newest.IsConfirmed = true
	}
}
