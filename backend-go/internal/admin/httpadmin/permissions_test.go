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
		// T-M10-33 (§5.8): KPI / OKR / thresholds / segments are their own rows; the anonymised export is export.
		Prefix + "/kpi":                                   CapKPI,
		Prefix + "/okr/objectives/:id":                    CapOKR,
		Prefix + "/dashboard":                             CapDashboard,
		Prefix + "/overview":                              CapDashboard,
		Prefix + "/clinical-config/proposals/:id/approve": CapThresholds,
		Prefix + "/ai/settings":                           CapConfig,
		Prefix + "/modules":                               CapConfig,
		Prefix + "/segments/:id":                          CapSegments,
		Prefix + "/users/directory/export":                CapExport,
		Prefix + "/users/directory":                       CapUsers,
		Prefix + "/audit/verify":                          CapAudit,
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

// SPEC v6 §5.8 (T-M10-33, D-182): each role's default matrix, capability by capability. Deny by default: anything
// not listed is not granted.
func TestDefaultPermissions_Spec58(t *testing.T) {
	m, v := LevelManage, LevelView
	want := map[string]Permissions{
		RoleProductManager:  {CapDashboard: m, CapKPI: m, CapOKR: m, CapUsers: v, CapMessages: m, CapChallenges: m},
		RoleMedicalAdvisor:  {CapDashboard: v, CapMessages: m, CapClinical: m, CapThresholds: m},
		RoleSupport:         {CapUsers: v, CapBilling: v, CapSupport: m},
		RoleClinicalSupport: {CapUsers: v, CapBilling: v, CapSupport: m, CapHealth: m},
		RoleAnalyst:         {CapKPI: v, CapExport: v},
		RoleMarketing:       {CapMessages: m, CapChallenges: m, CapSegments: m},
		RoleEngineering:     {CapConfig: m, CapThresholds: v},
	}
	for role, perms := range want {
		assert.Equal(t, perms, DefaultPermissions(role), role)
	}
	// Capability × role: the spec's key negatives.
	cases := []struct {
		role, capability, level string
		allowed                 bool
	}{
		{RoleAnalyst, CapUsers, v, false},
		{RoleAnalyst, CapKPI, v, true},
		{RoleAnalyst, CapKPI, m, false},
		{RoleAnalyst, CapDashboard, v, false},
		{RoleAnalyst, CapExport, v, true},
		{RoleMarketing, CapClinical, m, false},
		{RoleMarketing, CapMessages, m, true},
		{RoleMarketing, CapContent, v, false},
		{RoleEngineering, CapThresholds, v, true},
		{RoleEngineering, CapThresholds, m, false},
		{RoleEngineering, CapConfig, m, true},
		{RoleProductManager, CapUsers, m, false},
		{RoleProductManager, CapExport, v, false},
		{RoleProductManager, CapConfig, v, false},
		{RoleProductManager, CapClinical, v, false},
		{RoleMedicalAdvisor, CapThresholds, m, true},
		{RoleMedicalAdvisor, CapDashboard, m, false},
		{RoleMedicalAdvisor, CapUsers, v, false},
		{RoleSupport, CapUsers, m, false},
		{RoleSupport, CapHealth, m, false},
		{RoleClinicalSupport, CapHealth, m, true},
		{"unknown_role", CapDashboard, v, false},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.allowed, DefaultPermissions(tc.role).Allows(tc.capability, tc.level),
			"%s %s %s", tc.role, tc.capability, tc.level)
	}
	// Every default only uses known capabilities at allowed levels.
	for _, role := range Roles {
		raw := map[string]string{}
		for k, lvl := range DefaultPermissions(role) {
			raw[k] = lvl
		}
		_, ok := ParsePermissions(raw)
		assert.True(t, ok, role)
	}
}
