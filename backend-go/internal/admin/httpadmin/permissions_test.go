package httpadmin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCapabilityOf(t *testing.T) {
	cases := map[string]string{
		Prefix + "/auth/me":                          "",
		Prefix + "/users/:id":                        CapUsers,
		Prefix + "/telemed/doctor/bookings":          CapTelemedDoctor,
		Prefix + "/telemed/doctors/:id":              CapTelemed,
		Prefix + "/pregnancy-care-items":             CapClinical,
		Prefix + "/health-access/users/:id":          CapHealth,
		Prefix + "/something-new":                    "?",
		Prefix + "/challenge-completions/:id/reject": CapChallenges,
	}
	for path, want := range cases {
		assert.Equal(t, want, capabilityOf(path), path)
	}
}

func TestPermissions(t *testing.T) {
	p := Permissions{CapUsers: LevelView, CapContent: LevelManage}
	assert.True(t, p.Allows(CapUsers, LevelView))
	assert.False(t, p.Allows(CapUsers, LevelManage))
	assert.True(t, p.Allows(CapContent, LevelView))
	assert.False(t, p.Allows(CapHealth, LevelView))

	assert.NotContains(t, DefaultPermissions(RoleEditor), CapHealth)
	assert.Equal(t, LevelManage, DefaultPermissions(RoleClinicalSupport)[CapHealth])
	for _, role := range Roles {
		if role == RoleSuper || role == RoleEditor || role == RoleClinicalSupport {
			continue
		}
		assert.NotContains(t, DefaultPermissions(role), CapHealth, role) // health data: clinical support only
	}
	_, ok := ParsePermissions(map[string]string{CapHealth: LevelView})
	assert.False(t, ok)
	got, ok := ParsePermissions(map[string]string{CapUsers: "none", CapAudit: LevelView})
	assert.True(t, ok)
	assert.Equal(t, Permissions{CapAudit: LevelView}, got)
}
