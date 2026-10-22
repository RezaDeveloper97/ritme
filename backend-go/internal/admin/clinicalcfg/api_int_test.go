package clinicalcfg_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/clinicalcfg"
	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/vitals"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	advisor1    uint64 = 3
	advisor2    uint64 = 4
	advisor3    uint64 = 5
	engineering uint64 = 6
	marketing   uint64 = 7
)

func newEnv(t *testing.T) *admintest.Env {
	t.Helper()
	e := admintest.New(t)
	clinicalcfg.NewHandlers(e.DB, admintest.Quiet).Routes(e.Route(), e.Kit)
	const pw = "$2y$12$lo8Z3h/WEQp7b8AwCvwiTObVkv7/Xa68c/VnNTSszAOrR4kLj1HBi" //nolint:gosec // G101: test fixture hash
	e.Exec(`INSERT INTO admins (id, name, email, password, role, is_active, created_at, updated_at) VALUES
		(3, 'Dr Ahmadi', 'med1@ritme.test', ?, 'medical_advisor', 1, NOW(), NOW()),
		(4, 'Dr Niko', 'med2@ritme.test', ?, 'medical_advisor', 1, NOW(), NOW()),
		(5, 'Dr Rostami', 'med3@ritme.test', ?, 'medical_advisor', 1, NOW(), NOW()),
		(6, 'Eng', 'eng@ritme.test', ?, 'engineering', 1, NOW(), NOW()),
		(7, 'Mark', 'mkt@ritme.test', ?, 'marketing', 1, NOW(), NOW())`, pw, pw, pw, pw, pw)
	clinicalconfig.Install(clinicalcfg.Loader(e.DB), admintest.Quiet)
	t.Cleanup(clinicalconfig.Reset)
	return e
}

func today() civildate.Date { return civildate.InTehran(time.Now()) }

func proposalID(t *testing.T, r admintest.Resp) uint64 {
	t.Helper()
	id, ok := r.Obj("proposal")["id"].(float64)
	require.True(t, ok, "proposal id in %v", r.Body)
	return uint64(id)
}

func path(id uint64, action string) string {
	return "/clinical-config/proposals/" + strconv.FormatUint(id, 10) + "/" + action
}

func resolve(t *testing.T, e *admintest.Env, key string, d civildate.Date) map[string]any {
	t.Helper()
	r := e.As(advisor1).Get("/clinical-config/resolve?key=" + key + "&date=" + d.String())
	require.Equal(t, 200, r.Status, r.Body)
	return r.Data()
}

func TestTwoDistinctAdvisorApprovals(t *testing.T) {
	e := newEnv(t)
	day := today()
	body := map[string]any{"value": map[string]any{"systolic": 170, "diastolic": 110}, "effective_from": day.String(),
		"source": "ACC/AHA 2025 draft", "note": "test"}

	// Auth & capability.
	assert.Equal(t, 401, e.Anonymous().Get("/clinical-config").Status)
	assert.Equal(t, 403, e.As(marketing).Get("/clinical-config").Status)
	assert.Equal(t, 403, e.As(marketing).JSON(fiber.MethodPost, "/clinical-config/bp.crisis/proposals", body).Status)

	list := e.As(advisor1).Get("/clinical-config")
	require.Equal(t, 200, list.Status)
	assert.Equal(t, true, list.Data()["can_approve"])
	assert.Len(t, list.Data()["keys"], len(clinicalconfig.Defs))
	assert.Equal(t, false, e.As(engineering).Get("/clinical-config").Data()["can_approve"])

	// Validation.
	assert.Equal(t, 404, e.As(advisor1).JSON(fiber.MethodPost, "/clinical-config/nope/proposals", body).Status)
	bad := map[string]any{"value": map[string]any{"systolic": 999, "diastolic": 110}, "effective_from": day.String()}
	assert.Equal(t, 422, e.As(advisor1).JSON(fiber.MethodPost, "/clinical-config/bp.crisis/proposals", bad).Status)
	past := map[string]any{"value": map[string]any{"systolic": 170, "diastolic": 110}, "effective_from": day.AddDays(-1).String()}
	assert.Equal(t, 422, e.As(advisor1).JSON(fiber.MethodPost, "/clinical-config/bp.crisis/proposals", past).Status)
	missing := map[string]any{"value": map[string]any{"systolic": 170}, "effective_from": day.String()}
	assert.Equal(t, 422, e.As(advisor1).JSON(fiber.MethodPost, "/clinical-config/bp.crisis/proposals", missing).Status)

	// Engineering proposes (config: manage) …
	p := e.As(engineering).JSON(fiber.MethodPost, "/clinical-config/bp.crisis/proposals", body)
	require.Equal(t, 201, p.Status, p.Body)
	assert.EqualValues(t, 2, p.Obj("proposal")["version"])
	id := proposalID(t, p)
	assert.Equal(t, clinicalcfg.CodePendingExists,
		e.As(advisor1).JSON(fiber.MethodPost, "/clinical-config/bp.crisis/proposals", body).Code())

	// … but cannot approve; neither can the proposer.
	r := e.As(engineering).JSON(fiber.MethodPost, path(id, "approve"), nil)
	assert.Equal(t, 403, r.Status)
	assert.Equal(t, clinicalcfg.CodeAdvisorRequired, r.Code())

	r = e.As(advisor1).JSON(fiber.MethodPost, path(id, "approve"), nil)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, clinicalcfg.StatusPending, r.Obj("proposal")["status"])
	assert.Equal(t, clinicalcfg.CodeAlreadyApproved, e.As(advisor1).JSON(fiber.MethodPost, path(id, "approve"), nil).Code())
	assert.EqualValues(t, 1, resolve(t, e, clinicalconfig.KeyBPCrisis, day)["version"], "one approval is not enough")

	r = e.As(advisor2).JSON(fiber.MethodPost, path(id, "approve"), nil)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, clinicalcfg.StatusApproved, r.Obj("proposal")["status"])
	assert.Len(t, r.Obj("proposal")["approvals"], 2)
	assert.Equal(t, clinicalcfg.CodeNotPending, e.As(advisor3).JSON(fiber.MethodPost, path(id, "approve"), nil).Code())

	// Effective from today: today reads v2, yesterday still v1 — and the engines follow.
	assert.EqualValues(t, 2, resolve(t, e, clinicalconfig.KeyBPCrisis, day)["version"])
	assert.EqualValues(t, 1, resolve(t, e, clinicalconfig.KeyBPCrisis, day.AddDays(-1))["version"])
	reading := vitals.Reading{Type: vitals.TypeBP, Systolic: 175, Diastolic: 95, Date: day}
	assert.Equal(t, vitals.ClassCrisis, reading.Class())
	reading.Date = day.AddDays(-1)
	assert.Equal(t, vitals.ClassStage2, reading.Class(), "a historic reading keeps its old class")

	// Audit trail.
	assert.Equal(t, 3, e.Int(`SELECT COUNT(*) FROM admin_audit_logs WHERE action LIKE 'clinical_config.%'`))

	// Proposer self-approval.
	p2 := e.As(advisor3).JSON(fiber.MethodPost, "/clinical-config/epds.cutoff.likely/proposals",
		map[string]any{"value": map[string]any{"score": 12}, "effective_from": day.AddDays(30).String()})
	require.Equal(t, 201, p2.Status, p2.Body)
	r = e.As(advisor3).JSON(fiber.MethodPost, path(proposalID(t, p2), "approve"), nil)
	assert.Equal(t, 403, r.Status)
	assert.Equal(t, clinicalcfg.CodeSelfApproval, r.Code())

	// Reject: marketing cannot (no capability), engineering (not proposer, not advisor) cannot, the proposer withdraws.
	assert.Equal(t, 403, e.As(marketing).JSON(fiber.MethodPost, path(proposalID(t, p2), "reject"), nil).Status)
	assert.Equal(t, clinicalcfg.CodeAdvisorRequired,
		e.As(engineering).JSON(fiber.MethodPost, path(proposalID(t, p2), "reject"), nil).Code())
	r = e.As(advisor3).JSON(fiber.MethodPost, path(proposalID(t, p2), "reject"), nil)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, clinicalcfg.StatusRejected, r.Obj("proposal")["status"])

	// IDOR / unknown ids.
	assert.Equal(t, 404, e.As(advisor1).JSON(fiber.MethodPost, path(999999, "approve"), nil).Status)
	assert.Equal(t, 404, e.As(advisor1).Get("/clinical-config/nope").Status)
	show := e.As(advisor1).Get("/clinical-config/bp.crisis")
	require.Equal(t, 200, show.Status)
	assert.Len(t, show.Data()["versions"], 2, "v2 + the code default v1")
}

func TestEffectiveFromAndHistory(t *testing.T) {
	e := newEnv(t)
	day := today()
	// History as stored: v2 from 60 days ago, v3 from 10 days ahead.
	e.Exec(`INSERT INTO clinical_config_versions (config_key, version, value, effective_from, status, created_at, updated_at) VALUES
		('engine.period_hard_cap', 2, '{"days":11}', ?, 'approved', NOW(), NOW()),
		('engine.period_hard_cap', 3, '{"days":10}', ?, 'approved', NOW(), NOW())`,
		day.AddDays(-60), day.AddDays(10))
	require.NoError(t, clinicalconfig.Refresh(t.Context()))

	assert.EqualValues(t, 1, resolve(t, e, clinicalconfig.KeyPeriodHardCap, day.AddDays(-61))["version"])
	assert.EqualValues(t, 2, resolve(t, e, clinicalconfig.KeyPeriodHardCap, day.AddDays(-60))["version"])
	assert.EqualValues(t, 2, resolve(t, e, clinicalconfig.KeyPeriodHardCap, day)["version"])
	assert.EqualValues(t, 3, resolve(t, e, clinicalconfig.KeyPeriodHardCap, day.AddDays(10))["version"])
	assert.Equal(t, 11, clinicalconfig.Int(clinicalconfig.KeyPeriodHardCap, "days", day))

	list := e.As(advisor1).Get("/clinical-config?group=engine")
	require.Equal(t, 200, list.Status)
	var hardCap map[string]any
	for _, k := range list.Data()["keys"].([]any) {
		if m := k.(map[string]any); m["key"] == clinicalconfig.KeyPeriodHardCap {
			hardCap = m
		}
	}
	require.NotNil(t, hardCap)
	assert.EqualValues(t, 2, hardCap["current"].(map[string]any)["version"])
	assert.EqualValues(t, 3, hardCap["upcoming"].(map[string]any)["version"])
	assert.Equal(t, 422, e.As(advisor1).Get("/clinical-config?group=nope").Status)

	// A pending proposal whose date passed before its second approval cannot be approved.
	res := e.Exec(`INSERT INTO clinical_config_versions (config_key, version, value, effective_from, status, proposed_by, created_at, updated_at)
		VALUES ('hr.resting', 2, '{"min":55,"max":100}', ?, 'pending', 6, NOW(), NOW())`, day.AddDays(-1))
	id, err := res.LastInsertId()
	require.NoError(t, err)
	r := e.As(advisor1).JSON(fiber.MethodPost, path(uint64(id), "approve"), nil) //nolint:gosec // test id
	assert.Equal(t, 422, r.Status)
	assert.Equal(t, clinicalcfg.CodeEffectivePassed, r.Code())
}

func keyOf(t *testing.T, list admintest.Resp, key string) map[string]any {
	t.Helper()
	for _, k := range list.Data()["keys"].([]any) {
		if m := k.(map[string]any); m["key"] == key {
			return m
		}
	}
	t.Fatalf("key %s not listed", key)
	return nil
}

func redatingRows(firstUntil, firstLimit any) map[string]any {
	return map[string]any{"rows": []any{
		map[string]any{"ga_until_days": firstUntil, "max_diff_days": firstLimit},
		map[string]any{"ga_until_days": 97, "max_diff_days": 7},
		map[string]any{"ga_until_days": nil, "max_diff_days": 21},
	}}
}

// T-M10-16: rows keys in the list (kind, schema, v1 rows and source), proposals validated against the row schema,
// the unchanged two-advisor flow, and the approved rows and source read on and after the effective date.
func TestRowsKeys_ProposeApproveResolve(t *testing.T) {
	e := newEnv(t)
	day := today()
	const url = "/clinical-config/" + clinicalconfig.KeyPregnancyRedating + "/proposals"

	list := e.As(advisor1).Get("/clinical-config?group=pregnancy")
	require.Equal(t, 200, list.Status, list.Body)
	red := keyOf(t, list, clinicalconfig.KeyPregnancyRedating)
	assert.Equal(t, "rows", red["kind"])
	assert.Empty(t, red["fields"])
	schema := red["schema"].(map[string]any)
	assert.Len(t, schema["columns"], 2)
	assert.Equal(t, map[string]any{"column": "ga_until_days", "strict": true}, schema["sort_by"])
	cur := red["current"].(map[string]any)
	assert.EqualValues(t, 1, cur["version"])
	assert.Contains(t, cur["source"], "ACOG CO 700")
	assert.Len(t, cur["value"].(map[string]any)["rows"], 6)
	bp := keyOf(t, e.As(advisor1).Get("/clinical-config?group=vitals"), clinicalconfig.KeyBPCrisis)
	assert.Equal(t, "numbers", bp["kind"])
	assert.Nil(t, bp["schema"])
	for _, g := range []string{"child", "screening"} {
		assert.Equal(t, 200, e.As(advisor1).Get("/clinical-config?group="+g).Status, g)
	}

	// Validation: overlapping IOM bands, unsorted / out-of-range re-dating rows, a numbers value for a rows key, an
	// unknown screening test kind → 422 with the row in the message.
	iom := map[string]any{"rows": []any{
		map[string]any{"bmi_from": 0, "bmi_to": 20, "total_lo": 12.5, "total_hi": 18, "weekly_lo": 0.44, "weekly_hi": 0.58},
		map[string]any{"bmi_from": 18.5, "bmi_to": nil, "total_lo": 11.5, "total_hi": 16, "weekly_lo": 0.35, "weekly_hi": 0.5},
	}}
	r := e.As(advisor1).JSON(fiber.MethodPost, "/clinical-config/pregnancy.iom/proposals",
		map[string]any{"value": iom, "effective_from": day.String()})
	assert.Equal(t, 422, r.Status, r.Body)
	assert.Contains(t, r.Body["errors"].(map[string]any)["value"].([]any)[0], "Row 2 overlaps")
	for _, bad := range []any{redatingRows(120, 5), redatingRows(62, 50), redatingRows(62.5, 5), map[string]any{"days": 3},
		map[string]any{"rows": []any{}}} {
		r = e.As(advisor1).JSON(fiber.MethodPost, url, map[string]any{"value": bad, "effective_from": day.String()})
		assert.Equal(t, 422, r.Status, "%v → %s", bad, r.Body)
	}
	scr := map[string]any{"rows": []any{map[string]any{"type_key": "pap_smear", "age_min": 21, "age_max": 29,
		"interval_months": 36, "last_test_kind": "pap", "min_bmi": nil}}}
	r = e.As(advisor1).JSON(fiber.MethodPost, "/clinical-config/screening.uspstf/proposals",
		map[string]any{"value": scr, "effective_from": day.String()})
	assert.Equal(t, 422, r.Status, r.Body)
	assert.Equal(t, 0, e.Int(`SELECT COUNT(*) FROM clinical_config_versions`))

	// A valid proposal (a missing nullable column reads null), approved by two advisors, effective tomorrow.
	eff := day.AddDays(1)
	value := redatingRows(62, 6)
	value["rows"].([]any)[2] = map[string]any{"max_diff_days": 21}
	p := e.As(engineering).JSON(fiber.MethodPost, url, map[string]any{"value": value, "effective_from": eff.String(),
		"source": "ACOG CO 700 (advisor review)"})
	require.Equal(t, 201, p.Status, p.Body)
	assert.Equal(t, `{"rows":[{"ga_until_days":62,"max_diff_days":6},{"ga_until_days":97,"max_diff_days":7},{"ga_until_days":null,"max_diff_days":21}]}`,
		e.String(`SELECT value FROM clinical_config_versions WHERE config_key = ?`, clinicalconfig.KeyPregnancyRedating))
	id := proposalID(t, p)
	pend := keyOf(t, e.As(advisor1).Get("/clinical-config"), clinicalconfig.KeyPregnancyRedating)["pending"].(map[string]any)
	assert.Equal(t, "ACOG CO 700 (advisor review)", pend["source"])
	assert.Len(t, pend["value"].(map[string]any)["rows"], 3)
	assert.Equal(t, clinicalcfg.CodeAdvisorRequired, e.As(engineering).JSON(fiber.MethodPost, path(id, "approve"), nil).Code(),
		"the two-advisor flow is unchanged")
	require.Equal(t, 200, e.As(advisor1).JSON(fiber.MethodPost, path(id, "approve"), nil).Status)
	r = e.As(advisor2).JSON(fiber.MethodPost, path(id, "approve"), nil)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, clinicalcfg.StatusApproved, r.Obj("proposal")["status"])

	res := resolve(t, e, clinicalconfig.KeyPregnancyRedating, day)
	assert.EqualValues(t, 1, res["version"])
	assert.Len(t, res["value"].(map[string]any)["rows"], 6)
	res = resolve(t, e, clinicalconfig.KeyPregnancyRedating, eff)
	assert.EqualValues(t, 2, res["version"])
	assert.Equal(t, "ACOG CO 700 (advisor review)", res["source"])
	assert.Len(t, res["value"].(map[string]any)["rows"], 3)
	assert.Equal(t, 6.0, clinicalconfig.Rows(clinicalconfig.KeyPregnancyRedating, eff)[0]["max_diff_days"])
	assert.Equal(t, "ACOG CO 700 (advisor review)", clinicalconfig.SourceOf(clinicalconfig.KeyPregnancyRedating, eff.AddDays(1)))
	assert.Equal(t, "pregnancy.redating@2", clinicalconfig.Fingerprint(eff, clinicalconfig.KeyPregnancyRedating))

	// A broken stored rows value is never loaded: the key keeps its previous version.
	e.Exec(`INSERT INTO clinical_config_versions (config_key, version, value, effective_from, status, created_at, updated_at)
		VALUES ('pregnancy.iom', 2, '{"rows":[{"bmi_from":"x"}]}', ?, 'approved', NOW(), NOW())`, day.AddDays(-5))
	require.NoError(t, clinicalconfig.Refresh(t.Context()))
	assert.Equal(t, 1, clinicalconfig.At(clinicalconfig.KeyPregnancyIOM, day).Version)
	assert.EqualValues(t, 1, resolve(t, e, clinicalconfig.KeyPregnancyIOM, day)["version"])
	listed := keyOf(t, e.As(advisor1).Get("/clinical-config?group=pregnancy"), clinicalconfig.KeyPregnancyIOM)
	assert.EqualValues(t, 1, listed["current"].(map[string]any)["version"], "the list shows what the engines read")
}
