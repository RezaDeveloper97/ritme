package httpadmin

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Roles & permissions (bloom B-N9-02, D-84). A role grants each capability at a level: "view" (GET) or "manage"
// (every method; implies view). super has everything; editor keeps its pre-B-N9-02 reach (every capability except
// health data) so existing accounts work unchanged. A super admin may edit the other roles' matrices (admin_roles).

// Roles added by B-N9-02 (the Admin_Roles artboard).
const (
	RoleProductManager      = "product_manager"
	RoleMedicalAdvisor      = "medical_advisor"
	RoleSupport             = "support"
	RoleClinicalSupport     = "clinical_support"
	RoleAnalyst             = "analyst"
	RoleMarketing           = "marketing"
	RoleEngineering         = "engineering"
	RoleDoctor              = "doctor"
	RoleInstructorModerator = "instructor_moderator"
)

// Permission levels.
const (
	LevelView   = "view"
	LevelManage = "manage"
)

// Capabilities (rows of the matrix, in display order).
const (
	CapDashboard     = "dashboard"
	CapKPI           = "kpi" // KPI screen (T-M10-33: split from dashboard; SPEC §5.8 gives the analyst KPI only)
	CapOKR           = "okr" // OKR (T-M10-33: split from dashboard; product manager only)
	CapUsers         = "users"
	CapSegments      = "segments" // saved user-list filters (§5.2; marketing per §5.8)
	CapHealth        = "health"   // temporary access to one user's health data (manage only)
	CapMessages      = "messages"
	CapClinical      = "clinical"   // clinical content and approvals: pregnancy, checkups, catalog, children, assistant, safety, clinical message kinds
	CapThresholds    = "thresholds" // clinical thresholds (/clinical-config, T-M10-33: split from config)
	CapChallenges    = "challenges"
	CapConfig        = "config" // technical config: AI, modules
	CapExport        = "export" // anonymised exports (users directory, audit log CSV)
	CapContent       = "content"
	CapBilling       = "billing"
	CapSupport       = "support"
	CapLearning      = "learning"
	CapTelemed       = "telemed"
	CapTelemedDoctor = "telemed_doctor" // the doctor inbox (still needs telemed_doctors.admin_id)
	CapAudit         = "audit"
)

// Capability is one matrix row and the levels it can be granted at.
type Capability struct {
	Key    string
	Levels []string
}

var both = []string{LevelView, LevelManage}

// Capabilities lists the matrix rows.
var Capabilities = []Capability{
	{CapDashboard, both}, {CapKPI, both}, {CapOKR, both}, {CapUsers, both}, {CapSegments, both},
	{CapHealth, []string{LevelManage}}, {CapMessages, both}, {CapClinical, both}, {CapThresholds, both},
	{CapChallenges, both}, {CapConfig, both}, {CapExport, []string{LevelView}},
	{CapContent, both}, {CapBilling, both}, {CapSupport, both}, {CapLearning, both}, {CapTelemed, both},
	{CapTelemedDoctor, []string{LevelManage}}, {CapAudit, []string{LevelView}},
}

// Permissions maps a capability to its level.
type Permissions map[string]string

// Allows reports whether p grants capability at level (manage implies view).
func (p Permissions) Allows(capability, level string) bool {
	got := p[capability]
	return got == LevelManage || (got == LevelView && level == LevelView)
}

// Roles lists the valid roles.
var Roles = []string{
	RoleSuper, RoleEditor, RoleProductManager, RoleMedicalAdvisor, RoleSupport, RoleClinicalSupport, RoleAnalyst,
	RoleMarketing, RoleEngineering, RoleDoctor, RoleInstructorModerator,
}

// EditableRole reports whether a super admin may change role's matrix (only super is fixed; editor became editable
// in B-N9-02b).
func EditableRole(role string) bool {
	return role != RoleSuper && slices.Contains(Roles, role)
}

func allPermissions(except ...string) Permissions {
	p := Permissions{}
	for _, c := range Capabilities {
		if !slices.Contains(except, c.Key) {
			p[c.Key] = c.Levels[len(c.Levels)-1]
		}
	}
	return p
}

// DefaultPermissions is each editable role's matrix until a super admin edits it. Deny by default: a capability a
// role does not list is not granted, and an unknown role gets nothing.
//
// SPEC v6 §5.8 role table → capabilities (T-M10-33, D-182). A row of the spec that this matrix leaves out is not
// granted; "masked users" is users at view (the list and detail carry a masked mobile and no health value).
//
//	role (§5.8)       spec access                                    capabilities (m = manage, v = view)
//	----------------  ---------------------------------------------  --------------------------------------------------
//	product_manager   dashboard, KPI, OKR, masked users,             dashboard m, kpi m, okr m, users v, messages m,
//	                  non-clinical messages, challenges              challenges m (no clinical → clinical kinds are 403)
//	medical_advisor   clinical approval of messages and thresholds,  messages m + clinical m (approve / publish clinical
//	                  safety publishing, dashboard view              and safety rules), thresholds m, dashboard v
//	support           masked users, subscriptions, tickets           users v, billing v, support m
//	clinical_support  support + 30-minute health access (reason,     support's set + health m (per approved request,
//	                  ticket)                                        httpadmin.HealthGrant)
//	analyst           KPI, anonymous export                          kpi v, export v (no users: the anonymised export
//	                                                                 route maps to export)
//	marketing         product/marketing messages, challenges,        messages m (clinical kinds need clinical → 403),
//	                  segments                                       challenges m, segments m
//	engineering       technical config, view thresholds              config m (ai, modules), thresholds v (cannot
//	                                                                 propose or approve)
//
// Not in §5.8 (kept from B-N9-02): editor (everything but health, audit, export and the doctor inbox), doctor and
// instructor_moderator. A matrix a super admin saved in admin_roles wins over these defaults and is never rewritten.
func DefaultPermissions(role string) Permissions {
	m, v := LevelManage, LevelView
	switch role {
	case RoleSuper:
		return allPermissions()
	case RoleEditor:
		// B-N9-02b (least privilege): no health data, audit log, export or doctor inbox, and no unmapped routes.
		// Existing editors keep everything else; a super admin who wants the old reach back grants audit / export on
		// the Roles screen (PUT /roles/editor), or moves the account to a narrower role.
		return allPermissions(CapHealth, CapAudit, CapExport, CapTelemedDoctor)
	case RoleProductManager:
		return Permissions{CapDashboard: m, CapKPI: m, CapOKR: m, CapUsers: v, CapMessages: m, CapChallenges: m}
	case RoleMedicalAdvisor:
		return Permissions{CapDashboard: v, CapMessages: m, CapClinical: m, CapThresholds: m}
	case RoleSupport:
		return Permissions{CapUsers: v, CapBilling: v, CapSupport: m}
	case RoleClinicalSupport:
		return Permissions{CapUsers: v, CapBilling: v, CapSupport: m, CapHealth: m}
	case RoleAnalyst:
		return Permissions{CapKPI: v, CapExport: v}
	case RoleMarketing:
		return Permissions{CapMessages: m, CapChallenges: m, CapSegments: m}
	case RoleEngineering:
		return Permissions{CapConfig: m, CapThresholds: v}
	case RoleDoctor:
		return Permissions{CapDashboard: v, CapTelemedDoctor: m}
	case RoleInstructorModerator:
		return Permissions{CapDashboard: v, CapLearning: m}
	}
	return Permissions{}
}

// ParsePermissions validates a stored / submitted matrix: known capabilities at allowed levels only.
func ParsePermissions(raw map[string]string) (Permissions, bool) {
	p := Permissions{}
	for k, lvl := range raw {
		i := slices.IndexFunc(Capabilities, func(c Capability) bool { return c.Key == k })
		if i < 0 {
			return nil, false
		}
		if lvl == "" || lvl == "none" {
			continue
		}
		if !slices.Contains(Capabilities[i].Levels, lvl) {
			return nil, false
		}
		p[k] = lvl
	}
	return p, true
}

// capabilityOf maps an admin route (path under Prefix) to its capability. "" = any active admin (auth/me, …);
// unknown paths answer "?" (super only, B-N9-02b) so a new module is closed until it is mapped here.
func capabilityOf(path string) string {
	rest := strings.TrimPrefix(path, Prefix+"/")
	seg, tail, _ := strings.Cut(rest, "/")
	switch seg {
	case "auth":
		return ""
	case "dashboard", "overview":
		return CapDashboard
	case "kpi":
		return CapKPI
	case "okr":
		return CapOKR
	case "users":
		if tail == "directory/export" { // the anonymised CSV (no name, mobile or health value): export, not users
			return CapExport
		}
		return CapUsers
	case "segments":
		return CapSegments
	case "health-access":
		return CapHealth
	case "messages":
		return CapMessages
	case "challenges", "challenge-completions":
		return CapChallenges
	case "pregnancy-weeks", "pregnancy-week-details", "pregnancy-care-items", "pregnancy-alert-rules",
		"checkup-types", "catalog", "children", "assistant", "safety":
		return CapClinical
	case "clinical-config":
		return CapThresholds
	case "ai", "modules":
		return CapConfig
	case "articles", "affirmations", "banners", "info-sections", "phase-contents", "recommendations",
		"task-templates", "companions", "uploads", "media", "city-services":
		return CapContent
	case "shop":
		// CB-SHOP-05: the shipment queue, order detail and returns hold the recipient's name, phone and address →
		// support (the team that runs orders); the catalog, reviews, codes and layette template stay content (B-N10-03).
		if sub, _, _ := strings.Cut(tail, "/"); sub == "shipments" || sub == "orders" || sub == "returns" {
			return CapSupport
		}
		return CapContent
	case "insurance": // CB-INS-01: insurer catalog + contracted centres (admin content, no user data)
		return CapContent
	case "city-service-applications": // CB-DIR-05: directory join review queue (contact details also need «support»)
		return CapContent
	case "plus":
		return CapBilling
	case "support-reports", "partners", "city-service-bookings":
		return CapSupport
	case "learning":
		return CapLearning
	case "telemed":
		if tail == "doctor" || strings.HasPrefix(tail, "doctor/") {
			return CapTelemedDoctor
		}
		return CapTelemed
	case "audit-logs", "audit": // audit/verify is super-only on top (kit.Super)
		return CapAudit
	}
	return "?"
}

// RouteCapability is capabilityOf for tests (every registered admin route must map).
func RouteCapability(path string) string { return capabilityOf(path) }

// RBACStore is the slice of store.Queries the kit needs for roles and the audit log.
type RBACStore interface {
	GetAdminRolePermissions(ctx context.Context, role string) (string, error)
}

// PermissionsFor resolves role's matrix: the fixed super / editor sets, the admin_roles row, else the defaults.
func (k *Kit) PermissionsFor(ctx context.Context, role string) (Permissions, error) {
	if !EditableRole(role) || k.rbac == nil {
		return DefaultPermissions(role), nil
	}
	raw, err := k.rbac.GetAdminRolePermissions(ctx, role)
	if errors.Is(err, sql.ErrNoRows) {
		return DefaultPermissions(role), nil
	}
	if err != nil {
		return nil, err
	}
	var m map[string]string
	if uerr := json.Unmarshal([]byte(raw), &m); uerr != nil {
		// Fail closed: an unreadable row grants nothing (a super admin can re-save or reset the role).
		k.logger.ErrorContext(ctx, "admin_roles row unreadable; using no permissions", slog.String("role", role))
		m = nil
	}
	p, _ := ParsePermissions(m)
	return p, nil
}

// CurrentPermissions is the authenticated admin's matrix (loaded once per request).
func CurrentPermissions(c fiber.Ctx) Permissions {
	p, _ := c.Locals(permsKey).(Permissions)
	return p
}

// Can reports whether the current admin holds capability at level.
func Can(c fiber.Ctx, capability, level string) bool {
	if a := CurrentAdmin(c); a != nil && a.Role == RoleSuper {
		return true
	}
	return CurrentPermissions(c).Allows(capability, level)
}

// ForbiddenPermission is the 403 for a capability the admin's role lacks.
func ForbiddenPermission(capability string) error {
	return Fail(fiber.StatusForbidden, CodeForbidden, "Your role does not allow this action.", "permission", capability)
}

// RequirePermission maps the matched route to its capability and checks the admin's role (403). GET/HEAD need
// view, everything else manage. It runs after RequireAdmin.
func (k *Kit) RequirePermission(c fiber.Ctx) error {
	a := CurrentAdmin(c)
	if a == nil {
		return Unauthenticated(CodeUnauthenticated)
	}
	perms, err := k.PermissionsFor(c.Context(), a.Role)
	if err != nil {
		return err
	}
	c.Locals(permsKey, perms)
	if a.Role == RoleSuper {
		return c.Next()
	}
	capability := capabilityOf(c.Route().Path)
	switch capability {
	case "":
		return c.Next()
	case "?":
		return ForbiddenPermission(capability)
	}
	level := LevelManage
	if isSafeMethod(c.Method()) {
		level = LevelView
	}
	if !perms.Allows(capability, level) {
		return ForbiddenPermission(capability)
	}
	return c.Next()
}

// PermissionsJSON is the matrix in capability order ({capability: level}, ungranted ones omitted).
func PermissionsJSON(p Permissions) *jsonx.OrderedMap {
	out := jsonx.Obj()
	for _, c := range Capabilities {
		if lvl, ok := p[c.Key]; ok {
			out.Set(c.Key, lvl)
		}
	}
	return out
}
