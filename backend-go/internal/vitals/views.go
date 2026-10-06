package vitals

import (
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/phpround"
)

func iso(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return jsonx.ISO8601(t.In(civildate.Tehran))
}

func strNull(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func intNull(v float64) any {
	if v == 0 {
		return nil
	}
	return int(phpround.Round(v, 0))
}

// ClassJSON is {code, tone} or null.
func ClassJSON(typ, class string) any {
	if class == "" {
		return nil
	}
	return jsonx.Obj("code", class, "tone", Tone(typ, class))
}

// GlucoseValueJSON is a mg/dL value in both units: {mg_dl, mmol_l}.
func GlucoseValueJSON(mgdl float64) *jsonx.OrderedMap {
	return jsonx.Obj("mg_dl", MgDlRounded(mgdl), "mmol_l", MmolL(mgdl))
}

// ReadingJSON is one reading. Log readings (source "log") have id, measured_at, time and period null.
func ReadingJSON(r Reading) *jsonx.OrderedMap {
	var id, clock any
	if r.ID > 0 {
		id = r.ID
	}
	if r.Timed() {
		clock = r.MeasuredAt.In(civildate.Tehran).Format("15:04")
	}
	var bp, glucose, hr any
	switch r.Type {
	case TypeBP:
		bp = jsonx.Obj("systolic", int(r.Systolic), "diastolic", int(r.Diastolic), "pulse", intNull(r.Pulse),
			"arm", strNull(r.Arm), "position", strNull(r.Position))
	case TypeGlucose:
		glucose = jsonx.Obj("mg_dl", MgDlRounded(r.MgDl), "mmol_l", MmolL(r.MgDl), "value", InUnit(r.MgDl, r.Unit),
			"unit", r.Unit, "context", strNull(r.Context), "method", strNull(r.Method))
	case TypeHR:
		hr = jsonx.Obj("bpm", int(r.Pulse), "context", strNull(r.Context))
	}
	return jsonx.Obj(
		"id", id,
		"source", r.Source,
		"type", r.Type,
		"date", r.Date.String(),
		"time", clock,
		"measured_at", iso(r.MeasuredAt),
		"period", strNull(r.Period()),
		"blood_pressure", bp,
		"glucose", glucose,
		"heart_rate", hr,
		"classification", ClassJSON(r.Type, r.Class()),
		"urgent", r.Urgent(),
		"note", strNull(r.Note),
		"editable", r.Source == SourceVitals,
	)
}

// ReadingsJSON is a list of readings.
func ReadingsJSON(rs []Reading) []any {
	out := make([]any, 0, len(rs))
	for _, r := range rs {
		out = append(out, ReadingJSON(r))
	}
	return out
}

// ThresholdsJSON is GET /vitals/thresholds: the bands the server classifies with, with sources.
func ThresholdsJSON() *jsonx.OrderedMap {
	contexts := jsonx.Obj()
	for _, c := range GlucoseContexts {
		b := GlucoseBands[c]
		contexts.Set(c, jsonx.Obj("target_min", b.TargetMin, "target_max", b.TargetMax, "very_high_from", b.VeryHighFrom))
	}
	return jsonx.Obj(
		"version", ThresholdsVersion,
		"intervals", "min_inclusive_max_exclusive",
		"blood_pressure", jsonx.Obj(
			"unit", "mmhg",
			"source", SourceBP,
			"classes", []any{
				jsonx.Obj("code", ClassNormal, "tone", ToneOK, "rule", "systolic < 120 and diastolic < 80"),
				jsonx.Obj("code", ClassElevated, "tone", ToneWatch, "rule", "systolic 120–129 and diastolic < 80"),
				jsonx.Obj("code", ClassStage1, "tone", ToneHigh, "rule", "systolic 130–139 or diastolic 80–89"),
				jsonx.Obj("code", ClassStage2, "tone", ToneHigh, "rule", "systolic ≥ 140 or diastolic ≥ 90"),
				jsonx.Obj("code", ClassCrisis, "tone", ToneUrgent, "rule", "systolic > 180 and/or diastolic > 120"),
			},
			"elevated_systolic", BPElevatedSystolic,
			"stage1", jsonx.Obj("systolic", BPStage1Systolic, "diastolic", BPStage1Diastolic),
			"stage2", jsonx.Obj("systolic", BPStage2Systolic, "diastolic", BPStage2Diastolic),
			"urgent_above", jsonx.Obj("systolic", BPCrisisSystolic, "diastolic", BPCrisisDiastolic),
		),
		"glucose", jsonx.Obj(
			"unit", UnitMgDl,
			"mmol_factor", MmolFactor,
			"source", SourceGlucose,
			"classes", GlucoseClasses,
			"low_below", GlucoseLowBelow,
			"urgent_below", GlucoseUrgentBelow,
			"contexts", contexts,
		),
		"heart_rate", jsonx.Obj(
			"unit", "bpm",
			"source", SourceHR,
			"classes", HRClasses,
			"resting", jsonx.Obj("min", HRRestingMin, "max", HRRestingMax, "max_inclusive", true),
			"classified_contexts", []string{HRResting, HRAfterWaking},
		),
	)
}
