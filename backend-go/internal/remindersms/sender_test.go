package remindersms

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/platform/config"
)

func TestToken(t *testing.T) {
	assert.Equal(t, "آهن", Token("  آهن ", 4))
	assert.Equal(t, "a b c d e", Token("a b  c d e f g", 4))
	assert.Equal(t, "-", Token("   ", 4))
	assert.Len(t, []rune(Token(string(make([]rune, 0))+"ابپتثجچحخدذرزژسشصضطظعغفقکگلمنوهیابپتثجچحخدذرزژسشصضطظعغفقکگلمنوهیابپت", 4)), 60)
}

func TestTemplate(t *testing.T) {
	cfg := config.ReminderSMS{MedicationTemplate: "med", AppointmentTemplate: "appt", NeutralTemplate: "neutral"}
	med := SMS{Kind: KindMedication, Name: "قرص آهن", Clock: "08:00"}
	appt := SMS{Kind: KindAppointment, Name: "دکتر احمدی", Clock: "10:30", When: "فردا ساعت 10:30"}

	tpl, tokens, ok := Template(cfg, med)
	assert.True(t, ok)
	assert.Equal(t, "med", tpl)
	assert.Equal(t, map[string]string{"token": "08:00", "token10": "قرص آهن"}, tokens)

	tpl, tokens, ok = Template(cfg, appt)
	assert.True(t, ok)
	assert.Equal(t, "appt", tpl)
	assert.Equal(t, map[string]string{"token": "10:30", "token10": "دکتر احمدی", "token20": "فردا ساعت 10:30"}, tokens)

	// Discreet: the neutral template and nothing but the time.
	med.Discreet = true
	tpl, tokens, ok = Template(cfg, med)
	assert.True(t, ok)
	assert.Equal(t, "neutral", tpl)
	assert.Equal(t, map[string]string{"token": "08:00"}, tokens)

	// Discreet without a neutral template: never the regular wording.
	cfg.NeutralTemplate = ""
	_, _, ok = Template(cfg, med)
	assert.False(t, ok)

	_, _, ok = Template(cfg, SMS{Kind: "other"})
	assert.False(t, ok)
}
