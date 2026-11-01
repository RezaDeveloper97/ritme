package inboxalerts_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/messages/pregnancyalerts"
	"github.com/ritme/backend-go/internal/messages/pregnancyalerts/inboxalerts"
	"github.com/ritme/backend-go/internal/messages/rules"
)

// SPEC §4.2: every safety rule is locked, urgent, and its code-held copy renders in fa and en with the evidence and
// the contact line — what the inbox serves when the row is deleted or inactive.
func TestSafetyRules_LockedUrgentWithCodeHeldCopy(t *testing.T) {
	for _, d := range inboxalerts.Defs {
		if d.Kind != rules.KindSafety {
			continue
		}
		assert.True(t, d.Locked, d.Key)
		assert.Equal(t, rules.LevelFollowSoon, d.Level, d.Key)
		for _, loc := range inboxalerts.Languages() {
			c, ok := inboxalerts.CopyOf(d.Key, loc)
			require.True(t, ok, "%s %s", d.Key, loc)
			for name, txt := range map[string]string{"title": c.Title, "body": c.Body, "seen": c.Seen, "confidence": c.Confidence, "contact": c.Contact} {
				assert.NotEmpty(t, txt, "%s %s %s", d.Key, loc, name)
			}
		}
	}
	// Every rule has copy in every language, a kind allowed in alerts_inbox, a level its kind allows and registered
	// condition facts (what the admin editor checks on save).
	for _, d := range inboxalerts.Defs {
		for _, loc := range inboxalerts.Languages() {
			_, ok := inboxalerts.CopyOf(d.Key, loc)
			assert.True(t, ok, "%s %s", d.Key, loc)
		}
		assert.True(t, rules.KindAllowedIn(rules.SlotAlertsInbox, d.Kind), d.Key)
		assert.True(t, rules.LevelAllowed(d.Level, d.Kind), "%s: %s × %s", d.Key, d.Level, d.Kind)
		for _, g := range d.Groups {
			for _, c := range g.Items {
				msg, code := rules.CheckFact(c)
				assert.Empty(t, code, "%s: %s", d.Key, msg)
			}
		}
	}
}

// The reduced kick count fires urgent in fa and en with the 115 contact from the code-held definition.
func TestFire_KicksRenderLocalized(t *testing.T) {
	f := pregnancy(30, 2)
	start := today.TehranMidnight().Add(9 * time.Hour)
	f.Kicks = []pregnancyalerts.Kick{{ID: 1, Start: start, End: start.Add(2 * time.Hour), Count: 4}}
	in := &facts.PregnancyAlertInput{Facts: f}
	en, ok := inboxalerts.Fires("preg.fetal_movement_kicks", in, "en")
	require.True(t, ok)
	assert.Equal(t, "Fewer than 10 movements in 120 minutes", en.Title)
	assert.Contains(t, en.Seen, "you logged 4 movements")
	assert.Contains(t, en.Ask, "115")
	fa, ok := inboxalerts.Fires("preg.fetal_movement_kicks", in, "fa")
	require.True(t, ok)
	assert.Contains(t, fa.Title, "۱۲۰")
	assert.Contains(t, fa.Seen, "۴ حرکت")
	assert.Contains(t, fa.Ask, "۱۱۵")
}

func TestV1Trigger(t *testing.T) {
	assert.Equal(t, `{"systolic":165,"diastolic":112,"severe":true}`,
		string(must(inboxalerts.V1Trigger("preg.bp_severe", map[string]any{"preg_bp_sys": "165", "preg_bp_dia": "112"}).MarshalJSON())))
	assert.Equal(t, `{"fasting":"98.00","post_meal":null}`,
		string(must(inboxalerts.V1Trigger("preg.gdm", map[string]any{"preg_fasting": "98", "preg_post_meal": "—"}).MarshalJSON())))
	assert.Equal(t, `{"bleeding":true}`,
		string(must(inboxalerts.V1Trigger("preg.critical_symptom", map[string]any{"preg_symptoms": []any{"bleeding"}}).MarshalJSON())))
	assert.Equal(t, `[]`, string(must(inboxalerts.V1Trigger("preg.vomiting", nil).MarshalJSON())))
}

func must(b []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return b
}
