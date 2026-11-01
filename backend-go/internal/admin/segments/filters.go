// Package segments is the admin user-directory filter vocabulary and the saved segments built on it (T-M10-32, D-181;
// SPEC v6 §5.2, gap 06 §2.5): `GET /users/directory` reads Filters from the query string, `/segments` stores them by
// name (capability «segments», marketing per §5.8), and the `segment` message fact (T-M10-03) evaluates every saved
// segment against one user's projection row (admin_user_activity, nightly `admin.user_activity`).
//
// Every filter reads a code or a date of the projection or the account row — never a health value or what was logged.
package segments

import (
	"database/sql"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Filters is one directory query. "" = no filter. Q (free text over name / mobile / email, or an exact RT- code) is a
// directory search only: a saved segment cannot carry it.
type Filters struct {
	Q           string `json:"q,omitempty"`
	Status      string `json:"status,omitempty"`       // active | blocked | all
	Plan        string `json:"plan,omitempty"`         // free | trial | plus
	Activity    string `json:"activity,omitempty"`     // today | 7d | 30d | inactive (last activity, projection)
	Cohort      string `json:"cohort,omitempty"`       // YYYY-MM signup month (Tehran)
	Churn       string `json:"churn,omitempty"`        // new | low | medium | high (A-08)
	Mode        string `json:"mode,omitempty"`         // engine mode
	DataQuality string `json:"data_quality,omitempty"` // engine data_quality
	MainWarning string `json:"main_warning,omitempty"` // engine warning code
}

// Keys are the filter names in display order (query parameters and the segment JSON keys).
var Keys = []string{"q", "status", "plan", "activity", "cohort", "churn", "mode", "data_quality", "main_warning"}

// Churn buckets (A-08).
const (
	ChurnNew    = "new"
	ChurnLow    = "low"
	ChurnMedium = "medium"
	ChurnHigh   = "high"
)

var (
	cohortRe  = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)
	refCodeRe = regexp.MustCompile(`^RT-[A-Z2-7]{6,8}$`)
)

// Options are the allowed values of every enumerated filter (cohort and q are free-form).
func Options() map[string][]string {
	modes := []string{}
	for m := engine.ModeCycle; m <= engine.ModeTeen; m++ {
		modes = append(modes, m.String())
	}
	dq := []string{}
	for d := engine.DataQualityInsufficient; d <= engine.DataQualityPoor; d++ {
		dq = append(dq, d.String())
	}
	warn := []string{}
	for w := engine.WarnPeriodEndMissing; w <= engine.WarnContradictoryAnchorData; w++ {
		warn = append(warn, w.String())
	}
	return map[string][]string{
		"status":       {"active", "blocked", "all"},
		"plan":         {"free", "trial", "plus"},
		"activity":     {"today", "7d", "30d", "inactive"},
		"churn":        {ChurnNew, ChurnLow, ChurnMedium, ChurnHigh},
		"mode":         modes,
		"data_quality": dq,
		"main_warning": warn,
	}
}

// Get returns one filter by key.
func (f Filters) Get(key string) string {
	switch key {
	case "q":
		return f.Q
	case "status":
		return f.Status
	case "plan":
		return f.Plan
	case "activity":
		return f.Activity
	case "cohort":
		return f.Cohort
	case "churn":
		return f.Churn
	case "mode":
		return f.Mode
	case "data_quality":
		return f.DataQuality
	case "main_warning":
		return f.MainWarning
	}
	return ""
}

// Set sets one filter by key (false for an unknown key).
func (f *Filters) Set(key, v string) bool {
	switch key {
	case "q":
		f.Q = v
	case "status":
		f.Status = v
	case "plan":
		f.Plan = v
	case "activity":
		f.Activity = v
	case "cohort":
		f.Cohort = v
	case "churn":
		f.Churn = v
	case "mode":
		f.Mode = v
	case "data_quality":
		f.DataQuality = v
	case "main_warning":
		f.MainWarning = v
	default:
		return false
	}
	return true
}

// Invalid returns the first filter whose value is not allowed ("" = all valid).
func (f Filters) Invalid() string {
	opts := Options()
	for _, k := range Keys {
		v := f.Get(k)
		if v == "" {
			continue
		}
		switch k {
		case "q":
			if len([]rune(v)) > 100 {
				return k
			}
		case "cohort":
			if !cohortRe.MatchString(v) {
				return k
			}
		default:
			if !slices.Contains(opts[k], v) {
				return k
			}
		}
	}
	return ""
}

// Over returns f with every non-empty filter of o on top (explicit query parameters over a saved segment).
func (f Filters) Over(o Filters) Filters {
	for _, k := range Keys {
		if v := o.Get(k); v != "" {
			f.Set(k, v)
		}
	}
	return f
}

// Map is the filters as an ordered key list for JSON ("" for unset keys).
func (f Filters) Map() []any {
	out := make([]any, 0, 2*len(Keys))
	for _, k := range Keys {
		out = append(out, k, f.Get(k))
	}
	return out
}

// RefCodeQuery reports whether q is a reference code (A-09) and returns it upper-cased.
func RefCodeQuery(q string) (string, bool) {
	u := strings.ToUpper(strings.TrimSpace(q))
	return u, refCodeRe.MatchString(u)
}

var (
	nullSentinel = time.Date(1971, 1, 1, 0, 0, 0, 0, civildate.Tehran)
	maxTime      = time.Date(2037, 12, 31, 23, 59, 59, 0, civildate.Tehran) // TIMESTAMP ends 2038-01-19
)

// Bounds are the time bounds the filters resolve to at now (Tehran days), shared by the SQL and Match.
type Bounds struct {
	ActiveSince, InactiveBefore sql.NullTime
	CreatedFrom, CreatedTo      sql.NullTime
}

func valid(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: true} }

// BoundsAt resolves activity and cohort at now.
func (f Filters) BoundsAt(now time.Time) Bounds {
	today := civildate.InTehran(now)
	midnight := civildate.Date.TehranMidnight
	var b Bounds
	switch f.Activity {
	case "today":
		b.ActiveSince = valid(midnight(today))
	case "7d":
		b.ActiveSince = valid(midnight(today.AddDays(-7)))
	case "30d":
		b.ActiveSince = valid(midnight(today.AddDays(-30)))
	case "inactive":
		b.InactiveBefore = valid(midnight(today.AddDays(-30)))
	}
	if cohortRe.MatchString(f.Cohort) {
		if t, err := time.ParseInLocation("2006-01", f.Cohort, civildate.Tehran); err == nil {
			b.CreatedFrom, b.CreatedTo = valid(t), valid(t.AddDate(0, 1, 0))
		}
	}
	return b
}

func ns(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

// statusRange maps status onto the blocked_at range of the directory queries (NULL blocked_at = 1971-01-01).
func statusRange(status string) (from, to sql.NullTime) {
	switch status {
	case "active":
		return valid(nullSentinel), valid(nullSentinel)
	case "blocked":
		return valid(nullSentinel.AddDate(0, 0, 1)), valid(maxTime)
	default:
		return valid(nullSentinel), valid(maxTime)
	}
}

// escapeLike escapes LIKE wildcards so the search is a plain substring match.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// CountParams is the directory count query of f at now.
func (f Filters) CountParams(now time.Time) store.CountAdminDirectoryParams {
	b := f.BoundsAt(now)
	p := store.CountAdminDirectoryParams{
		Plan: ns(f.Plan), Churn: ns(f.Churn), Mode: ns(f.Mode), DataQuality: ns(f.DataQuality), MainWarning: ns(f.MainWarning),
		ActiveSince: b.ActiveSince, InactiveBefore: b.InactiveBefore, CreatedFrom: b.CreatedFrom, CreatedTo: b.CreatedTo,
	}
	p.BlockedFrom, p.BlockedTo = statusRange(f.Status)
	if q := strings.TrimSpace(f.Q); q != "" {
		if code, ok := RefCodeQuery(q); ok {
			p.RefCode = ns(code)
		} else {
			p.Pattern = ns("%" + escapeLike(q) + "%")
		}
	}
	return p
}

// ListParams is the directory page query of f at now.
func (f Filters) ListParams(now time.Time, limit, offset int32) store.ListAdminDirectoryParams {
	c := f.CountParams(now)
	return store.ListAdminDirectoryParams{
		Pattern: c.Pattern, RefCode: c.RefCode, BlockedFrom: c.BlockedFrom, BlockedTo: c.BlockedTo,
		CreatedFrom: c.CreatedFrom, CreatedTo: c.CreatedTo, Plan: c.Plan, Churn: c.Churn, Mode: c.Mode,
		DataQuality: c.DataQuality, MainWarning: c.MainWarning, ActiveSince: c.ActiveSince,
		InactiveBefore: c.InactiveBefore, Limit: limit, Offset: offset,
	}
}

// Subject is one user's directory row as the filters see it (Projected = false before her first nightly row).
type Subject struct {
	Blocked      bool
	CreatedAt    sql.NullTime
	Projected    bool
	LastActivity sql.NullTime
	Churn        string
	Mode         string
	Plan         string
	DataQuality  string
	MainWarning  string
}

// SubjectOf maps the GetSegmentSubject row.
func SubjectOf(r store.GetSegmentSubjectRow) Subject {
	return Subject{
		Blocked: r.BlockedAt.Valid, CreatedAt: r.CreatedAt, Projected: r.ProjectedUserID.Valid,
		LastActivity: r.LastActivityAt, Churn: r.Churn.String, Mode: r.Mode.String, Plan: r.Plan.String,
		DataQuality: r.DataQuality.String, MainWarning: r.MainWarning.String,
	}
}

// Match reports whether s passes f at now — the same predicate as the directory SQL (a projection filter never
// matches a user without a projection row; free text never matches: segments carry no q).
func (f Filters) Match(s Subject, now time.Time) bool {
	if f.Q != "" {
		return false
	}
	switch f.Status {
	case "active":
		if s.Blocked {
			return false
		}
	case "blocked":
		if !s.Blocked {
			return false
		}
	}
	b := f.BoundsAt(now)
	if b.CreatedFrom.Valid {
		if !s.CreatedAt.Valid || s.CreatedAt.Time.Before(b.CreatedFrom.Time) || !s.CreatedAt.Time.Before(b.CreatedTo.Time) {
			return false
		}
	}
	eq := func(want, got string) bool { return want == "" || (s.Projected && want == got) }
	if !eq(f.Plan, s.Plan) || !eq(f.Churn, s.Churn) || !eq(f.Mode, s.Mode) || !eq(f.DataQuality, s.DataQuality) ||
		!eq(f.MainWarning, s.MainWarning) {
		return false
	}
	if b.ActiveSince.Valid && (!s.Projected || !s.LastActivity.Valid || s.LastActivity.Time.Before(b.ActiveSince.Time)) {
		return false
	}
	if b.InactiveBefore.Valid && (!s.Projected || (s.LastActivity.Valid && !s.LastActivity.Time.Before(b.InactiveBefore.Time))) {
		return false
	}
	return true
}
