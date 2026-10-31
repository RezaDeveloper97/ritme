package messages

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/messages"
	"github.com/ritme/backend-go/internal/messages/feed"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Message simulator v6 (T-M10-11, SPEC §5.4.1): POST /messages/simulate {user_id?, date?, time?, slots?,
// facts_override?} runs selection v6 per slot and returns the engine summary, the winners of each slot with their
// variant and every dropped candidate with its reason (rules.SimulateSlot: the T-M10-04 reasons plus
// missing_variable). It never writes events.
//
//   - With user_id it reads a real user's health state: the admin needs the health capability and a live approved
//     access request for that user (httpadmin.HealthGrant: reason + ticket, 30 minutes) — else 403
//     health_access_required — and the health_access.read audit row is written, fail closed (503
//     audit_unavailable), before any fact is built. The facts are the user's real facts for that date (her zone) and
//     the history is her real message history, so caps, cooldowns and the 24 h after a safety message trigger.
//   - Without user_id the facts are facts_override only and the history is empty (source synthetic).

// SimulateSlots are the slots the simulator answers (feed.DefaultSlots when the request names none).
var SimulateSlots = []string{"urgent_modal", "status_banner", "smart_tip", "daily_advice", "daily_challenge", "modal",
	"push", rules.SlotAlertsInbox, "sms"}

// EngineSummaryFacts are the engine facts the simulator summarises.
var EngineSummaryFacts = []string{"main_phase", "subphase", "cycle_day", "days_late", "confidence", "data_quality",
	"warnings", "requires_user_input"}

// Simulation sources.
const (
	SourceUser      = "user"
	SourceSynthetic = "synthetic"
)

type simulateV6Body struct {
	UserID        uint64         `json:"user_id"`
	Date          string         `json:"date"`
	Time          string         `json:"time"`
	Slots         []string       `json:"slots"`
	FactsOverride map[string]any `json:"facts_override"`
}

// userZoner is the user's stored zone (user_profiles.timezone). The user-facing handlers do not expose it yet
// (T-M10-11 report): until they do the simulator runs in Asia/Tehran and says so (zone).
type userZoner interface {
	UserZone(ctx context.Context, userID uint64) (*time.Location, error)
}

var timeLayouts = []string{"15:04", "15:04:05"}

func (b *simulateV6Body) slots() ([]string, error) {
	if b.Slots == nil {
		return slices.Clone(feed.DefaultSlots), nil
	}
	var out []string
	for _, s := range b.Slots {
		if !slices.Contains(SimulateSlots, s) {
			return nil, httpadmin.FieldError("slots", "The slot «"+s+"» is invalid.")
		}
		if !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil, httpadmin.FieldError("slots", "Name at least one slot.")
	}
	return out, nil
}

// clock is the simulated moment in loc: date (default today in loc) at time (default the current time of day).
func (b *simulateV6Body) clock(wall time.Time, loc *time.Location) (time.Time, error) {
	local := wall.In(loc)
	day := civildate.FromTime(local)
	if b.Date != "" {
		d, err := civildate.Parse(b.Date)
		if err != nil {
			return time.Time{}, httpadmin.FieldError("date", "The date is invalid (Y-m-d).")
		}
		day = d
	}
	hh, mm, ss := local.Hour(), local.Minute(), local.Second()
	if b.Time != "" {
		var (
			t   time.Time
			err error
		)
		for _, l := range timeLayouts {
			if t, err = time.Parse(l, b.Time); err == nil {
				break
			}
		}
		if err != nil {
			return time.Time{}, httpadmin.FieldError("time", "The time is invalid (H:i or H:i:s).")
		}
		hh, mm, ss = t.Hour(), t.Minute(), t.Second()
	}
	return time.Date(day.Year, day.Month, day.Day, hh, mm, ss, 0, loc), nil
}

// SimulateV6 is POST /messages/simulate.
func (h *Handlers) SimulateV6(c fiber.Ctx) error {
	var b simulateV6Body
	if err := json.Unmarshal(c.Body(), &b); err != nil {
		return httpadmin.FieldError("user_id", "The request body is not valid JSON.")
	}
	slots, err := b.slots()
	if err != nil {
		return err
	}
	ctx := c.Context()
	wall := httpadmin.Now(c)
	loc := civildate.Tehran
	if b.UserID > 0 {
		// Gate first: without a grant nothing about the user (not even whether she exists) is answered.
		grant, err := httpadmin.HealthGrant(c, h.q, b.UserID)
		if err != nil {
			return err
		}
		if grant == nil {
			return httpadmin.HealthAccessRequired()
		}
		if _, err := h.mq.GetMessageUserName(ctx, b.UserID); errors.Is(err, sql.ErrNoRows) {
			return httpadmin.NotFound("User")
		} else if err != nil {
			return err
		}
		if z, ok := any(h.user).(userZoner); ok {
			if loc, err = z.UserZone(ctx, b.UserID); err != nil {
				return err
			}
		}
		now, err := b.clock(wall, loc)
		if err != nil {
			return err
		}
		// T-M10-33: audited before any health fact is built; no row, no data (503).
		if err := httpadmin.AuditHealthRead(c, h.logger, b.UserID, grant.ID); err != nil {
			return err
		}
		httpadmin.Audit(c, h.logger, "messages.simulate", "user", b.UserID, slog.Uint64("request_id", grant.ID),
			slog.String("at", now.Format(time.RFC3339)), slog.String("slots", strings.Join(slots, ",")))
		f, err := messages.BuildFacts(ctx, h.user, b.UserID, civildate.FromTime(now))
		if err != nil {
			return err
		}
		overridden := override(f, b.FactsOverride)
		hist, err := rules.LoadHistory(ctx, h.mq, b.UserID, rules.CapWindows(now, f))
		if err != nil {
			return err
		}
		return h.simulateOut(c, SourceUser, b.UserID, now, wall, slots, f, hist, overridden)
	}
	now, err := b.clock(wall, loc)
	if err != nil {
		return err
	}
	f := rules.Facts{}
	overridden := override(f, b.FactsOverride)
	return h.simulateOut(c, SourceSynthetic, 0, now, wall, slots, f, nil, overridden)
}

// override writes the override facts over f and returns their names (sorted).
func override(f rules.Facts, with map[string]any) []string {
	keys := make([]string, 0, len(with))
	for k, v := range with {
		f[k] = v
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func (h *Handlers) simulateOut(c fiber.Ctx, source string, userID uint64, now, wall time.Time, slots []string,
	f rules.Facts, hist *rules.UserHistory, overridden []string,
) error {
	rows, err := rules.ActiveRows(c.Context(), h.mq, "") // the feed's rows: served versions, approved snapshots applied
	if err != nil {
		return err
	}
	all := make([]rules.Rule, len(rows))
	titles := make(map[uint64]string, len(rows))
	for i := range rows {
		all[i] = rules.FromRow(&rows[i])
		titles[rows[i].ID] = rows[i].Title
	}
	byID := make(map[uint64]*rules.Rule, len(all))
	for i := range all {
		byID[all[i].ID] = &all[i]
	}
	out := make([]*jsonx.OrderedMap, 0, len(slots))
	for _, slot := range slots {
		winners, dropped := rules.SimulateSlot(slot, all, f, hist, userID, now)
		ws := make([]*jsonx.OrderedMap, 0, len(winners))
		for _, w := range winners {
			r := w.Rule
			v, _ := rules.RenderVariant(r.VariantContent(w.Variant), f)
			ws = append(ws, jsonx.Obj("rule_id", r.ID, "rule_key", orNull(r.RuleKey), "title", titles[r.ID], "kind", r.Kind,
				"level", r.Level, "priority", r.Priority, "version", feed.ServedVersion(r), "variant", w.Variant,
				"message_title", v.Title, "message_body", v.Body, "cta_label", orNull(v.CTALabel)))
		}
		ds := make([]*jsonx.OrderedMap, 0, len(dropped))
		for _, d := range dropped {
			r := byID[d.RuleID]
			ds = append(ds, jsonx.Obj("rule_id", d.RuleID, "title", titles[d.RuleID], "kind", r.Kind, "priority", r.Priority,
				"reason", d.Reason))
		}
		s := rules.Slots[slot]
		var capacity any = s.Capacity
		if s.List {
			capacity = nil
		}
		out = append(out, jsonx.Obj("slot", slot, "capacity", capacity, "winners", ws, "dropped", ds))
	}
	engine := jsonx.Obj()
	for _, k := range EngineSummaryFacts {
		engine.Set(k, f[k])
	}
	var uid any
	notes := []string{}
	history := "empty"
	if source == SourceUser {
		uid, history = userID, "real"
		notes = append(notes,
			"The engine resolves this date over the user's current inputs: logs and settings saved later count, nothing is replayed as of that day.",
			"The history is the user's real message history; it holds every event recorded so far.")
		if now.Before(wall) {
			notes = append(notes, "The simulated moment is in the past: events recorded after it still count towards caps and cooldowns.")
		}
	} else {
		notes = append(notes, "Synthetic facts and an empty history: caps, cooldowns and the 24 h after a safety message never trigger.")
	}
	reasons := slices.Concat([]string{rules.ReasonInactive, rules.ReasonNotApproved}, rules.SelectionReasons,
		[]string{rules.ReasonMissingVariable})
	return httpadmin.OK(c, jsonx.Obj("source", source, "user_id", uid, "at", now.Format(time.RFC3339),
		"date", civildate.FromTime(now).String(), "time", now.Format("15:04:05"), "zone", now.Location().String(),
		"resolution", map[string]string{SourceUser: "current_inputs", SourceSynthetic: "synthetic"}[source],
		"history", history, "facts_overridden", overridden, "engine", engine, "slots", out,
		"reasons", jsonx.List(reasons), "notes", notes))
}

func orNull(s string) any {
	if s == "" {
		return nil
	}
	return s
}
