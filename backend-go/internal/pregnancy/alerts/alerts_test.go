package alerts

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

var (
	yes      = sql.NullBool{Bool: true, Valid: true}
	highRisk = &store.PregnancyProfile{HasMiscarriageHistory: yes, RhFactor: sql.NullString{String: "negative", Valid: true}, PreExistingConditions: db.NullRawJSON{V: json.RawMessage(`["hypertension"]`), Valid: true}}
	lowRisk  = &store.PregnancyProfile{}
)

func TestMedicalHistoryFlags(t *testing.T) {
	enc := func(p *store.PregnancyProfile) string {
		b, err := jsonx.Marshal(MedicalHistoryFlags(p), 0)
		require.NoError(t, err)
		return string(b)
	}
	assert.Equal(t, `[]`, enc(nil))
	assert.Equal(t, `[]`, enc(lowRisk))
	assert.Equal(t, `[]`, enc(&store.PregnancyProfile{PreExistingConditions: db.NullRawJSON{V: json.RawMessage(`[]`), Valid: true}}))
	assert.JSONEq(t, `{"miscarriage_history":true,"rh_negative":true,"pre_existing_conditions":["hypertension"]}`, enc(highRisk))
	assert.JSONEq(t, `{"high_risk_history":true}`, enc(&store.PregnancyProfile{HasHighRiskHistory: yes}))
}
