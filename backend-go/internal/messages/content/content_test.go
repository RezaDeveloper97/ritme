package content

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// TestDefaultsMatchPHPExport re-runs the PHP export and requires the embedded defaults.json to
// be byte-identical (the code fallback copy must never drift from contentDefaults()).
func TestDefaultsMatchPHPExport(t *testing.T) {
	php, err := exec.LookPath("php")
	if err != nil {
		t.Skip("php not installed")
	}
	backend, err := filepath.Abs("../../../../backend")
	require.NoError(t, err)
	if _, err := os.Stat(filepath.Join(backend, "vendor", "autoload.php")); err != nil {
		t.Skip("backend/vendor not installed (composer install)")
	}
	var stderr bytes.Buffer
	cmd := exec.Command(php, "export_defaults.php", backend)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	require.NoError(t, err, stderr.String())
	out = applyDecidedDeviations(out)
	require.True(t, bytes.Equal(out, defaultsJSON), "defaults.json differs from the PHP export — regenerate it (see defaults.go)")
}

// applyDecidedDeviations rewrites the PHP export the way defaults.json deliberately differs from it.
// D-22: the fa BMI copy names the app «ریتمی» where Laravel has the Latin "Ritme" (en unchanged).
func applyDecidedDeviations(export []byte) []byte {
	export = bytes.ReplaceAll(export, []byte("Ritme تلاش"), []byte("ریتمی تلاش"))
	return bytes.ReplaceAll(export, []byte("در Ritme سعی"), []byte("در ریتمی سعی"))
}

// TestDefaultsBmiBrand pins D-22: the brand is Persian in the fa BMI copy and stays "Ritme" in en.
func TestDefaultsBmiBrand(t *testing.T) {
	for _, item := range []string{"normal", "obese"} {
		fa, ok := Default("bmi_message", item, "fa").Field("message")
		require.True(t, ok, item)
		assert.Contains(t, fa, "ریتمی", item)
		assert.NotContains(t, fa, "Ritme", item)
		en, ok := Default("bmi_message", item, "en").Field("message")
		require.True(t, ok, item)
		assert.Contains(t, en, "Ritme", item)
	}
}

func TestDefaultsShape(t *testing.T) {
	assert.Equal(t, []string{
		"cycle_base_non_ttc", "cycle_base_ttc", "cycle_override", "pregnancy_trimester", "pregnancy_week",
		"pregnancy_override", "pregnancy_base", "nutrition_cycle", "nutrition_ttc", "nutrition_symptom",
		"nutrition_trimester", "sleep_cycle", "sleep_symptom", "sleep_trimester", "exercise_cycle",
		"exercise_symptom", "exercise_ttc", "exercise_trimester", "correlation_cycle", "correlation_pregnancy",
		"correlation_common", "pattern", "bmi_message",
	}, Groups())
	assert.Equal(t, []string{"4", "8", "12", "20", "28", "36", "40"}, Items("pregnancy_week"))
	assert.True(t, HasItem("cycle_base_ttc", "default"))
	assert.False(t, HasItem("cycle_base_ttc", "nope"))
	assert.Nil(t, Items("nope"))

	en, _ := Default("cycle_base_non_ttc", "luteal", "en").Field("short")
	fa, _ := Default("cycle_base_non_ttc", "luteal", "fa").Field("short")
	ar, _ := Default("cycle_base_non_ttc", "luteal", "ar").Field("short")
	assert.Equal(t, "Time to focus on yourself and unfinished tasks", en)
	assert.Equal(t, fa, ar, "slice(): a locale without copy falls back to fa")
	_, ok := Default("nope", "x", "en").Field("short")
	assert.False(t, ok)
}

type fakeRows struct {
	rows  map[string][]store.ListLiveMessageContentsRow
	calls int
}

func (f *fakeRows) ListLiveMessageContents(_ context.Context, locale string) ([]store.ListLiveMessageContentsRow, error) {
	f.calls++
	return f.rows[locale], nil
}

func TestRepository(t *testing.T) {
	ctx := context.Background()
	q := &fakeRows{rows: map[string][]store.ListLiveMessageContentsRow{
		"en": {
			{Group: "cycle_base_non_ttc", ItemKey: "luteal", Payload: json.RawMessage(`{"short":"edited","dos":{"0":"a","1":"b"}}`)},
			{Group: "pattern", ItemKey: "ttc_tracking", Payload: json.RawMessage(`not json`)},
			{Group: "pattern", ItemKey: "chronic_fatigue", Payload: json.RawMessage(`null`)},
			{Group: "pattern", ItemKey: "poor_sleep", Payload: json.RawMessage(`"scalar"`)},
		},
	}}
	r := NewRepository(q)

	p, err := r.Resolve(ctx, "cycle_base_non_ttc", "luteal", "en")
	require.NoError(t, err)
	short, _ := p.Field("short")
	assert.Equal(t, "edited", short)
	dos, _ := p.Field("dos")
	assert.Equal(t, []any{"a", "b"}, dos, "a sequential PHP array encodes as a list")
	_, ok := p.Field("long")
	assert.False(t, ok, "the DB payload replaces the fallback entirely")

	// Invalid JSON and JSON null → the code fallback.
	for _, key := range []string{"ttc_tracking", "chronic_fatigue", "severe_pain"} {
		p, err := r.Resolve(ctx, "pattern", key, "en")
		require.NoError(t, err)
		msg, _ := p.Field("message")
		want, _ := Default("pattern", key, "en").Field("message")
		assert.Equal(t, want, msg, key)
	}

	// A scalar payload is PHP's TypeError (array return type) → error.
	_, err = r.Resolve(ctx, "pattern", "poor_sleep", "en")
	require.Error(t, err)

	// profile.MessageContents shape.
	v, ok, err := r.Payload(ctx, "cycle_base_non_ttc", "luteal", "en")
	require.NoError(t, err)
	assert.True(t, ok)
	_, isMap := v.(phpval.Map)
	assert.True(t, isMap)

	assert.Equal(t, 1, q.calls, "one query per locale per request")
	_, _, _ = r.Payload(ctx, "pattern", "x", "fa")
	assert.Equal(t, 2, q.calls)
}
