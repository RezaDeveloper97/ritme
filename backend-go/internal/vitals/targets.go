package vitals

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
	"github.com/ritme/backend-go/internal/vitals/store"
)

// Diabetic glucose targets (T-M10-19, SPEC v6 §3.2, D-170). A user whose life profile lists `diabetes` in
// chronic_illnesses has her fasting / pre-meal readings classified against glucose.diabetic fasting_lo–fasting_hi
// (80–130, both inclusive) and her post-meal readings against post_meal_max (< 180): below_target · in_target ·
// above_target. A vital_targets row of the matching context (set by her doctor or herself) replaces the registry band;
// a bedtime row adds a bedtime target (no registry default). Random readings never have a target. Hypoglycaemia
// (< 70, < 54) and the pregnancy table (T-M9-05) win over every target. A user without diabetes is classified as
// before, even when she has target rows.

// Target contexts (vital_targets.context).
const (
	TargetFasting  = "fasting"
	TargetPreMeal  = "pre_meal"
	TargetPostMeal = "post_meal"
	TargetBedtime  = "bedtime"
)

// TargetContexts is every target context, in screen order.
var TargetContexts = []string{TargetFasting, TargetPreMeal, TargetPostMeal, TargetBedtime}

// Who set a target.
const (
	SetByDoctor = "doctor"
	SetByUser   = "user"
)

// SetByValues are the accepted set_by values.
var SetByValues = []string{SetByDoctor, SetByUser}

// Physiological bounds of a target value (mg/dL) and the note length (vital_targets.note varchar(160)).
const (
	MinTargetMgDl     = 50
	MaxTargetMgDl     = 300
	MaxTargetNoteLen  = 160
	diabetesCondition = string(enums.ChronicConditionDiabetes)
)

// TargetContext is the target context of a glucose reading context ("" = no target: random, unknown).
func TargetContext(readingContext string) string {
	switch readingContext {
	case ContextFasting:
		return TargetFasting
	case ContextBeforeMeal:
		return TargetPreMeal
	case ContextAfterMeal:
		return TargetPostMeal
	case ContextBedtime:
		return TargetBedtime
	}
	return ""
}

// Target is one target row: Low / High in mg/dL, 0 = open.
type Target struct {
	Context   string
	Low, High float64
	SetBy     string
	Note      string
	UpdatedAt time.Time
}

// Targets are a user's targets by target context.
type Targets map[string]Target

// TargetFromRow is a vital_targets row as a Target.
func TargetFromRow(r store.VitalTarget) Target {
	return Target{Context: r.Context, Low: i16(r.LowMgdl), High: i16(r.HighMgdl), SetBy: r.SetBy, Note: strOf(r.Note),
		UpdatedAt: timeOf(r.UpdatedAt)}
}

// DiabeticBand is the target band of a target context: Low ≤ v (0 = open), v ≤ High when HighInclusive, else v < High
// (0 = open).
type DiabeticBand struct {
	Low, High     float64
	HighInclusive bool
	Source        string // "registry" | "doctor" | "user"
}

// TargetSourceRegistry marks a band that comes from glucose.diabetic.
const TargetSourceRegistry = "registry"

// DefaultDiabeticBand is the registry band of a target context on t's day (ok false: bedtime, random).
func (t Thresholds) DefaultDiabeticBand(targetContext string) (DiabeticBand, bool) {
	switch targetContext {
	case TargetFasting, TargetPreMeal: // SPEC "before meals"; the registry names both fasting_lo / fasting_hi
		return DiabeticBand{Low: t.Diabetic.FastingLo, High: t.Diabetic.FastingHi, HighInclusive: true,
			Source: TargetSourceRegistry}, true
	case TargetPostMeal:
		return DiabeticBand{High: t.Diabetic.PostMealMax, Source: TargetSourceRegistry}, true
	}
	return DiabeticBand{}, false
}

// DiabeticBand is the band a diabetic reading of readingContext is compared with: the user's row, else the registry
// (ok false: no target for this context).
func (t Thresholds) DiabeticBand(readingContext string, targets Targets) (DiabeticBand, bool) {
	tc := TargetContext(readingContext)
	if tc == "" {
		return DiabeticBand{}, false
	}
	if row, ok := targets[tc]; ok && (row.Low > 0 || row.High > 0) {
		// The registry's post-meal ceiling is exclusive («< 180»); a fasting / pre-meal / bedtime range is inclusive.
		return DiabeticBand{Low: row.Low, High: row.High, HighInclusive: tc != TargetPostMeal, Source: row.SetBy}, true
	}
	return t.DefaultDiabeticBand(tc)
}

// classifyDiabetic is the target class of a diabetic reading (ok false: no target, the general rule applies).
// Hypoglycaemia wins.
func (t Thresholds) classifyDiabetic(mgdl float64, context string, targets Targets) (string, bool) {
	b, ok := t.DiabeticBand(context, targets)
	if !ok {
		return "", false
	}
	switch {
	case mgdl < t.GlucoseUrgentBelow:
		return ClassUrgentLow, true
	case mgdl < t.GlucoseLowBelow:
		return ClassLow, true
	case b.Low > 0 && mgdl < b.Low:
		return ClassBelowTarget, true
	case b.High > 0 && (mgdl > b.High || (!b.HighInclusive && mgdl == b.High)):
		return ClassAboveTarget, true
	}
	return ClassInTarget, true
}

// HasDiabetes reports whether a chronic_illnesses JSON list contains diabetes.
func HasDiabetes(raw []byte) bool {
	var codes []string
	if json.Unmarshal(raw, &codes) != nil {
		return false
	}
	return slices.Contains(codes, diabetesCondition)
}

// lifeProfiles reads the life profile (chronic_illnesses).
type lifeProfiles interface {
	GetLifeProfile(ctx context.Context, userID uint64) (profilestore.UserLifeProfile, error)
}

// Diabetic reports whether the user's life profile lists diabetes.
func (s *Service) Diabetic(ctx context.Context, userID uint64) (bool, error) {
	if s.life == nil {
		return false, nil
	}
	p, err := s.life.GetLifeProfile(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("vitals: life profile: %w", err)
	}
	return p.ChronicIllnesses.Valid && HasDiabetes(p.ChronicIllnesses.V), nil
}

// Targets is the user's target rows by context.
func (s *Service) Targets(ctx context.Context, userID uint64) (Targets, error) {
	rows, err := s.q.ListVitalTargets(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("vitals: targets: %w", err)
	}
	out := Targets{}
	for _, r := range rows {
		out[r.Context] = TargetFromRow(r)
	}
	return out, nil
}

// markDiabetes sets Diabetic and Targets on the glucose readings of rs (one profile read, targets only when diabetic).
func (s *Service) markDiabetes(ctx context.Context, userID uint64, rs []Reading) error {
	if !slices.ContainsFunc(rs, func(r Reading) bool { return r.Type == TypeGlucose }) {
		return nil
	}
	diabetic, err := s.Diabetic(ctx, userID)
	if err != nil || !diabetic {
		return err
	}
	targets, err := s.Targets(ctx, userID)
	if err != nil {
		return err
	}
	for i := range rs {
		if rs[i].Type == TypeGlucose {
			rs[i].Diabetic, rs[i].Targets = true, targets
		}
	}
	return nil
}

// SaveTargets replaces the user's targets: every context of items is upserted, every other context deleted.
func (s *Service) SaveTargets(ctx context.Context, userID uint64, items []Target, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("vitals: begin: %w", err)
	}
	q := s.q.WithTx(tx)
	kept := map[string]bool{}
	for _, it := range items {
		if err = q.UpsertVitalTarget(ctx, store.UpsertVitalTargetParams{
			UserID: userID, Context: it.Context, LowMgdl: nullI16(int(it.Low)), HighMgdl: nullI16(int(it.High)),
			SetBy: it.SetBy, Note: nullStr(it.Note), Now: sql.NullTime{Time: dbTime(now), Valid: true},
		}); err != nil {
			break
		}
		kept[it.Context] = true
	}
	for _, c := range TargetContexts {
		if err != nil {
			break
		}
		if !kept[c] {
			_, err = q.DeleteVitalTarget(ctx, store.DeleteVitalTargetParams{UserID: userID, Context: c})
		}
	}
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("vitals: save targets: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("vitals: commit: %w", err)
	}
	return nil
}

// ClearTargets deletes every target of the user.
func (s *Service) ClearTargets(ctx context.Context, userID uint64) error {
	for _, c := range TargetContexts {
		if _, err := s.q.DeleteVitalTarget(ctx, store.DeleteVitalTargetParams{UserID: userID, Context: c}); err != nil {
			return fmt.Errorf("vitals: clear targets: %w", err)
		}
	}
	return nil
}

// ValidateTargets is PUT /vitals/targets {items: [{context, low_mgdl?, high_mgdl?, set_by: doctor|user, note?}]}:
// one item per context, at least one bound, low < high, both within 50–300 mg/dL. An empty list clears the targets.
func ValidateTargets(body phpval.Map, locale string, now time.Time) ([]Target, error) {
	data := pick(body, []string{"items"})
	bound := between(MinTargetMgDl, MaxTargetMgDl)
	if err := validate(data, validation.Rules{
		validation.F("items", "present", "array", "max:"+strconv.Itoa(len(TargetContexts))),
		validation.F("items.*", "required", "array"),
		validation.F("items.*.context", "required", validation.In(TargetContexts...)),
		validation.F("items.*.low_mgdl", "nullable", "integer", bound),
		validation.F("items.*.high_mgdl", "nullable", "integer", bound),
		validation.F("items.*.set_by", "required", validation.In(SetByValues...)),
		validation.F("items.*.note", "nullable", "string", "max:"+strconv.Itoa(MaxTargetNoteLen)),
	}, locale, now); err != nil {
		return nil, err
	}
	raw, _ := data.Get("items")
	list := elems(raw)
	out := make([]Target, 0, len(list))
	seen := map[string]bool{}
	for i, v := range list {
		m, _ := v.(phpval.Map)
		if m == nil {
			m = phpval.NewMap()
		}
		it := Target{Context: str(m, "context"), Low: float64(intOf(m, "low_mgdl")),
			High: float64(intOf(m, "high_mgdl")), SetBy: str(m, "set_by"), Note: str(m, "note")}
		field := "items." + strconv.Itoa(i)
		switch {
		case seen[it.Context]:
			return nil, fieldFail(locale, field+".context", "duplicate_target")
		case it.Low == 0 && it.High == 0:
			return nil, fieldFail(locale, field+".high_mgdl", "target_bound_required")
		case it.Low > 0 && it.High > 0 && it.Low >= it.High:
			return nil, fieldFail(locale, field+".high_mgdl", "target_low_lt_high")
		}
		seen[it.Context] = true
		out = append(out, it)
	}
	return out, nil
}

func numOrNull(v float64) any {
	if v == 0 {
		return nil
	}
	return int(v)
}

// TargetsJSON is GET /vitals/targets: whether target classes apply, the registry defaults of the day and the user's
// rows (in context order).
func TargetsJSON(diabetic bool, targets Targets, t Thresholds) *jsonx.OrderedMap {
	defaults := jsonx.Obj()
	for _, c := range TargetContexts {
		if b, ok := t.DefaultDiabeticBand(c); ok {
			defaults.Set(c, jsonx.Obj("low_mgdl", numOrNull(b.Low), "high_mgdl", numOrNull(b.High),
				"high_inclusive", b.HighInclusive))
		} else {
			defaults.Set(c, nil)
		}
	}
	items := []any{}
	for _, c := range TargetContexts {
		row, ok := targets[c]
		if !ok {
			continue
		}
		items = append(items, jsonx.Obj("context", c, "low_mgdl", numOrNull(row.Low), "high_mgdl", numOrNull(row.High),
			"high_inclusive", c != TargetPostMeal, "set_by", row.SetBy, "note", strNull(row.Note),
			"updated_at", iso(row.UpdatedAt)))
	}
	return jsonx.Obj(
		"diabetic", diabetic,
		"contexts", TargetContexts,
		"bounds", jsonx.Obj("min_mgdl", MinTargetMgDl, "max_mgdl", MaxTargetMgDl),
		"defaults", defaults,
		"source", clinicalconfig.SourceOf(clinicalconfig.KeyGlucoseDiabetic, t.On),
		"items", items,
	)
}

// targetsOn is today's thresholds of a targets payload.
func targetsOn(now time.Time) Thresholds { return ThresholdsOn(civildate.InTehran(now)) }
