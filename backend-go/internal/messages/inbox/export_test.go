package inbox

import dto "github.com/prometheus/client_model/go"

// MissingEvidenceValue is the current inbox_missing_evidence_total (tests).
func MissingEvidenceValue() float64 {
	var m dto.Metric
	if missingEvidence.Write(&m) != nil {
		return -1
	}
	return m.GetCounter().GetValue()
}
