// Package alerts is what is left of the v1 pregnancy alert rules (PregnancyAlertService,
// backend/app/Services/PregnancyEngine/PregnancyAlertService.php): the medical-history flags of a v1 symptom save's
// alerts and the severity of a day-log `flow` value in pregnancy (flow.go). Every pregnancy alert is a preg.* rule of
// the message engine (T-M10-08, D-165: messages/pregnancyalerts → the alerts inbox); nothing here creates an alert.
//
// Quirk kept: medical_history_flags is `[]` when empty, else an object.
package alerts

import (
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// MedicalHistoryFlags is getMedicalHistoryFlags(): `[]` without a profile or flags,
// else {miscarriage_history, high_risk_history, rh_negative, pre_existing_conditions}.
func MedicalHistoryFlags(p *store.PregnancyProfile) *jsonx.OrderedMap {
	flags := jsonx.NewArray()
	if p == nil {
		return flags
	}
	if p.HasMiscarriageHistory.Valid && p.HasMiscarriageHistory.Bool {
		flags.Set("miscarriage_history", true)
	}
	if p.HasHighRiskHistory.Valid && p.HasHighRiskHistory.Bool {
		flags.Set("high_risk_history", true)
	}
	if p.RhFactor.Valid && p.RhFactor.String == "negative" {
		flags.Set("rh_negative", true)
	}
	if p.PreExistingConditions.Valid {
		if v, err := phpval.Decode(p.PreExistingConditions.V); err == nil && phpval.Truthy(v) {
			flags.Set("pre_existing_conditions", p.PreExistingConditions.V)
		}
	}
	return flags
}
