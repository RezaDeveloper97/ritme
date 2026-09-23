// Package model is App\Models\DailyHealthLog for the Go API: the row (as sqlc scans it), its
// casts, DailyHealthLog::toArray() and the typed field accessors the engines read.
//
// Every package that selects `SELECT * FROM daily_health_logs` through its own sqlc package
// gets a struct with the same fields, so it converts directly:
//
//	l := model.FromRow(store.DailyHealthLog(cycleStoreRow)) // healthlog/store.DailyHealthLog
//	l.ToArray()                                             // DailyHealthLog::toArray()
//	enums.RecommendationTriggerActiveFor(l)                 // l is an enums.TriggerLog
//
// Casts (DailyHealthLog::casts()): log_date date:Y-m-d; ~20 booleans; weight and
// basal_body_temperature decimal:2, blood_sugar decimal:1 (JSON strings); heart rate,
// pressures and exercise_duration integer; moods, exercise_type, sexual_activities and
// medications array (json_decode($v, true)); created_at / updated_at datetime.
package model

import (
	"database/sql"
	"encoding/json"

	rootdb "github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Kind is the Eloquent cast of a data column.
type Kind uint8

// Column kinds.
const (
	String  Kind = iota // no cast (enum values, notes)
	Bool                // boolean
	Int                 // integer
	Decimal             // decimal:N (Column.Scale)
	Array               // array (JSON column)
)

// Column is one fillable data column of daily_health_logs (everything except id, user_id,
// log_date and the timestamps).
type Column struct {
	Name  string
	Kind  Kind
	Scale int // decimal places for Decimal
	field func(r *store.DailyHealthLog) any
}

// Field returns a pointer to the column's field in r (*sql.NullString, *sql.NullBool,
// *sql.NullInt16 or *db.NullRawJSON).
func (c Column) Field(r *store.DailyHealthLog) any { return c.field(r) }

// Columns are the data columns in table order (= the toArray() key order between log_date
// and created_at).
var Columns = []Column{
	{"bleeding_intensity", String, 0, func(r *store.DailyHealthLog) any { return &r.BleedingIntensity }},
	{"blood_color", String, 0, func(r *store.DailyHealthLog) any { return &r.BloodColor }},
	{"has_clots", Bool, 0, func(r *store.DailyHealthLog) any { return &r.HasClots }},
	{"clots_amount", String, 0, func(r *store.DailyHealthLog) any { return &r.ClotsAmount }},
	{"spotting", Bool, 0, func(r *store.DailyHealthLog) any { return &r.Spotting }},
	{"bleeding_smell", String, 0, func(r *store.DailyHealthLog) any { return &r.BleedingSmell }},
	{"headache_intensity", String, 0, func(r *store.DailyHealthLog) any { return &r.HeadacheIntensity }},
	{"stomach_ache_intensity", String, 0, func(r *store.DailyHealthLog) any { return &r.StomachAcheIntensity }},
	{"pelvic_pain_intensity", String, 0, func(r *store.DailyHealthLog) any { return &r.PelvicPainIntensity }},
	{"breast_pain_intensity", String, 0, func(r *store.DailyHealthLog) any { return &r.BreastPainIntensity }},
	{"back_pain_intensity", String, 0, func(r *store.DailyHealthLog) any { return &r.BackPainIntensity }},
	{"ovarian_pain_intensity", String, 0, func(r *store.DailyHealthLog) any { return &r.OvarianPainIntensity }},
	{"nausea_intensity", String, 0, func(r *store.DailyHealthLog) any { return &r.NauseaIntensity }},
	{"bloating_intensity", String, 0, func(r *store.DailyHealthLog) any { return &r.BloatingIntensity }},
	{"diarrhea", Bool, 0, func(r *store.DailyHealthLog) any { return &r.Diarrhea }},
	{"constipation", Bool, 0, func(r *store.DailyHealthLog) any { return &r.Constipation }},
	{"appetite_change", String, 0, func(r *store.DailyHealthLog) any { return &r.AppetiteChange }},
	{"food_craving", Bool, 0, func(r *store.DailyHealthLog) any { return &r.FoodCraving }},
	{"breast_sensitivity_intensity", String, 0, func(r *store.DailyHealthLog) any { return &r.BreastSensitivityIntensity }},
	{"vaginal_dryness", Bool, 0, func(r *store.DailyHealthLog) any { return &r.VaginalDryness }},
	{"vaginal_burning", Bool, 0, func(r *store.DailyHealthLog) any { return &r.VaginalBurning }},
	{"vaginal_burning_intensity", String, 0, func(r *store.DailyHealthLog) any { return &r.VaginalBurningIntensity }},
	{"vaginal_itching", Bool, 0, func(r *store.DailyHealthLog) any { return &r.VaginalItching }},
	{"vaginal_itching_intensity", String, 0, func(r *store.DailyHealthLog) any { return &r.VaginalItchingIntensity }},
	{"vaginal_smell_change", Bool, 0, func(r *store.DailyHealthLog) any { return &r.VaginalSmellChange }},
	{"urination_change", String, 0, func(r *store.DailyHealthLog) any { return &r.UrinationChange }},
	{"urination_burning_intensity", String, 0, func(r *store.DailyHealthLog) any { return &r.UrinationBurningIntensity }},
	{"acne", Bool, 0, func(r *store.DailyHealthLog) any { return &r.Acne }},
	{"oily_skin", Bool, 0, func(r *store.DailyHealthLog) any { return &r.OilySkin }},
	{"hair_loss", Bool, 0, func(r *store.DailyHealthLog) any { return &r.HairLoss }},
	{"swelling", Bool, 0, func(r *store.DailyHealthLog) any { return &r.Swelling }},
	{"fatigue", Bool, 0, func(r *store.DailyHealthLog) any { return &r.Fatigue }},
	{"dizziness", Bool, 0, func(r *store.DailyHealthLog) any { return &r.Dizziness }},
	{"hot_flashes", Bool, 0, func(r *store.DailyHealthLog) any { return &r.HotFlashes }},
	{"chills", Bool, 0, func(r *store.DailyHealthLog) any { return &r.Chills }},
	{"moods", Array, 0, func(r *store.DailyHealthLog) any { return &r.Moods }},
	{"sleep_duration", String, 0, func(r *store.DailyHealthLog) any { return &r.SleepDuration }},
	{"sleep_quality", String, 0, func(r *store.DailyHealthLog) any { return &r.SleepQuality }},
	{"exercise_type", Array, 0, func(r *store.DailyHealthLog) any { return &r.ExerciseType }},
	{"exercise_duration", Int, 0, func(r *store.DailyHealthLog) any { return &r.ExerciseDuration }},
	{"exercise_intensity", String, 0, func(r *store.DailyHealthLog) any { return &r.ExerciseIntensity }},
	{"sexual_activities", Array, 0, func(r *store.DailyHealthLog) any { return &r.SexualActivities }},
	{"sexual_desire", String, 0, func(r *store.DailyHealthLog) any { return &r.SexualDesire }},
	{"intercourse_type", String, 0, func(r *store.DailyHealthLog) any { return &r.IntercourseType }},
	{"weight", Decimal, 2, func(r *store.DailyHealthLog) any { return &r.Weight }},
	{"basal_body_temperature", Decimal, 2, func(r *store.DailyHealthLog) any { return &r.BasalBodyTemperature }},
	{"heart_rate", Int, 0, func(r *store.DailyHealthLog) any { return &r.HeartRate }},
	{"systolic_pressure", Int, 0, func(r *store.DailyHealthLog) any { return &r.SystolicPressure }},
	{"diastolic_pressure", Int, 0, func(r *store.DailyHealthLog) any { return &r.DiastolicPressure }},
	{"blood_sugar", Decimal, 1, func(r *store.DailyHealthLog) any { return &r.BloodSugar }},
	{"energy_level", String, 0, func(r *store.DailyHealthLog) any { return &r.EnergyLevel }},
	{"discharge_color", String, 0, func(r *store.DailyHealthLog) any { return &r.DischargeColor }},
	{"discharge_texture", String, 0, func(r *store.DailyHealthLog) any { return &r.DischargeTexture }},
	{"discharge_amount", String, 0, func(r *store.DailyHealthLog) any { return &r.DischargeAmount }},
	{"discharge_smell", String, 0, func(r *store.DailyHealthLog) any { return &r.DischargeSmell }},
	{"discharge_itching", Bool, 0, func(r *store.DailyHealthLog) any { return &r.DischargeItching }},
	{"discharge_burning", Bool, 0, func(r *store.DailyHealthLog) any { return &r.DischargeBurning }},
	{"frequent_urination", Bool, 0, func(r *store.DailyHealthLog) any { return &r.FrequentUrination }},
	{"medications", Array, 0, func(r *store.DailyHealthLog) any { return &r.Medications }},
	{"notes", String, 0, func(r *store.DailyHealthLog) any { return &r.Notes }},
}

var columnIndex = func() map[string]int {
	m := make(map[string]int, len(Columns))
	for i, c := range Columns {
		m[c.Name] = i
	}
	return m
}()

// ColumnByName returns the data column called name.
func ColumnByName(name string) (Column, bool) {
	i, ok := columnIndex[name]
	if !ok {
		return Column{}, false
	}
	return Columns[i], true
}

// DailyHealthLog is one daily_health_logs row with the model's casts and accessors.
type DailyHealthLog struct {
	Row store.DailyHealthLog
}

// FromRow wraps a scanned row.
func FromRow(r store.DailyHealthLog) *DailyHealthLog { return &DailyHealthLog{Row: r} }

// ToArray is DailyHealthLog::toArray(): every column in table order with the casts applied.
func (l *DailyHealthLog) ToArray() *jsonx.OrderedMap { return l.Attributes(nil) }

// MarshalJSON encodes ToArray().
func (l *DailyHealthLog) MarshalJSON() ([]byte, error) { return jsonx.Marshal(l.ToArray(), 0) }

// Attributes is toArray() restricted to keys, in that order (nil = every column in table
// order). A model created with only some attributes (Model::create) serialises this way.
// Unknown keys are skipped.
func (l *DailyHealthLog) Attributes(keys []string) *jsonx.OrderedMap {
	if keys == nil {
		keys = make([]string, 0, len(Columns)+5)
		keys = append(keys, "id", "user_id", "log_date")
		for _, c := range Columns {
			keys = append(keys, c.Name)
		}
		keys = append(keys, "created_at", "updated_at")
	}
	out := jsonx.NewObject()
	for _, k := range keys {
		if v, ok := l.Value(k); ok {
			out.Set(k, v)
		}
	}
	return out
}

// Value is $log->toArray()[$key]: the cast JSON value of one column.
func (l *DailyHealthLog) Value(key string) (any, bool) {
	r := &l.Row
	switch key {
	case "id":
		return r.ID, true
	case "user_id":
		return r.UserID, true
	case "log_date":
		return r.LogDate, true
	case "created_at":
		return nullDateTime(r.CreatedAt), true
	case "updated_at":
		return nullDateTime(r.UpdatedAt), true
	}
	c, ok := ColumnByName(key)
	if !ok {
		return nil, false
	}
	return CastValue(c, c.Field(r)), true
}

// CastValue is the cast JSON value of a column field (a pointer returned by Column.Field).
func CastValue(c Column, field any) any {
	switch f := field.(type) {
	case *sql.NullString:
		if !f.Valid {
			return nil
		}
		if c.Kind == Decimal {
			d, err := jsonx.Decimal(f.String, c.Scale)
			if err != nil { // not reachable for a DECIMAL column
				return f.String
			}
			return d
		}
		return f.String
	case *sql.NullBool:
		if !f.Valid {
			return nil
		}
		return f.Bool
	case *sql.NullInt16:
		if !f.Valid {
			return nil
		}
		return int64(f.Int16)
	case *rootdb.NullRawJSON:
		if !f.Valid {
			return nil
		}
		return DecodeArray(f.V)
	}
	return nil
}

// DecodeArray is the `array` cast read side: json_decode($raw, true) — objects become PHP
// arrays (an empty or 0..n-1-keyed object re-encodes as a list), invalid JSON is null.
func DecodeArray(raw json.RawMessage) any {
	v, err := phpval.Decode(raw)
	if err != nil {
		return nil
	}
	return phpval.Packed(v)
}

func nullDateTime(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return jsonx.DateTime(t.Time)
}

// ---------------------------------------------------------------------------
// Typed accessors ($log->column with the cast applied). Nil means NULL.

// Str returns a String column (nil when NULL or not a string column).
func (l *DailyHealthLog) Str(name string) *string {
	c, ok := ColumnByName(name)
	if !ok {
		return nil
	}
	if f, ok := c.Field(&l.Row).(*sql.NullString); ok && f.Valid {
		s := f.String
		return &s
	}
	return nil
}

// Bool returns a boolean column (nil when NULL).
func (l *DailyHealthLog) Bool(name string) *bool {
	c, ok := ColumnByName(name)
	if !ok {
		return nil
	}
	if f, ok := c.Field(&l.Row).(*sql.NullBool); ok && f.Valid {
		b := f.Bool
		return &b
	}
	return nil
}

// Int returns an integer column (nil when NULL).
func (l *DailyHealthLog) Int(name string) *int64 {
	c, ok := ColumnByName(name)
	if !ok {
		return nil
	}
	if f, ok := c.Field(&l.Row).(*sql.NullInt16); ok && f.Valid {
		n := int64(f.Int16)
		return &n
	}
	return nil
}

// Strings returns an array column as a list of strings: nil when NULL or not a list;
// non-string elements are skipped (moods, exercise_type, sexual_activities).
func (l *DailyHealthLog) Strings(name string) []string {
	c, ok := ColumnByName(name)
	if !ok || c.Kind != Array {
		return nil
	}
	list, ok := CastValue(c, c.Field(&l.Row)).([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, x := range list {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// enums.TriggerLog (RecommendationTrigger::matches).
var _ enums.TriggerLog = (*DailyHealthLog)(nil)

// HeadacheIntensity implements enums.TriggerLog.
func (l *DailyHealthLog) HeadacheIntensity() *string { return l.Str("headache_intensity") }

// PelvicPainIntensity implements enums.TriggerLog.
func (l *DailyHealthLog) PelvicPainIntensity() *string { return l.Str("pelvic_pain_intensity") }

// StomachAcheIntensity implements enums.TriggerLog.
func (l *DailyHealthLog) StomachAcheIntensity() *string { return l.Str("stomach_ache_intensity") }

// SleepQuality implements enums.TriggerLog.
func (l *DailyHealthLog) SleepQuality() *string { return l.Str("sleep_quality") }

// Moods implements enums.TriggerLog.
func (l *DailyHealthLog) Moods() []string { return l.Strings("moods") }

// BloatingIntensity implements enums.TriggerLog.
func (l *DailyHealthLog) BloatingIntensity() *string { return l.Str("bloating_intensity") }

// Fatigue implements enums.TriggerLog.
func (l *DailyHealthLog) Fatigue() *bool { return l.Bool("fatigue") }

// BleedingIntensity is $log->bleeding_intensity.
func (l *DailyHealthLog) BleedingIntensity() *string { return l.Str("bleeding_intensity") }

// Spotting is $log->spotting.
func (l *DailyHealthLog) Spotting() *bool { return l.Bool("spotting") }
