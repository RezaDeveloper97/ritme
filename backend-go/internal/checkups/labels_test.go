package checkups

import (
	"encoding/json"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/checkups/engine"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// flatten lists the dotted keys of a decoded lang file (arrays are leaves).
func flatten(prefix string, v any, out map[string]bool) {
	m, ok := v.(map[string]any)
	if !ok {
		out[prefix] = true
		return
	}
	for k, x := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		flatten(key, x, out)
	}
}

// Every shipped language has exactly the English keys, twelve month names and ten digits.
func TestLangFiles_SameKeys(t *testing.T) {
	files, err := fs.Glob(langFS, "lang/*/checkups.json")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(files), 2)
	keys := map[string]map[string]bool{}
	for _, f := range files {
		b, err := langFS.ReadFile(f)
		require.NoError(t, err)
		var v map[string]any
		require.NoError(t, json.Unmarshal(b, &v), f)
		keys[f] = map[string]bool{}
		flatten("", v, keys[f])
		assert.Len(t, v["months"], 12, f)
		assert.Len(t, []rune(v["digits"].(string)), 10, f)
		assert.Contains(t, []string{"jalali", "gregorian"}, v["calendar"], f)
	}
	for f, k := range keys {
		assert.Equal(t, keys["lang/en/checkups.json"], k, f)
	}
}

func TestLabels(t *testing.T) {
	assert.Equal(t, "هر ماه", IntervalLabel(1, 0, "fa"))
	assert.Equal(t, "هر ۶ ماه", IntervalLabel(6, 0, "fa"))
	assert.Equal(t, "هر سال", IntervalLabel(12, 0, "fa"))
	assert.Equal(t, "هر ۳ سال", IntervalLabel(36, 0, "fa"))
	assert.Equal(t, "هر ۱۸ ماه", IntervalLabel(18, 0, "fa"))
	assert.Equal(t, "هر ۱ تا ۲ سال", IntervalLabel(12, 24, "fa"))
	assert.Equal(t, "Every 6–9 months", IntervalLabel(6, 9, "en"))
	assert.Equal(t, "Every 2 years", IntervalLabel(24, 0, "en"))
	assert.Equal(t, "Every month", IntervalLabel(1, 0, "de"), "unknown language → English")

	today := civildate.New(2026, 9, 23) // 1 Mehr 1405
	assert.Equal(t, "امروز", RelativeDue(today, today, "fa"))
	assert.Equal(t, "فردا", RelativeDue(today.AddDays(1), today, "fa"))
	assert.Equal(t, "۳ روز دیگر", RelativeDue(today.AddDays(3), today, "fa"))
	assert.Equal(t, "In 30 days", RelativeDue(today.AddDays(30), today, "en"))
	assert.Equal(t, "آبان ۱۴۰۵", RelativeDue(today.AddDays(31), today, "fa"))
	assert.Equal(t, "زمانش رسیده", RelativeDue(today.AddDays(-1), today, "fa"))
	assert.Equal(t, "۱ مهر ۱۴۰۵", FullDate(today, "fa"))
	assert.Equal(t, "September 23, 2026", FullDate(today, "en"))
	assert.Equal(t, civildate.New(2026, 3, 21), YearStart(today, "fa"))
	assert.Equal(t, civildate.New(2026, 1, 1), YearStart(today, "en"))

	overdue := engine.Item{Status: engine.StatusOverdue, NextDueOn: civildate.New(2025, 12, 1)}
	assert.Equal(t, "عقب\u200cافتاده از آذر ۱۴۰۴", NextDueLabel(overdue, engine.Type{}, today, "fa"), "another year → with the year")
	assert.Equal(t, "Overdue since December 2025", NextDueLabel(overdue, engine.Type{}, today, "en"))

	assert.Equal(t, "۳ روز قبل یادآوری می\u200cکنیم", ReminderLabel(3, true, "fa"))
	assert.Equal(t, "We'll remind you on the day", ReminderLabel(0, true, "en"))
	assert.Equal(t, "یادآوری خاموش است", ReminderLabel(3, false, "fa"))
}
