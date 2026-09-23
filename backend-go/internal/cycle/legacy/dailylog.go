package legacy

// DailyLog is the part of a daily_health_logs row (after the model's casts) the legacy engine
// reads, plus the serialized model for `source_daily_log_data`. Pointers are nil for NULL; list
// fields are nil when the column is NULL or not a JSON array (PHP `is_array` false).
type DailyLog struct {
	Spotting             *bool
	VaginalDryness       *bool
	Fatigue              *bool
	DischargeTexture     *string
	OvarianPainIntensity *string
	BloatingIntensity    *string
	HeadacheIntensity    *string
	PelvicPainIntensity  *string
	StomachAcheIntensity *string
	SleepQuality         *string
	SexualDesire         *string
	SexualActivities     []string
	Moods                []string

	// Source is DailyHealthLog::toArray() — emitted verbatim as `source_daily_log_data`. Any value
	// that marshals to the Eloquent model JSON (e.g. the health-log resource, a json.RawMessage).
	Source any
}

// triggerLog adapts a DailyLog to enums.TriggerLog (RecommendationTrigger::matches reads).
type triggerLog struct{ *DailyLog }

func (l triggerLog) HeadacheIntensity() *string    { return l.DailyLog.HeadacheIntensity }
func (l triggerLog) PelvicPainIntensity() *string  { return l.DailyLog.PelvicPainIntensity }
func (l triggerLog) StomachAcheIntensity() *string { return l.DailyLog.StomachAcheIntensity }
func (l triggerLog) SleepQuality() *string         { return l.DailyLog.SleepQuality }
func (l triggerLog) Moods() []string               { return l.DailyLog.Moods }
func (l triggerLog) BloatingIntensity() *string    { return l.DailyLog.BloatingIntensity }
func (l triggerLog) Fatigue() *bool                { return l.DailyLog.Fatigue }

func isTrue(b *bool) bool { return b != nil && *b }

func eq(s *string, v string) bool { return s != nil && *s == v }
