package modules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModuleOf(t *testing.T) {
	cases := map[string]string{
		"/api/v1/companions":                        Companion,
		"/api/v1/companions/12":                     Companion,
		"/api/v1/companion/home":                    Companion,
		"/api/v1/fertility/today":                   TTC,
		"/api/v1/analysis/ttc":                      TTC,
		"/api/v1/ivf/cycles/current":                IVF,
		"/api/v1/menopause/today":                   Menopause,
		"/api/v1/teen/profile":                      Teen,
		"/api/v1/postpartum/epds":                   Postpartum,
		"/api/v1/children/3/feeds":                  Children,
		"/api/v1/contraception/pills/2026-10":       Contraception,
		"/api/v1/companionship":                     "",
		"/api/v1/profile/life-stage":                "",
		"/api/v1/analysis/cycle":                    "",
		"/api/v1/modules":                           "",
		"/api/v1/conditions/pain":                   Programs,
		"/api/v1/pelvic/sessions":                   Programs,
		"/api/v1/city-services/listings":            MotherChild,
		"/api/v1/learning/courses":                  Learning,
		"/api/v1/shop/cart":                         Shop,
		"/api/v1/insurance/claims":                  Insurance,
		"/api/v1/health-record/documents":           Record,
		"/api/v1/health-record/emergency-card":      "",
		"/api/v1/health-record/emergency-card/lock": "",
		"/api/v1/telemed/doctors/3":                 Doctors,
		"/api/v1/telemed/bookings":                  Doctors,
		"/api/v1/todo/tasks/9":                      Todo,
		"/api/v1/todos":                             "",
		"/api/v1/care/today":                        "",
		"/api/v1/reminders":                         "",
	}
	for path, want := range cases {
		assert.Equal(t, want, ModuleOf(path), path)
	}
}

func TestStates_MissingKeyIsOn(t *testing.T) {
	s := States{TTC: false}
	assert.False(t, s.Enabled(TTC))
	assert.True(t, s.Enabled(Teen))
	j := s.JSON()
	assert.Len(t, j, len(Keys))
	assert.False(t, j[TTC])
	assert.True(t, j[Companion])
}
