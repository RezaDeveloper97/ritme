package alerts

// Bleeding logged as `flow` in pregnancy (T-M9-28, DECISIONS.md I-02, SPEC v6 §1.4): the general day log
// (POST /health-logs, PUT /logs/days/{date}) never turns it into a period while the effective mode is pregnancy; it is
// recorded as the day's bleeding symptom and raises the preg.critical_symptom alert of the message engine instead
// (healthlog bridge, T-M10-08b).

// FlowSeverity maps daily_health_logs.bleeding_intensity (low · medium · high · very_high) onto the symptom severity
// of the bleeding symptom: heavy flow is "severe", medium "moderate", low "mild". "" = no flow (or an unknown value).
func FlowSeverity(intensity string) string {
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
// raises the alert only when the rank of the day goes up, so a re-save (or a lighter value) never alerts twice.
func FlowRank(intensity string) int {
	switch FlowSeverity(intensity) {
	case "":
		return 0
	case "severe":
		return 2
	}
	return 1
}

// SeverityRank orders symptom severities (mild < moderate < severe; 0 = none or unknown), so the day log never lowers
// a bleeding severity the user logged herself.
func SeverityRank(severity string) int {
	switch severity {
	case "mild":
		return 1
	case "moderate":
		return 2
	case "severe":
		return 3
	}
	return 0
}
