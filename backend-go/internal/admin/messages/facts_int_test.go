package messages_test

// T-M10-03 (D-161): GET /messages/facts (the registry for the editor, golden in a stable order) and the 422 codes of
// rules naming an unknown fact or a value that does not fit the fact.

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var updateFacts = flag.Bool("update", false, "rewrite testdata/facts_registry.json from the endpoint")

func TestFactsInt_RegistryGolden(t *testing.T) {
	e := rulesEnv(t)
	r := e.As(marketingID).Get("/messages/facts")
	require.Equal(t, 200, r.Status, string(r.Raw))

	var out bytes.Buffer
	require.NoError(t, json.Indent(&out, mustMarshal(t, r.Body["data"]), "", "  "))
	out.WriteByte('\n')
	golden := filepath.Join("testdata", "facts_registry.json")
	if *updateFacts {
		require.NoError(t, os.WriteFile(golden, out.Bytes(), 0o600))
	}
	want, err := os.ReadFile(golden)
	require.NoError(t, err, "run with -update to record the golden")
	assert.Equal(t, string(want), out.String())

	// Stable order: by name; text variables are not listed.
	items, _ := r.Data()["facts"].([]any)
	var names []string
	for _, it := range items {
		names = append(names, it.(map[string]any)["name"].(string))
	}
	assert.IsIncreasing(t, names)
	assert.NotContains(t, names, "name")
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

func TestFactsInt_RuleValidationCodes(t *testing.T) {
	e := rulesEnv(t)
	med := e.As(medicalID)
	with := func(cond map[string]any) map[string]any {
		r := clinicalRule()
		r["conditions"] = []any{map[string]any{"op": "and", "items": []any{cond}}}
		return r
	}
	r := med.JSON(fiber.MethodPost, "/messages/rules", with(map[string]any{"fact": "bp", "op": "gt", "value": 180}))
	assert.Equal(t, 422, r.Status)
	assert.Equal(t, "unknown_fact", r.Code())
	assert.Contains(t, r.Errors(), "conditions")

	r = med.JSON(fiber.MethodPost, "/messages/rules", with(map[string]any{"fact": "main_phase", "op": "between", "value": []any{1, 2}}))
	assert.Equal(t, 422, r.Status)
	assert.Equal(t, "bad_value", r.Code())

	r = med.JSON(fiber.MethodPost, "/messages/rules", with(map[string]any{"fact": "confidence", "op": "eq", "value": "very_high"}))
	assert.Equal(t, 422, r.Status)
	assert.Equal(t, "bad_value", r.Code())

	// Other editor errors keep validation_failed; a valid registry condition saves.
	bad := clinicalRule()
	bad["title"] = ""
	assert.Equal(t, "validation_failed", med.JSON(fiber.MethodPost, "/messages/rules", bad).Code())
	ok := med.JSON(fiber.MethodPost, "/messages/rules", with(map[string]any{"fact": "days_late", "op": "between", "value": []any{8, 30}}))
	assert.Equal(t, 201, ok.Status, string(ok.Raw))
}
