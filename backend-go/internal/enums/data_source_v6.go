package enums

// Go-only DataSource cases (engine v6). They are not in backend/app/Enums/DataSource.php yet (D-138: add them there
// before any cutover, T-M9-28), so they stay out of the generated case list, DataSourceCases and the PHP parity test.

// DataSourceBleedingLog is the `source` of a period the bleeding-log bridge builds from `flow` logs (DECISIONS.md
// I-02). Actual data, like DataSourceUserLogged.
const DataSourceBleedingLog DataSource = "bleeding_log"

// IsKnown reports a PHP case or a Go-only engine v6 case. cycle_histories.source is read as a plain string
// everywhere, so an unknown value never fails a read; IsKnown is for code that has to tell them apart.
func (e DataSource) IsKnown() bool {
	return e.IsValid() || e == DataSourceBleedingLog
}

// ParseDataSource is DataSourceFrom plus the Go-only cases: the case backed by v and whether it is known.
func ParseDataSource(v string) (DataSource, bool) {
	e := DataSource(v)
	return e, e.IsKnown()
}
