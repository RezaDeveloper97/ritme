package alerts

import "github.com/ritme/backend-go/internal/platform/jsonx"

// Bleeding logged as `flow` in pregnancy (T-M9-28, DECISIONS.md I-02, SPEC v6 §1.4): the general day log
// (POST /health-logs, PUT /logs/days/{date}) never turns it into a period while the effective mode is pregnancy; it
// raises the existing v1 bleeding alert instead (symptomAlert "bleeding": the same copy as a bleeding symptom log).

// flowSeverity maps daily_health_logs.bleeding_intensity (low · medium · high · very_high) onto the symptom severity
// the bleeding rule reads: heavy flow is "severe" (the emergency branch), the rest a warning. "" = no flow.
func flowSeverity(intensity string) string {
	switch intensity {
	case "high", "very_high":
		return "severe"
	case "medium":
		return "moderate"
	case "low":
		return "mild"
	}
	return ""
}

// FlowRank orders flow intensities by the severity they are read as: 0 none, 1 low / medium, 2 heavy. The day log
// raises a new alert only when the rank of the day goes up, so a re-save (or a lighter value) never alerts twice.
func FlowRank(intensity string) int {
	switch flowSeverity(intensity) {
	case "":
		return 0
	case "severe":
		return 2
	}
	return 1
}

// ForLoggedFlow is the alert of a day-log `flow` value in pregnancy (ok false without flow). The trigger records the
// logged intensity next to the severity it was read as.
func ForLoggedFlow(ctx Context, intensity string) (Draft, bool) {
	sev := flowSeverity(intensity)
	if sev == "" {
		return Draft{}, false
	}
	d, ok := ctx.symptomAlert("bleeding", sev)
	if !ok {
		return Draft{}, false
	}
	d.Trigger = jsonx.Obj("bleeding", sev, "flow", intensity, "source", "day_log")
	return d, true
}
