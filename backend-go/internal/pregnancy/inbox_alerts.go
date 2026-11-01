package pregnancy

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/messages/pregnancyalerts"
	"github.com/ritme/backend-go/internal/messages/pregnancyalerts/inboxalerts"
	"github.com/ritme/backend-go/internal/messages/rules"
	mstore "github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/pregnancy/alerts"
	"github.com/ritme/backend-go/internal/pregnancy/store"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"
)

// The v1 alert endpoints over the alerts inbox (T-M10-08b, D-165). Every pregnancy alert since T-M10-08 is a
// message_inbox row of a preg.* rule; the v1 routes answer those rows in the v1 PregnancyAlert shape (key order of
// alertModel) next to the legacy pregnancy_alerts rows written before T-M10-08 (db/queries/pregnancy/alerts.sql):
//
//	id            the message_inbox id          alert_type        the rule's v2 key (critical_symptom, bp_high, …)
//	alert_level   urgent → emergency, follow_up → warning, else info (pregnancyalerts.V1Level)
//	title/message the rule's texts in the request locale, rendered with the stored evidence (engine copy)
//	is_read       state ≠ active (read = acknowledged)     is_dismissed  state = archived (dismiss = archived)
//	created_at    first hit                     updated_at        last hit (the list's sort key)

// inboxRow is one preg.* inbox row as the v1 routes read it.
type inboxRow store.ListPregnancyInboxAlertsRow

// listInbox is the user's preg.* rows the v1 list shows (not archived), newest activity first.
func listInbox(ctx context.Context, q store.Querier, userID uint64) ([]inboxRow, error) {
	rows, err := q.ListPregnancyInboxAlerts(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("pregnancy: inbox alerts: %w", err)
	}
	out := make([]inboxRow, len(rows))
	for i := range rows {
		out[i] = inboxRow(rows[i])
	}
	return out, nil
}

// inboxRowOf converts the single-row read.
func inboxRowOf(r store.GetPregnancyInboxAlertRow) inboxRow { return inboxRow(r) }

func (r *inboxRow) isRead() bool { return r.State != stateActive }

// v1Level is the row's v1 alert_level.
func (r *inboxRow) v1Level() string { return pregnancyalerts.V1Level(rules.SpecLevel(r.Level)) }

// at is the row's activity time (last hit, else first): the v1 list's newest-first order.
func (r *inboxRow) at() time.Time {
	if r.LastAt.Valid {
		return r.LastAt.Time
	}
	return r.FirstAt.Time
}

// Inbox states the v1 routes distinguish (messages/inbox).
const (
	stateActive   = "active"
	stateArchived = "archived"
)

// vars are the evidence variables stored at the row's last hit.
func (r *inboxRow) vars() map[string]any {
	var d struct {
		Vars map[string]any `json:"vars"`
	}
	if !r.Evidence.Valid || json.Unmarshal(r.Evidence.V, &d) != nil || d.Vars == nil {
		return map[string]any{}
	}
	return d.Vars
}

// alertTexts renders inbox rows in one locale; it reads each rule once per request.
type alertTexts struct {
	q       store.Querier
	locale  string
	rules   map[uint64]*ruleTexts
	profile *store.PregnancyProfile // read once, for the week of the alerts whose evidence has none
	loaded  bool
}

// ruleTexts is a served rule's content (nil content = the code-held copy).
type ruleTexts struct {
	content      *rules.Content
	translations map[string]rules.Content
}

func newAlertTexts(q store.Querier, locale string) *alertTexts {
	return &alertTexts{q: q, locale: locale, rules: map[uint64]*ruleTexts{}}
}

// rule is the texts source of a rule id: its row's content when the row is served (as the inbox snapshot serves it),
// else nil — the code-held copy (a locked rule whose row is gone, off or unapproved; inboxalerts' guarantee).
func (t *alertTexts) rule(ctx context.Context, id uint64) (*ruleTexts, error) {
	if rt, ok := t.rules[id]; ok {
		return rt, nil
	}
	rt := &ruleTexts{}
	row, err := t.q.GetPregnancyAlertRule(ctx, id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return nil, fmt.Errorf("pregnancy: alert rule: %w", err)
	default:
		mr := mstore.MessageRule(row)
		r := rules.FromRow(&mr)
		if r.Served() && strings.HasPrefix(row.RuleKey.String, inboxalerts.KeyPrefix) {
			var sc struct {
				Translations map[string]rules.Content `json:"translations"`
			}
			_ = json.Unmarshal([]byte(row.Content), &sc)
			rt.content, rt.translations = &r.Content, sc.Translations
		}
	}
	t.rules[id] = rt
	return rt, nil
}

// copyOf is the row's title and body in the locale, before rendering.
func (t *alertTexts) copyOf(ctx context.Context, r *inboxRow) (inboxalerts.Copy, error) {
	rt, err := t.rule(ctx, r.RuleID)
	if err != nil {
		return inboxalerts.Copy{}, err
	}
	if rt.content == nil {
		c, _ := inboxalerts.CopyOf(r.RuleKey.String, t.locale)
		return c, nil
	}
	a := rt.content.A
	if tr, ok := rt.translations[t.locale]; ok && (tr.A.Title != "" || tr.A.Body != "") {
		a = tr.A
	}
	return inboxalerts.Copy{Title: a.Title, Body: a.Body}, nil
}

// week is the v1 pregnancy_week: the evidence's preg_week, else the week of the active pregnancy on the day of the
// alert's first hit (v1 stored the week the alert was created in); nil without either.
func (t *alertTexts) week(ctx context.Context, userID uint64, r *inboxRow, vars map[string]any) (any, error) {
	if w, ok := vars["preg_week"].(float64); ok {
		return int64(w), nil
	}
	if err := t.loadProfile(ctx, userID); err != nil {
		return nil, err
	}
	if t.profile == nil || !t.profile.PregnancyMode || !r.FirstAt.Valid {
		return nil, nil
	}
	d, ok := v2.Resolve(t.profile, civildate.FromTime(r.FirstAt.Time.In(civildate.Tehran)))
	if !ok {
		return nil, nil
	}
	return int64(d.CurrentWeek()), nil
}

// loadProfile reads the user's pregnancy profile once (nil without one).
func (t *alertTexts) loadProfile(ctx context.Context, userID uint64) error {
	if t.loaded {
		return nil
	}
	t.loaded = true
	p, err := t.q.GetProfileByUser(ctx, userID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return fmt.Errorf("pregnancy: alert profile: %w", err)
	default:
		t.profile = &p
	}
	return nil
}

// inboxSymptomRules are the rules over the symptom logs: v1 stored medical_history_flags on its symptom alerts only.
var inboxSymptomRules = []string{"preg.critical_symptom", "preg.severe_symptoms", "preg.vomiting", "preg.cant_keep_fluids"}

// medicalFlags is the v1 medical_history_flags of a row: the profile's flags for a symptom rule (`[]` when none, as
// the symptom save answers them), else null. json loads the profile first.
func (t *alertTexts) medicalFlags(r *inboxRow) any {
	if !slices.Contains(inboxSymptomRules, r.RuleKey.String) {
		return nil
	}
	return alerts.MedicalHistoryFlags(t.profile)
}

// json is the row in the v1 alert shape (alertModel's keys and casts).
func (t *alertTexts) json(ctx context.Context, userID uint64, r *inboxRow) (any, error) {
	c, err := t.copyOf(ctx, r)
	if err != nil {
		return nil, err
	}
	def, _ := inboxalerts.DefOf(r.RuleKey.String)
	key := def.Legacy
	if key == "" {
		key = r.RuleKey.String
	}
	vars := r.vars()
	title := inboxalerts.Render(c.Title, vars, t.locale)
	if title == "" {
		title = key
	}
	if err := t.loadProfile(ctx, userID); err != nil {
		return nil, err
	}
	week, err := t.week(ctx, userID, r, vars)
	if err != nil {
		return nil, err
	}
	actions := make([]any, 0, len(def.Actions))
	for _, a := range def.Actions {
		actions = append(actions, pregnancyalerts.T("actions."+a, t.locale))
	}
	var readAt sql.NullTime
	if r.isRead() {
		for _, ts := range []sql.NullTime{r.AckedAt, r.NotedAt, r.ResolvedAt, r.ArchivedAt, r.LastAt} {
			if ts.Valid {
				readAt = ts
				break
			}
		}
	}
	return newModel(alertCasts).
		set("id", i64(r.ID)).
		set("user_id", i64(userID)).
		set("alert_level", r.v1Level()).
		set("alert_type", key).
		set("title", title).
		set("message", inboxalerts.Render(c.Body, vars, t.locale)).
		set("pregnancy_week", week).
		set("trigger_symptoms", inboxalerts.V1Trigger(r.RuleKey.String, vars)).
		set("medical_history_flags", t.medicalFlags(r)).
		set("is_read", rawFlag(r.isRead())).
		set("is_dismissed", rawFlag(r.State == stateArchived)).
		set("read_at", rawTime(readAt)).
		set("dismissed_at", rawTime(r.ArchivedAt)).
		set("recommended_actions", actions).
		set("created_at", rawTime(r.FirstAt)).
		set("updated_at", rawTime(r.LastAt)).
		JSON(), nil
}

// inboxCounts are the v1 counts of the inbox rows (all of them: the counts ignore the list filters, as Laravel's).
func inboxCounts(rows []inboxRow) store.CountActiveAlertsRow {
	var c store.CountActiveAlertsRow
	for i := range rows {
		r := &rows[i]
		c.Total++
		unread := !r.isRead()
		if unread {
			c.Unread++
		}
		switch r.v1Level() {
		case "emergency":
			c.Emergency++
			if unread {
				c.UnreadEmergency++
			}
		case "warning":
			c.Warning++
		default:
			c.Info++
		}
	}
	return c
}

// addCounts sums the legacy and the inbox counts.
func addCounts(a, b store.CountActiveAlertsRow) store.CountActiveAlertsRow {
	return store.CountActiveAlertsRow{
		Total: a.Total + b.Total, Unread: a.Unread + b.Unread, Emergency: a.Emergency + b.Emergency,
		UnreadEmergency: a.UnreadEmergency + b.UnreadEmergency, Warning: a.Warning + b.Warning, Info: a.Info + b.Info,
	}
}

// archivePregnancyAlerts archives the user's preg.* inbox rows (a pregnancy loss; q may be a transaction).
func archivePregnancyAlerts(ctx context.Context, q store.Querier, userID uint64, now time.Time) error {
	if _, err := q.ArchivePregnancyInboxAlerts(ctx, store.ArchivePregnancyInboxAlertsParams{UserID: userID,
		Now: sql.NullTime{Time: dbNow(now), Valid: true}}); err != nil {
		return fmt.Errorf("pregnancy: archive inbox alerts: %w", err)
	}
	return nil
}
