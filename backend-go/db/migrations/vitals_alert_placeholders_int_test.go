package migrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const vitalsAlertPlaceholdersVersion int64 = 116 // 00116_vitals_alert_threshold_placeholders.sql

// 00116 (T-M10-19b): the seeded vitals_alert copy loses its threshold numbers for {systolic}/{diastolic}/{mgdl}/
// {mmol_l}; a row an admin rewrote is left alone, and Down puts the seeded numbers back.
func TestVitalsAlertPlaceholdersMigration(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	_, err = p.DownTo(ctx, vitalsAlertPlaceholdersVersion-1)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	field := func(item, locale, path string) string { return msgField(t, db, "vitals_alert", item, locale, path) }
	require.Contains(t, field("bp_crisis", "fa", "$.what_we_saw"), "بالاتر از ۱۸۰/۱۲۰", "00035 seed")
	require.Contains(t, field("glucose_low", "en", "$.what_we_saw"), "below 54 mg/dL (3.0 mmol/L)", "00035 seed")

	// An admin rewrite of the en glucose sentence survives Up and Down.
	const custom = "You logged {value}; that is very low."
	exec(t, db, "UPDATE message_contents SET payload = JSON_SET(payload, '$.what_we_saw', ?) "+
		"WHERE `group` = 'vitals_alert' AND item_key = 'glucose_low' AND locale = 'en'", custom)

	_, err = p.UpByOne(ctx)
	require.NoError(t, err)
	assert.Equal(t, "فشار {value} ثبت کردی؛ عدد بالاتر از {systolic}/{diastolic} اورژانسی حساب می‌شود.",
		field("bp_crisis", "fa", "$.what_we_saw"))
	assert.Contains(t, field("bp_crisis", "fa", "$.advice"), "اگر باز هم بالاتر از {systolic}/{diastolic} بود")
	assert.Equal(t, "You logged {value}; a reading above {systolic}/{diastolic} counts as an emergency.",
		field("bp_crisis", "en", "$.what_we_saw"))
	assert.Contains(t, field("bp_crisis", "en", "$.advice"), "If it is still above {systolic}/{diastolic},")
	assert.Equal(t, "قند {value} ثبت کردی؛ کمتر از {mgdl} mg/dL ({mmol_l} mmol/L) افت قند شدید حساب می‌شود.",
		field("glucose_low", "fa", "$.what_we_saw"))
	assert.Equal(t, custom, field("glucose_low", "en", "$.what_we_saw"), "admin edit kept")
	assert.Equal(t, "115", field("bp_crisis", "fa", "$.actions[0].phone"), "the emergency number is not a threshold")

	_, err = p.DownTo(ctx, vitalsAlertPlaceholdersVersion-1)
	require.NoError(t, err)
	assert.Contains(t, field("bp_crisis", "fa", "$.what_we_saw"), "بالاتر از ۱۸۰/۱۲۰")
	assert.Contains(t, field("bp_crisis", "en", "$.advice"), "If it is still above 180/120,")
	assert.Contains(t, field("glucose_low", "fa", "$.what_we_saw"), "کمتر از ۵۴ mg/dL (۳٫۰ mmol/L)")
	assert.Equal(t, custom, field("glucose_low", "en", "$.what_we_saw"))
}
