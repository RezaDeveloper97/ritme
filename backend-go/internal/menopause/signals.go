package menopause

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/checkups"
	"github.com/ritme/backend-go/internal/checkups/engine"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/menopause/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// PlacementHome is the meno_tips meta.placement of the tips shown on the home (the stage tips).
const PlacementHome = "home"

// Signals are the facts the menopause message engine (CB-MENO-12, internal/messages/menomessages) turns into
// alerts and tips. They are this package's own rules, read in one place so the engine never re-implements them:
// the stage rule (§2), the bleeding flag of the home (BleedingAlertDays), the latest questionnaire with its delta,
// the menopause checkups of her M4 plan and her active HRT items.
type Signals struct {
	Date civildate.Date
	// Mode is her effective life mode (enums.ResolveLifeMode, as the checkups plan resolves it). Every other field
	// is empty unless it is menopause.
	Mode     enums.LifeMode
	Stage    Stage
	Bleeding Bleeding
	// Score is her newest questionnaire (any month) with the one before it; nil when she never filled one.
	Score *ScoreEntry
	Plan  *checkups.Plan
	// Checkups are the pending menopause checkups of the plan (overdue, due, soon or never done), overdue first.
	Checkups []engine.Item
	// HRT are her hormone-therapy items active today, in treatment order.
	HRT []store.TreatmentItem
	// Tips are the active meno_tips items placed on the home for her stage (meta.stages empty = every stage; no
	// stage yet = the "unsure" tips).
	Tips []catalog.Item
}

// Signals loads the message facts of userID on now. A user who is not in menopause mode gets only Date and Mode
// (one plan read, nothing else).
func (s *Service) Signals(ctx context.Context, userID uint64, now time.Time) (Signals, error) {
	today := civildate.InTehran(now)
	sig := Signals{Date: today}
	p, err := checkups.LoadPlan(ctx, s.db, userID, clock.At(now))
	if err != nil {
		return Signals{}, err
	}
	if sig.Mode = p.LifeMode; sig.Mode != enums.LifeModeMenopause {
		return sig, nil
	}
	sig.Plan = p
	if sig.Stage, err = s.Profile(ctx, userID, today); err != nil {
		return Signals{}, err
	}
	logs, err := s.logDays(ctx, userID, today.AddDays(-(BleedingAlertDays - 1)), today)
	if err != nil {
		return Signals{}, err
	}
	if sig.Bleeding, err = s.bleeding(ctx, sig.Stage, logs); err != nil {
		return Signals{}, err
	}
	if sig.Score, err = s.newestScore(ctx, userID, today); err != nil {
		return Signals{}, err
	}
	if sig.Checkups, err = s.pendingCheckups(ctx, p); err != nil {
		return Signals{}, err
	}
	if sig.HRT, err = s.activeHRT(ctx, userID, today); err != nil {
		return Signals{}, err
	}
	if sig.Tips, err = s.HomeTips(ctx, sig.Stage); err != nil {
		return Signals{}, err
	}
	return sig, nil
}

// newestScore is her newest questionnaire (this Jalali month included) with the one before it.
func (s *Service) newestScore(ctx context.Context, userID uint64, today civildate.Date) (*ScoreEntry, error) {
	sc, err := s.PreviousScore(ctx, userID, engine.JalaliMonthEnd(today).AddDays(1))
	if err != nil || sc == nil {
		return nil, err
	}
	prev, err := s.PreviousScore(ctx, userID, sc.Month)
	if err != nil {
		return nil, err
	}
	return &ScoreEntry{Score: *sc, Previous: prev}, nil
}

// activeHRT are the hrt items active on today.
func (s *Service) activeHRT(ctx context.Context, userID uint64, today civildate.Date) ([]store.TreatmentItem, error) {
	items, err := s.q.ListTreatmentItems(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("menopause: list treatment: %w", err)
	}
	out := []store.TreatmentItem{}
	for _, it := range items {
		if it.Kind == KindHRT && activeOn(it, today) {
			out = append(out, it)
		}
	}
	return out, nil
}

// HomeTips are the active meno_tips items with meta.placement home whose meta.stages hold her stage (no stages =
// every stage). Without a stage the "unsure" tips are picked: they ask for the last period.
func (s *Service) HomeTips(ctx context.Context, st Stage) ([]catalog.Item, error) {
	items, err := s.catalog.Items(ctx, GroupTips)
	if err != nil {
		return nil, fmt.Errorf("menopause: load catalog %s: %w", GroupTips, err)
	}
	stage := st.Stage
	if stage == "" {
		stage = StageUnsure
	}
	out := []catalog.Item{}
	for _, it := range items {
		var meta struct {
			Placement string   `json:"placement"`
			Stages    []string `json:"stages"`
		}
		_ = json.Unmarshal(it.Meta, &meta)
		if meta.Placement == PlacementHome && (len(meta.Stages) == 0 || slices.Contains(meta.Stages, stage)) {
			out = append(out, it)
		}
	}
	return out, nil
}
