package rbac_test

import (
	"bytes"
	"database/sql"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/auditchain"
	"github.com/ritme/backend-go/internal/admin/clinicalcfg"
	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/messages"
	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// T-M10-33 (SPEC v6 §5.8, D-182): the §5.8 role matrix against the real message and threshold handlers, the
// hash-chained audit log and fail-closed health reads.

const (
	productManagerID uint64 = 20
	advisorID        uint64 = 21
	engineeringID    uint64 = 22
)

func matrixEnv(t *testing.T) *admintest.Env {
	t.Helper()
	e := newEnv(t)
	messages.New(e.DB, admintest.Quiet).Routes(e.Route(), e.Kit)
	clinicalcfg.NewHandlers(e.DB, admintest.Quiet).Routes(e.Route(), e.Kit)
	e.Exec(`INSERT INTO admins (id, name, email, password, role, is_active, created_at, updated_at)
		SELECT 20, 'PM', 'pm@ritme.test', password, 'product_manager', 1, NOW(), NOW() FROM admins WHERE id = 1
		UNION ALL SELECT 21, 'Dr', 'dr2@ritme.test', password, 'medical_advisor', 1, NOW(), NOW() FROM admins WHERE id = 1
		UNION ALL SELECT 22, 'Eng', 'eng@ritme.test', password, 'engineering', 1, NOW(), NOW() FROM admins WHERE id = 1`)
	e.Exec("DELETE FROM message_rules")
	clinicalconfig.Install(clinicalcfg.Loader(e.DB), admintest.Quiet)
	t.Cleanup(clinicalconfig.Reset)
	return e
}

func clinicalRuleBody() map[string]any {
	return map[string]any{
		"title": "PMS", "level": "worth_followup", "kind": "clinical", "priority": 160, "placement": "smart_tip",
		"content": map[string]any{"a": map[string]any{"title": "t {name}", "body": "b"}, "seen": "s", "confidence": "c"},
		"conditions": []any{map[string]any{"op": "and", "items": []any{
			map[string]any{"fact": "mode", "op": "in", "value": []any{"cycle"}}}}},
		"exceptions": []any{}, "cooldown_days": 7, "audience_percent": 100, "goal": "open symptoms",
	}
}

func productRuleBody() map[string]any {
	return map[string]any{
		"title": "Log reminder", "level": "suggestion", "kind": "product", "priority": 30, "placement": "push",
		"content":    map[string]any{"a": map[string]any{"title": "t", "body": "b", "cta_label": "Log", "cta_target": "ritme://log"}},
		"conditions": []any{}, "exceptions": []any{}, "cap_count": 1, "cap_period": "day",
	}
}

// The default matrix per role and capability (code defaults, no admin_roles rows).
func TestMatrix_SpecRolesAgainstRealHandlers(t *testing.T) {
	e := matrixEnv(t)

	// Marketing writes product messages, never clinical ones (403 with T-M10-02's message authority).
	mk := e.As(marketingID)
	r := mk.JSON(fiber.MethodPost, "/messages/rules", clinicalRuleBody())
	assert.Equal(t, 403, r.Status, string(r.Raw))
	assert.Equal(t, httpadmin.CodeForbidden, r.Code())
	r = mk.JSON(fiber.MethodPost, "/messages/rules", productRuleBody())
	assert.Equal(t, 201, r.Status, string(r.Raw))
	// The product manager: non-clinical messages only.
	assert.Equal(t, 403, e.As(productManagerID).JSON(fiber.MethodPost, "/messages/rules", clinicalRuleBody()).Status)
	assert.Equal(t, 201, e.As(productManagerID).JSON(fiber.MethodPost, "/messages/rules", productRuleBody()).Status)
	// The medical advisor authors the clinical one; support and the analyst have no messages at all.
	assert.Equal(t, 201, e.As(advisorID).JSON(fiber.MethodPost, "/messages/rules", clinicalRuleBody()).Status)
	assert.Equal(t, 403, e.As(supportID).Get("/messages/rules").Status)
	assert.Equal(t, 403, e.As(analystID).Get("/messages/rules").Status)

	// Thresholds: engineering views and can neither propose nor approve; the advisor proposes.
	day := civildate.InTehran(time.Now())
	body := map[string]any{"value": map[string]any{"systolic": 170, "diastolic": 110}, "effective_from": day.String(),
		"source": "ACC/AHA 2025 draft", "note": "test"}
	eng := e.As(engineeringID)
	assert.Equal(t, 200, eng.Get("/clinical-config").Status)
	assert.Equal(t, 403, eng.JSON(fiber.MethodPost, "/clinical-config/bp.crisis/proposals", body).Status)
	p := e.As(advisorID).JSON(fiber.MethodPost, "/clinical-config/bp.crisis/proposals", body)
	require.Equal(t, 201, p.Status, string(p.Raw))
	id := itoa(uint64(p.Obj("proposal")["id"].(float64)))
	assert.Equal(t, 403, eng.JSON(fiber.MethodPost, "/clinical-config/proposals/"+id+"/approve", nil).Status)
	assert.Equal(t, 403, e.As(marketingID).Get("/clinical-config").Status)
	assert.Equal(t, 403, e.As(productManagerID).Get("/clinical-config").Status)

	// The defaults, capability by capability (nothing beyond the §5.8 row).
	want := map[string]httpadmin.Permissions{
		httpadmin.RoleProductManager: {"dashboard": "manage", "kpi": "manage", "okr": "manage", "users": "view",
			"messages": "manage", "challenges": "manage"},
		httpadmin.RoleMedicalAdvisor:  {"dashboard": "view", "messages": "manage", "clinical": "manage", "thresholds": "manage"},
		httpadmin.RoleSupport:         {"users": "view", "billing": "view", "support": "manage"},
		httpadmin.RoleClinicalSupport: {"users": "view", "billing": "view", "support": "manage", "health": "manage"},
		httpadmin.RoleAnalyst:         {"kpi": "view", "export": "view"},
		httpadmin.RoleMarketing:       {"messages": "manage", "challenges": "manage", "segments": "manage"},
		httpadmin.RoleEngineering:     {"config": "manage", "thresholds": "view"},
	}
	for role, perms := range want {
		got, err := e.Kit.PermissionsFor(t.Context(), role)
		require.NoError(t, err)
		assert.Equal(t, perms, got, role)
	}
}

// ---------------------------------------------------------------------------
// Hash chain

func headRow(t *testing.T, e *admintest.Env) (lastID int64, lastHash, genesisHash string, genesisLastID int64) {
	t.Helper()
	var gid sql.NullInt64
	var lh, gh sql.NullString
	var lid sql.NullInt64
	require.NoError(t, e.DB.QueryRowContext(t.Context(),
		`SELECT last_id, last_hash, genesis_last_id, genesis_hash FROM admin_audit_chain_head WHERE id = 1`).
		Scan(&lid, &lh, &gid, &gh))
	return lid.Int64, lh.String, gh.String, gid.Int64
}

// liftGuard drops the append-only triggers in this package's test database and puts them back afterwards.
func liftGuard(t *testing.T, e *admintest.Env) {
	t.Helper()
	e.Exec("DROP TRIGGER IF EXISTS `admin_audit_logs_no_update`")
	e.Exec("DROP TRIGGER IF EXISTS `admin_audit_logs_no_delete`")
	t.Cleanup(restoreGuard(e))
}

func restoreGuard(e *admintest.Env) func() {
	return func() {
		for _, ev := range []string{"UPDATE", "DELETE"} {
			name := "admin_audit_logs_no_" + strings.ToLower(ev)
			_, _ = e.DB.Exec("CREATE TRIGGER IF NOT EXISTS `" + name + "` BEFORE " + ev + " ON `admin_audit_logs` FOR EACH ROW\n" +
				"  SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'admin_audit_logs is append-only'")
		}
	}
}

func verifyAs(t *testing.T, e *admintest.Env, query string) map[string]any {
	t.Helper()
	r := e.As(admintest.SuperID).Get("/audit/verify" + query)
	require.Equal(t, 200, r.Status, string(r.Raw))
	return r.Data()
}

func breakOf(d map[string]any) (float64, string) {
	b, _ := d["break"].(map[string]any)
	if b == nil {
		return 0, ""
	}
	return b["id"].(float64), b["reason"].(string)
}

func appendN(t *testing.T, e *admintest.Env, n int) {
	t.Helper()
	c := auditchain.New(e.DB)
	for i := range n {
		_, err := c.Append(t.Context(), auditchain.Row{AdminID: sql.NullInt64{Int64: 1, Valid: true}, Action: "test.append",
			TargetType: "test", TargetID: uint64(i + 1), IP: "10.0.0.1",
			Detail:    sql.NullString{String: `{"n":` + itoa(uint64(i)) + `}`, Valid: true},
			CreatedAt: httpadmin.DBTime(time.Now())})
		require.NoError(t, err)
	}
}

func TestAuditChain_GenesisAppendAndVerify(t *testing.T) {
	e := newEnv(t)
	// Before the chain: nothing to verify.
	d := verifyAs(t, e, "")
	assert.Equal(t, true, d["ok"])
	assert.Equal(t, false, d["started"])

	// Three rows written before the chain started (no hashes).
	for i := range 3 {
		e.Exec(`INSERT INTO admin_audit_logs (admin_id, action, target_type, target_id, ip, detail, created_at)
			VALUES (1, 'legacy.action', 'user', ?, '', NULL, '2026-09-01 10:00:00')`, i+1)
	}
	// The first chained write (a real endpoint) runs genesis first.
	require.Equal(t, 200, e.As(admintest.SuperID).JSON(fiber.MethodPost, "/users/11/block", nil).Status)
	lastID, lastHash, genesisHash, genesisLastID := headRow(t, e)
	assert.EqualValues(t, 3, genesisLastID)
	assert.EqualValues(t, 4, lastID)
	assert.Equal(t, genesisHash, e.String(`SELECT prev_hash FROM admin_audit_logs WHERE id = 4`))
	assert.Equal(t, lastHash, e.String(`SELECT row_hash FROM admin_audit_logs WHERE id = 4`))
	assert.Equal(t, 3, e.Int(`SELECT COUNT(*) FROM admin_audit_logs WHERE row_hash IS NULL`), "legacy rows stay unchanged")

	appendN(t, e, 5)
	d = verifyAs(t, e, "")
	assert.Equal(t, true, d["ok"], d)
	assert.Equal(t, true, d["started"])
	assert.EqualValues(t, 9, d["checked"])
	assert.EqualValues(t, 3, d["genesis_last_id"])
	assert.EqualValues(t, 9, d["head"].(map[string]any)["id"])
	// Partial walks: from inside the chain and from inside genesis.
	d = verifyAs(t, e, "?from=6")
	assert.Equal(t, true, d["ok"], d)
	assert.EqualValues(t, 4, d["checked"])
	assert.Equal(t, true, verifyAs(t, e, "?from=2")["ok"])
	assert.Equal(t, 422, e.As(admintest.SuperID).Get("/audit/verify?from=x").Status)

	// The nightly job anchors the head in the structured log, never row content.
	var buf bytes.Buffer
	rep, err := auditchain.RunVerify(t.Context(), auditchain.New(e.DB), slog.New(slog.NewJSONHandler(&buf, nil)))
	require.NoError(t, err)
	assert.Nil(t, rep.Break)
	_, lastHash, _, _ = headRow(t, e)
	assert.Contains(t, buf.String(), "admin audit chain anchor")
	assert.Contains(t, buf.String(), lastHash)
	assert.NotContains(t, buf.String(), "10.0.0.1")
	assert.NotContains(t, buf.String(), "test.append")
}

func TestAuditChain_AppendOnlyInTheDatabase(t *testing.T) {
	e := newEnv(t)
	appendN(t, e, 2)
	_, err := e.DB.ExecContext(t.Context(), `UPDATE admin_audit_logs SET action = 'x' WHERE id = 1`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "append-only")
	_, err = e.DB.ExecContext(t.Context(), `DELETE FROM admin_audit_logs WHERE id = 1`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "append-only")
	assert.Equal(t, 2, e.Int(`SELECT COUNT(*) FROM admin_audit_logs`))
}

func TestAuditChain_ConcurrentAppendsDoNotFork(t *testing.T) {
	e := newEnv(t)
	const workers, each = 8, 25
	c := auditchain.New(e.DB)
	var wg sync.WaitGroup
	errs := make(chan error, workers*each)
	for w := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range each {
				_, err := c.Append(t.Context(), auditchain.Row{Action: "test.concurrent", TargetType: "worker",
					TargetID: uint64(w*1000 + i), CreatedAt: httpadmin.DBTime(time.Now())}) //nolint:gosec // small
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	n := workers * each
	assert.Equal(t, n, e.Int(`SELECT COUNT(*) FROM admin_audit_logs`))
	// No fork: every prev_hash is used once, and each is the row_hash of the row before (by id).
	assert.Equal(t, n, e.Int(`SELECT COUNT(DISTINCT prev_hash) FROM admin_audit_logs`))
	assert.Equal(t, n-1, e.Int(`SELECT COUNT(*) FROM admin_audit_logs a JOIN admin_audit_logs b
		ON b.id = (SELECT MIN(id) FROM admin_audit_logs WHERE id > a.id) AND b.prev_hash = a.row_hash`))
	d := verifyAs(t, e, "")
	assert.Equal(t, true, d["ok"], d)
	assert.EqualValues(t, n, d["checked"])
}

func TestAuditChain_TamperingIsDetected(t *testing.T) {
	t.Run("edited row", func(t *testing.T) {
		e := newEnv(t)
		appendN(t, e, 5)
		liftGuard(t, e)
		e.Exec(`UPDATE admin_audit_logs SET target_id = 99 WHERE id = 3`)
		id, reason := breakOf(verifyAs(t, e, ""))
		assert.EqualValues(t, 3, id)
		assert.Equal(t, auditchain.BreakHash, reason)
		assert.Equal(t, false, verifyAs(t, e, "")["ok"])
		// The nightly job raises the alert line.
		var buf bytes.Buffer
		rep, err := auditchain.RunVerify(t.Context(), auditchain.New(e.DB), slog.New(slog.NewJSONHandler(&buf, nil)))
		require.NoError(t, err)
		require.NotNil(t, rep.Break)
		assert.Contains(t, buf.String(), `"alert":"admin_audit_chain_broken"`)
		// Putting the value back heals it (the hash is over the content).
		e.Exec(`UPDATE admin_audit_logs SET target_id = 3 WHERE id = 3`)
		assert.Equal(t, true, verifyAs(t, e, "")["ok"])
	})
	t.Run("edited row with a recomputed hash", func(t *testing.T) {
		e := newEnv(t)
		appendN(t, e, 5)
		liftGuard(t, e)
		var prev string
		var row auditchain.Row
		require.NoError(t, e.DB.QueryRowContext(t.Context(), `SELECT prev_hash, admin_id, action, target_type, target_id,
			ip, detail, created_at FROM admin_audit_logs WHERE id = 3`).Scan(&prev, &row.AdminID, &row.Action,
			&row.TargetType, &row.TargetID, &row.IP, &row.Detail, &row.CreatedAt))
		row.Action = "test.forged"
		e.Exec(`UPDATE admin_audit_logs SET action = ?, row_hash = ? WHERE id = 3`, row.Action, auditchain.Hash(prev, row))
		id, reason := breakOf(verifyAs(t, e, ""))
		assert.EqualValues(t, 4, id)
		assert.Equal(t, auditchain.BreakPrev, reason)
	})
	t.Run("deleted row", func(t *testing.T) {
		e := newEnv(t)
		appendN(t, e, 5)
		liftGuard(t, e)
		e.Exec(`DELETE FROM admin_audit_logs WHERE id = 2`)
		id, reason := breakOf(verifyAs(t, e, ""))
		assert.EqualValues(t, 3, id)
		assert.Equal(t, auditchain.BreakPrev, reason)
	})
	t.Run("deleted tail", func(t *testing.T) {
		e := newEnv(t)
		appendN(t, e, 5)
		liftGuard(t, e)
		e.Exec(`DELETE FROM admin_audit_logs WHERE id = 5`)
		id, reason := breakOf(verifyAs(t, e, ""))
		assert.EqualValues(t, 5, id)
		assert.Equal(t, auditchain.BreakHead, reason)
	})
	t.Run("unchained row after genesis", func(t *testing.T) {
		e := newEnv(t)
		appendN(t, e, 2)
		e.Exec(`INSERT INTO admin_audit_logs (admin_id, action, target_type, target_id, ip, created_at)
			VALUES (1, 'sneaky', 'user', 1, '', NOW())`)
		id, reason := breakOf(verifyAs(t, e, ""))
		assert.EqualValues(t, 3, id)
		assert.Equal(t, auditchain.BreakUnchained, reason)
	})
	t.Run("edited or deleted legacy row", func(t *testing.T) {
		e := newEnv(t)
		for i := range 3 {
			e.Exec(`INSERT INTO admin_audit_logs (admin_id, action, target_type, target_id, ip, created_at)
				VALUES (1, 'legacy', 'user', ?, '', '2026-09-01 10:00:00')`, i+1)
		}
		appendN(t, e, 2)
		require.Equal(t, true, verifyAs(t, e, "")["ok"])
		liftGuard(t, e)
		e.Exec(`DELETE FROM admin_audit_logs WHERE id = 2`)
		id, reason := breakOf(verifyAs(t, e, ""))
		assert.EqualValues(t, 3, id)
		assert.Equal(t, auditchain.BreakGenesis, reason)
	})
}

// ---------------------------------------------------------------------------
// Fail-closed health reads

func TestHealthRead_FailsClosedWhenTheAuditRowFails(t *testing.T) {
	e := newEnv(t)
	clin := e.As(clinicalID)
	path := requestAccess(t, clin)
	require.Equal(t, 200, e.As(admintest.SuperID).JSON(fiber.MethodPost, path+"/approve", nil).Status)
	require.Equal(t, 200, clin.Get("/health-access/users/10").Status)
	reads := e.Int(`SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'health_access.read'`)

	// Every insert into the audit log now fails.
	e.Exec("CREATE TRIGGER `admin_audit_logs_test_no_insert` BEFORE INSERT ON `admin_audit_logs` FOR EACH ROW\n" +
		"  SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'audit insert refused (test)'")
	t.Cleanup(func() { _, _ = e.DB.Exec("DROP TRIGGER IF EXISTS `admin_audit_logs_test_no_insert`") })

	r := clin.Get("/health-access/users/10")
	assert.Equal(t, 503, r.Status, string(r.Raw))
	assert.Equal(t, httpadmin.CodeAuditUnavailable, r.Code())
	assert.Nil(t, r.Body["data"])
	assert.NotContains(t, string(r.Raw), "cycle_duration")
	assert.NotContains(t, string(r.Raw), "2026-09-20")
	// The user detail fails closed the same way (no health field, no body data).
	u := clin.Get("/users/10")
	assert.Equal(t, 503, u.Status, string(u.Raw))
	assert.Equal(t, httpadmin.CodeAuditUnavailable, u.Code())
	assert.NotContains(t, string(u.Raw), "2026-09-20")
	assert.Equal(t, reads, e.Int(`SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'health_access.read'`))
	// The chain did not move either (the failed append rolled back).
	assert.Equal(t, true, verifyAs(t, e, "")["ok"])
}
