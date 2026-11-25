package safety

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/pregnancy/tools"
	"github.com/ritme/backend-go/internal/safety/store"
	"github.com/ritme/backend-go/internal/vitals"
)

// WindowDays is the signal window; the delta compares it with the window before.
const WindowDays = 7

// Signal keys (display order of the artboard).
const (
	SignalBPCrisis      = "bp_crisis"
	SignalGlucoseLow    = "glucose_urgent_low"
	SignalEPDSQ10       = "epds_q10"
	SignalReducedKicks  = "reduced_kicks"
	SignalContractions  = "contractions_511"
	SignalHeavyBleeding = "postpartum_heavy_bleeding"
	thresholdApprovals  = 1  // D-215 (was D-88's two advisors): one clinical approval publishes a clinical config version
	SLAHours            = 72 // review SLA of every source (D-90)
	claimLookbackDays   = 30
	claimTargetPrefix   = "safety."
	ActionClaim         = "safety.claim"
	ActionRelease       = "safety.release"
	maxTitleRunes       = 160
)

// Review sources.
const (
	SourceMessage   = "message"   // message_rules awaiting medical approval (B-N9-05)
	SourceThreshold = "threshold" // clinical_config_versions pending (B-N9-11)
	SourceLesson    = "lesson"    // learning_lessons open review states (B-N8-08)
	SourceAssistant = "assistant" // assistant_departments needs_review (B-N7-08)
	SourceCatalog   = "catalog"   // catalog_items needs_review, one item per group
)

// Sources are the queue sources in display order.
var Sources = []string{SourceMessage, SourceThreshold, SourceLesson, SourceAssistant, SourceCatalog}

// UrgentCopyGroups are the message_contents groups that carry urgent-path copy (the approved-text parity check).
var UrgentCopyGroups = []string{"emergency", vitals.AlertGroup, "pregnancy_alert", "condition_alerts", "meno_alerts",
	"loss_warning_signs"}

// Signal is one aggregated count: this window and the one before.
type Signal struct {
	Key      string
	Count    int64
	Previous int64
}

// Service reads the dashboard.
type Service struct{ q *store.Queries }

// New wires the service.
func New(db store.DBTX) *Service { return &Service{q: store.New(db)} }

type window struct {
	from, to         time.Time
	fromDay, toDayEx civildate.Date
}

func windows(now time.Time) (cur, prev window) {
	today := civildate.InTehran(now)
	end := today.AddDays(1)
	start := end.AddDays(-WindowDays)
	pstart := start.AddDays(-WindowDays)
	mk := func(a, b civildate.Date) window {
		return window{from: a.TehranMidnight(), to: b.TehranMidnight(), fromDay: a, toDayEx: b}
	}
	return mk(start, end), mk(pstart, start)
}

func (s *Service) count(ctx context.Context, key string, w window, t vitals.Thresholds) (int64, error) {
	at := func(x time.Time) sql.NullTime { return sql.NullTime{Time: x, Valid: true} }
	switch key {
	case SignalBPCrisis:
		return s.q.CountBPCrisisReadings(ctx, store.CountBPCrisisReadingsParams{FromAt: w.from, ToAt: w.to,
			Systolic: nullInt16(t.BPCrisisSystolic), Diastolic: nullInt16(t.BPCrisisDiastolic)})
	case SignalGlucoseLow:
		return s.q.CountGlucoseUrgentLowReadings(ctx, store.CountGlucoseUrgentLowReadingsParams{FromAt: w.from, ToAt: w.to,
			Below: sql.NullString{String: strconv.FormatFloat(t.GlucoseUrgentBelow, 'f', 1, 64), Valid: true}})
	case SignalEPDSQ10:
		return s.q.CountEPDSSelfHarmChecks(ctx, store.CountEPDSSelfHarmChecksParams{FromDay: w.fromDay, ToDay: w.toDayEx})
	case SignalReducedKicks:
		return s.q.CountReducedKickSessions(ctx, store.CountReducedKickSessionsParams{FromAt: at(w.from), ToAt: at(w.to),
			KickTarget: tools.KickTarget, WindowMinutes: tools.KickWindowMinutes})
	case SignalContractions:
		return s.q.CountContraction511Alerts(ctx, store.CountContraction511AlertsParams{FromAt: at(w.from), ToAt: at(w.to)})
	case SignalHeavyBleeding:
		return s.q.CountHeavyLochiaDays(ctx, store.CountHeavyLochiaDaysParams{FromDay: w.fromDay, ToDay: w.toDayEx})
	}
	return 0, fmt.Errorf("safety: unknown signal %q", key)
}

func nullInt16(v float64) sql.NullInt16 {
	return sql.NullInt16{Int16: int16(min(max(math.Floor(v), 0), math.MaxInt16)), Valid: true}
}

// SignalKeys are the signals in display order.
var SignalKeys = []string{SignalBPCrisis, SignalGlucoseLow, SignalEPDSQ10, SignalReducedKicks, SignalContractions,
	SignalHeavyBleeding}

// Signals counts every signal over the last WindowDays (today included) and the window before, with the thresholds
// in force today.
func (s *Service) Signals(ctx context.Context, now time.Time) ([]Signal, error) {
	cur, prev := windows(now)
	t := vitals.ThresholdsOn(civildate.InTehran(now))
	out := make([]Signal, 0, len(SignalKeys))
	for _, k := range SignalKeys {
		c, err := s.count(ctx, k, cur, t)
		if err != nil {
			return nil, fmt.Errorf("safety: count %s: %w", k, err)
		}
		p, err := s.count(ctx, k, prev, t)
		if err != nil {
			return nil, fmt.Errorf("safety: count %s: %w", k, err)
		}
		out = append(out, Signal{Key: k, Count: c, Previous: p})
	}
	return out, nil
}

// Claim is who took an item.
type Claim struct {
	AdminID uint64
	Name    string
	At      time.Time
}

// Item is one review-queue row: metadata only (never the content under review).
type Item struct {
	Source      string
	ID          uint64
	Title       string
	Ref         string // catalog group / config key / department code
	Status      string // pending | changed | partially_approved
	Approvals   int
	Required    int
	Items       int // catalog rows of the group (1 elsewhere)
	SubmittedBy string
	SubmittedAt time.Time
	Claim       *Claim
}

// AgeHours is how long the item has waited.
func (it Item) AgeHours(now time.Time) int {
	if it.SubmittedAt.IsZero() || now.Before(it.SubmittedAt) {
		return 0
	}
	return int(now.Sub(it.SubmittedAt) / time.Hour)
}

func stamp(ts ...sql.NullTime) time.Time {
	for _, t := range ts {
		if t.Valid {
			return t.Time
		}
	}
	return time.Time{}
}

func truncate(s string) string {
	r := []rune(s)
	if len(r) > maxTitleRunes {
		return string(r[:maxTitleRunes])
	}
	return s
}

// Queue lists every open review item, oldest first per source; locale/def pick translatable titles.
func (s *Service) Queue(ctx context.Context, now time.Time, locale, def string) ([]Item, error) {
	var out []Item
	msgs, err := s.q.ListSafetyPendingMessageRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("safety: messages: %w", err)
	}
	for _, m := range msgs {
		out = append(out, Item{Source: SourceMessage, ID: m.ID, Title: truncate(m.Title), Ref: m.Level, Status: "pending",
			Items: 1, SubmittedBy: m.SubmittedBy.String, SubmittedAt: stamp(m.UpdatedAt, m.CreatedAt)})
	}
	cfg, err := s.q.ListSafetyPendingClinicalConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("safety: thresholds: %w", err)
	}
	for _, v := range cfg {
		st := "pending"
		if v.Approvals > 0 {
			st = "partially_approved"
		}
		out = append(out, Item{Source: SourceThreshold, ID: v.ID, Title: v.ConfigKey + " v" + strconv.FormatUint(uint64(v.Version), 10),
			Ref: v.ConfigKey, Status: st, Approvals: int(v.Approvals), Required: thresholdApprovals, Items: 1,
			SubmittedBy: v.SubmittedBy.String, SubmittedAt: stamp(v.CreatedAt)})
	}
	lessons, err := s.q.ListSafetyLessonsToReview(ctx)
	if err != nil {
		return nil, fmt.Errorf("safety: lessons: %w", err)
	}
	for _, l := range lessons {
		st := "pending"
		if l.Changed == 1 {
			st = "changed"
		}
		out = append(out, Item{Source: SourceLesson, ID: l.ID, Title: truncate(l.Title), Status: st, Items: 1,
			SubmittedBy: l.SubmittedBy, SubmittedAt: stamp(l.UpdatedAt, l.CreatedAt)})
	}
	deps, err := s.q.ListSafetyAssistantDepartments(ctx)
	if err != nil {
		return nil, fmt.Errorf("safety: assistant: %w", err)
	}
	for _, d := range deps {
		title := strings.TrimSpace(i18n.PickString(json.RawMessage(d.Name), locale, def))
		if title == "" {
			title = d.Code
		}
		out = append(out, Item{Source: SourceAssistant, ID: d.ID, Title: truncate(title), Ref: d.Code, Status: "pending",
			Items: 1, SubmittedAt: stamp(d.UpdatedAt, d.CreatedAt)})
	}
	groups, err := s.q.ListSafetyCatalogGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("safety: catalog: %w", err)
	}
	for _, g := range groups {
		out = append(out, Item{Source: SourceCatalog, ID: uint64(max(g.FirstID, 0)), Title: g.Group,
			Ref: g.Group, Status: "pending", Items: int(g.Items), SubmittedAt: g.OldestAt})
	}
	if err := s.attachClaims(ctx, now, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ClaimTarget is the audit-log target type of a source's claims.
func ClaimTarget(source string) string { return claimTargetPrefix + source }

func (s *Service) attachClaims(ctx context.Context, now time.Time, items []Item) error {
	rows, err := s.q.ListSafetyClaims(ctx, sql.NullTime{Time: now.AddDate(0, 0, -claimLookbackDays), Valid: true})
	if err != nil {
		return fmt.Errorf("safety: claims: %w", err)
	}
	type key struct {
		t  string
		id uint64
	}
	last := map[key]*Claim{}
	for _, r := range rows {
		k := key{r.TargetType, r.TargetID}
		if r.Action == ActionRelease || !r.AdminID.Valid {
			delete(last, k)
			continue
		}
		last[k] = &Claim{AdminID: uint64(max(r.AdminID.Int64, 0)), Name: r.AdminName.String, At: stamp(r.CreatedAt)}
	}
	for i := range items {
		items[i].Claim = last[key{ClaimTarget(items[i].Source), items[i].ID}]
	}
	return nil
}

// UnapprovedUrgentCopy counts live urgent copy rows that are not approved (0 = parity).
func (s *Service) UnapprovedUrgentCopy(ctx context.Context) (int64, error) {
	return s.q.CountUnapprovedUrgentCopy(ctx, UrgentCopyGroups)
}
