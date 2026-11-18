package pregnancyalerts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// ---- The evaluate-before-Today skip (T-M12-30c) ----
//
// GET /pregnancy/v2/today runs the daily evaluation first (EvaluateFirst). The evaluation is idempotent, so a repeat
// read may skip it when nothing it reads has changed and no time-based rule can have flipped since:
//
//   - the input revision: user_revisions (engine_rev, analysis_rev — every log, profile, vitals, weight write bumps
//     them, free with the auth query), the content stamp of the inputs no revision covers (GetV2AlertInputStamp: the
//     kick sessions, the running contraction session, the appointments, the open preg.* inbox rows — an ack, a
//     dismiss, another evaluation), the served rule snapshot, the clinical registry valid today and the binary;
//   - next_due_at: the earliest instant a time-based rule may change its answer with no write (NextDue + the rule
//     gates of the served snapshot).
//
// The pair is kept per user in Redis (cache.Client.Key). The evaluation is skipped only when the stored revision
// equals the request's AND evaluated_at ≤ now < next_due_at on the request clock (X-Test-Now in tests). Any doubt —
// no cache, a Redis error, no revisions on the request, a stamp read error, an input that moved during the
// evaluation, an unknown due — evaluates (fail safe).

// NextDue is the earliest instant after now at which a time-based detector may change its answer although no input
// was written (now is the evaluation instant). Every rule that reads the clock is listed:
//
//   - the local day change (Tehran midnight, the day every detector runs on — civildate.InTehran): week_entered and
//     the week's weekday (weight_missing_week from weekday 5, days since the last weight), every 7-day window
//     (vomiting, severe symptoms, critical symptoms, manual fetal movement, BP, GDM, depression), the kick look-back
//     and gestational week, next_visit_in_days, the dating itself and the clinical registry versions valid on a day;
//   - the end of the counting window of a running kick session (H-05: start + pregnancy.kicks.window_min, after
//     which a session short of the target counts as ended and raises the urgent fetal_movement alert);
//   - the next appointment's own instant (once it passes, the next visit is the following one).
//
// 5-1-1 is not time-based: it reads only the contractions of the running session (a write).
func NextDue(f Facts, nextVisit, now time.Time) time.Time {
	due := civildate.InTehran(now).AddDays(1).TehranMidnight()
	earlier := func(t time.Time) {
		if t.After(now) && t.Before(due) {
			due = t
		}
	}
	for _, k := range f.Kicks {
		if !k.End.IsZero() {
			continue
		}
		earlier(k.Start.Add(KickParamsOn(civildate.InTehran(k.Start)).Window))
	}
	if !nextVisit.IsZero() {
		earlier(nextVisit)
	}
	return due
}

// RulesVersioner is the optional side of a Backend that names the served rule snapshot (it moves on an admin rule
// edit). Without it nothing is cached.
type RulesVersioner interface {
	RulesVersion(ctx context.Context) (string, error)
}

// Stamp is the content digest of the alert inputs no revision covers (GetV2AlertInputStamp).
type Stamp struct{ Data, Inbox string }

// InputStamp reads the stamp of userID at now.
func (e *Engine) InputStamp(ctx context.Context, userID uint64, now time.Time) (Stamp, error) {
	today := civildate.InTehran(now)
	r, err := e.q.GetV2AlertInputStamp(ctx, store.GetV2AlertInputStampParams{
		UserID:            userID,
		KicksSince:        today.AddDays(-(KickLookbackDays + 1)).TehranMidnight(),
		AppointmentsSince: nullTimeOf(today.TehranMidnight()),
	})
	if err != nil {
		return Stamp{}, fmt.Errorf("pregnancy alerts: input stamp: %w", err)
	}
	d, ok1 := text(r.DataStamp)
	i, ok2 := text(r.InboxStamp)
	if !ok1 || !ok2 {
		return Stamp{}, errors.New("pregnancy alerts: input stamp: unexpected column type")
	}
	return Stamp{Data: d, Inbox: i}, nil
}

func text(v any) (string, bool) {
	switch x := v.(type) {
	case []byte:
		return string(x), true
	case string:
		return x, true
	}
	return "", false
}

// evalKey is the Redis key of userID's last evaluation (cache.Client prefixes it).
func evalKey(userID uint64) string { return "preg-alerts:eval:" + strconv.FormatUint(userID, 10) }

// revisionOf is the input revision of one request.
func revisionOf(revs auth.Revisions, st Stamp, rules, build string, today civildate.Date) string {
	keys := make([]string, 0, len(clinicalconfig.Defs))
	for _, d := range clinicalconfig.Defs {
		keys = append(keys, d.Key)
	}
	return strings.Join([]string{
		"e" + strconv.FormatUint(revs.Engine, 10), "a" + strconv.FormatUint(revs.Analysis, 10),
		"d" + st.Data, "i" + st.Inbox, "r" + rules, "c" + clinicalconfig.Fingerprint(today, keys...), "b" + build,
	}, ";")
}

// entry is the stored value: revision, evaluated_at and next_due_at (unix seconds).
type entry struct {
	rev        string
	at, dueSec int64
}

func (e entry) String() string {
	return strconv.FormatInt(e.at, 10) + "|" + strconv.FormatInt(e.dueSec, 10) + "|" + e.rev
}

func parseEntry(s string) (entry, bool) {
	parts := strings.SplitN(s, "|", 3)
	if len(parts) != 3 {
		return entry{}, false
	}
	at, err1 := strconv.ParseInt(parts[0], 10, 64)
	due, err2 := strconv.ParseInt(parts[1], 10, 64)
	if err1 != nil || err2 != nil {
		return entry{}, false
	}
	return entry{rev: parts[2], at: at, dueSec: due}, true
}

// fresh reports whether e lets the evaluation of rev at now be skipped.
func (e entry) fresh(rev string, now time.Time) bool {
	return e.rev == rev && e.at <= now.Unix() && now.Unix() < e.dueSec
}

func nullTimeOf(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: true} }
