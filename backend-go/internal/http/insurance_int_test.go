package http

import (
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const insP = "/api/v1/insurance"

func itoaU(n uint64) string { return strconv.FormatUint(n, 10) }

func (e *companionEnv) insExec(q string, args ...any) uint64 {
	e.t.Helper()
	res, err := e.db.Exec(q, args...)
	require.NoError(e.t, err)
	id, _ := res.LastInsertId()
	return uint64(id) //nolint:gosec // test ids
}

func insPolicy(insurerID uint64, childID uint64) map[string]any {
	return map[string]any{
		"insurer_id": insurerID, "plan_name": "Gold", "policy_number": "PN-99-۴۸۲۱", "iban": "IR12 0000 0000 0000 1234",
		"valid_from": "2026-03-21", "valid_to": "2027-03-20", "claim_deadline_days": 15,
		"not_covered_note": "Cosmetic surgery",
		"members": []any{
			map[string]any{"kind": "self", "name": "Sara"},
			map[string]any{"kind": "spouse", "name": "Ali"},
			map[string]any{"kind": "child", "child_id": childID, "name": "Ava"},
		},
		"coverages": []any{
			map[string]any{"category": "paraclinic", "cap_amount": 20000000, "deductible_pct": 30},
			map[string]any{"category": "delivery", "cap_amount": 40000000, "deductible_pct": 10, "waiting_days": 270},
			map[string]any{"category": "optical", "status": "not_covered"},
		},
	}
}

// Routes, validation, coverage maths, the user-maintained timeline and strict owner isolation (no IDOR) of the
// insurance self-tracking API (CB-INS-01).
func TestInsurance_PolicyClaimsAndIsolation(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120009101")
	nina := e.user("Nina", "09120009102")

	for _, path := range []string{insP + "/policy", insP + "/claims", insP + "/questionnaires", insP + "/centres", insP + "/insurers"} {
		assert.Equal(t, fiber.StatusUnauthorized, e.do(fiber.MethodGet, path, "", nil).Status, path)
	}

	insurer := e.insExec(`INSERT INTO insurance_insurers (code, name, is_active, sort_order, created_at, updated_at) VALUES ('alpha', 'Alpha Insurance', 1, 1, NOW(), NOW())`)
	inactive := e.insExec(`INSERT INTO insurance_insurers (code, name, is_active, sort_order, created_at, updated_at) VALUES ('beta', 'Beta', 0, 2, NOW(), NOW())`)
	e.insExec(`INSERT INTO insurance_centres (insurer_id, name, type, lat, lng, is_direct, is_active, sort_order, created_at, updated_at) VALUES (?, 'Alpha Lab', 'lab', 35.7, 51.4, 1, 1, 1, NOW(), NOW())`, insurer)
	e.insExec(`INSERT INTO insurance_centres (insurer_id, name, type, is_direct, is_active, sort_order, created_at, updated_at) VALUES (?, 'Beta Clinic', 'gyn', 0, 1, 2, NOW(), NOW())`, inactive)
	centre := e.insExec(`INSERT INTO insurance_centres (insurer_id, name, type, is_direct, is_active, sort_order, created_at, updated_at) VALUES (NULL, 'City Imaging', 'imaging', 0, 1, 3, NOW(), NOW())`)
	child := e.insExec(`INSERT INTO children (owner_id, name, birth_date, created_at, updated_at) VALUES (?, 'Ava', '2024-01-01', NOW(), NOW())`, e.ids["Sara"])
	ninaChild := e.insExec(`INSERT INTO children (owner_id, name, birth_date, created_at, updated_at) VALUES (?, 'Nim', '2024-01-01', NOW(), NOW())`, e.ids["Nina"])
	doc := e.insExec(`INSERT INTO record_documents (user_id, kind, title, document_date, review_state, created_at, updated_at) VALUES (?, 'imaging', 'Ultrasound report', '2026-09-20', 'manual', NOW(), NOW())`, e.ids["Sara"])
	ninaDoc := e.insExec(`INSERT INTO record_documents (user_id, kind, title, document_date, review_state, created_at, updated_at) VALUES (?, 'imaging', 'Nina scan', '2026-09-20', 'manual', NOW(), NOW())`, e.ids["Nina"])

	// No policy yet: policy null with her children; claims need a policy.
	r := e.do(fiber.MethodGet, insP+"/policy", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Nil(t, r.data()["policy"])
	assert.Len(t, r.data()["children"], 1)
	assert.Equal(t, "insurance_no_policy", e.do(fiber.MethodPost, insP+"/claims", sara, map[string]any{}).Body["error_code"])

	// Validation: an inactive insurer, another user's child, two «self» members, duplicate categories.
	bad := insPolicy(inactive, child)
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPut, insP+"/policy", sara, bad).Status)
	bad = insPolicy(insurer, ninaChild)
	got := e.do(fiber.MethodPut, insP+"/policy", sara, bad)
	assert.Equal(t, fiber.StatusUnprocessableEntity, got.Status)
	assert.Contains(t, got.Raw, "members.2.child_id")
	bad = insPolicy(insurer, child)
	bad["members"] = []any{map[string]any{"kind": "self", "name": "A"}, map[string]any{"kind": "self", "name": "B"}}
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPut, insP+"/policy", sara, bad).Status)
	bad = insPolicy(insurer, child)
	bad["coverages"] = []any{map[string]any{"category": "dental"}, map[string]any{"category": "dental"}}
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPut, insP+"/policy", sara, bad).Status)
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPut, insP+"/policy", sara, map[string]any{}).Status, "insurer required")

	r = e.do(fiber.MethodPut, insP+"/policy", sara, insPolicy(insurer, child))
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	pol := r.data()["policy"].(map[string]any)
	assert.Equal(t, "4821", pol["number_last4"])
	assert.Equal(t, "1234", pol["iban_last4"])
	assert.NotContains(t, r.Raw, "PN-99", "the full number is never stored or returned")
	members := pol["members"].([]any)
	require.Len(t, members, 3)
	saraMember := idOf(members[0].(map[string]any)["id"])
	assert.Len(t, pol["coverages"], 3)

	// Re-saving keeps member ids, number and IBAN when omitted.
	again := insPolicy(insurer, child)
	delete(again, "policy_number")
	delete(again, "iban")
	again["members"] = []any{map[string]any{"id": saraMember, "kind": "self", "name": "Sara K"}}
	r = e.do(fiber.MethodPut, insP+"/policy", sara, again)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	pol = r.data()["policy"].(map[string]any)
	assert.Equal(t, "4821", pol["number_last4"])
	assert.Len(t, pol["members"], 1)
	assert.Equal(t, float64(saraMember), pol["members"].([]any)[0].(map[string]any)["id"])

	// Centres: her insurer's + the ones for every insurer, never another insurer's.
	r = e.do(fiber.MethodGet, insP+"/centres", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Len(t, r.data()["centres"], 2)
	assert.NotContains(t, r.Raw, "Beta Clinic")
	assert.Len(t, e.do(fiber.MethodGet, insP+"/centres?type=lab", sara, nil).data()["centres"], 1)
	assert.Len(t, e.do(fiber.MethodGet, insP+"/insurers", sara, nil).data()["insurers"], 1)

	// Claims: validation (future day, another user's document, unknown member).
	claim := map[string]any{"member_id": saraMember, "cost_type": "paraclinic", "title": "Pregnancy ultrasound",
		"centre_id": centre, "service_date": "2026-09-20", "amount": 1800000, "note": "private claim note",
		"documents": []any{map[string]any{"id": doc, "label": "report"}}}
	future := map[string]any{"cost_type": "paraclinic", "service_date": "2026-12-01", "amount": 10}
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, insP+"/claims", sara, future).Status)
	foreign := map[string]any{"cost_type": "paraclinic", "service_date": "2026-09-20", "amount": 10,
		"documents": []any{map[string]any{"id": ninaDoc}}}
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, insP+"/claims", sara, foreign).Status)
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, insP+"/claims", sara,
		map[string]any{"member_id": 99999, "cost_type": "visit", "service_date": "2026-09-20", "amount": 10}).Status)

	r = e.do(fiber.MethodPost, insP+"/claims", sara, claim)
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	cid := idOf(r.data()["id"])
	assert.Equal(t, "City Imaging", r.data()["centre_name"])
	assert.Equal(t, "private claim note", r.data()["note"])
	pay := r.data()["payable"].(map[string]any)
	assert.Equal(t, float64(540000), pay["deductible"])
	assert.Equal(t, float64(1260000), pay["payable"])
	assert.Len(t, r.data()["events"], 1)
	assert.Len(t, r.data()["documents"], 1)
	assert.Equal(t, "2026-10-05", r.data()["submit_by"])
	var stored string
	require.NoError(t, e.db.QueryRow(`SELECT notes_enc FROM insurance_claims WHERE id = ?`, cid).Scan(&stored))
	assert.NotContains(t, stored, "private claim note", "the note is sealed at rest")

	// Timeline she maintains: docs requested with a deadline → needs action; paid → coverage used.
	c := insP + "/claims/" + itoaU(cid)
	r = e.do(fiber.MethodPost, c+"/events", sara, map[string]any{"status": "docs_requested", "occurred_on": "2026-09-23",
		"deadline_on": "2026-10-02", "note": "send the report"})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	assert.Equal(t, "docs_requested", r.data()["status"])
	assert.Equal(t, "2026-10-02", r.data()["deadline_on"])
	list := e.do(fiber.MethodGet, insP+"/claims", sara, nil)
	totals := list.data()["totals"].(map[string]any)
	assert.Equal(t, float64(1), totals["needs_action"])
	assert.Equal(t, float64(1800000), totals["requested"])
	assert.Len(t, e.do(fiber.MethodGet, insP+"/claims?status=paid", sara, nil).data()["claims"], 0)
	assert.Len(t, e.do(fiber.MethodGet, insP+"/claims?status=needs_action&q=ultra", sara, nil).data()["claims"], 1)
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, c+"/events", sara,
		map[string]any{"status": "paid", "occurred_on": "2026-09-23"}).Status, "a paid step needs its amount")
	r = e.do(fiber.MethodPost, c+"/events", sara, map[string]any{"status": "paid", "occurred_on": "2026-09-23", "amount": 1260000})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	assert.Equal(t, "paid", r.data()["status"])
	assert.Equal(t, float64(1260000), r.data()["paid_amount"])
	evs := r.data()["events"].([]any)
	require.Len(t, evs, 3)
	pol = e.do(fiber.MethodGet, insP+"/policy", sara, nil).data()["policy"].(map[string]any)
	para := pol["coverages"].([]any)[0].(map[string]any)
	assert.Equal(t, float64(1260000), para["used"])
	assert.Equal(t, float64(18740000), para["remaining"])

	// Removing the paid step re-derives the status; the first step cannot be removed.
	paidID := idOf(evs[2].(map[string]any)["id"])
	r = e.do(fiber.MethodDelete, c+"/events/"+itoaU(paidID), sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "docs_requested", r.data()["status"])
	assert.Nil(t, r.data()["paid_amount"])

	// Documents: detach, re-attach; another user's document answers 404.
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodDelete, c+"/documents/"+itoaU(doc), sara, nil).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPost, c+"/documents", sara, map[string]any{"document_id": ninaDoc}).Status)
	r = e.do(fiber.MethodPost, c+"/documents", sara, map[string]any{"document_id": doc, "label": "report"})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Len(t, r.data()["documents"], 1)

	// Owner-only: Nina gets 404 everywhere and sees nothing.
	assert.Nil(t, e.do(fiber.MethodGet, insP+"/policy", nina, nil).data()["policy"])
	for _, cse := range [][2]string{{fiber.MethodGet, c}, {fiber.MethodPut, c}, {fiber.MethodDelete, c},
		{fiber.MethodPost, c + "/events"}, {fiber.MethodDelete, c + "/events/" + itoaU(paidID)},
		{fiber.MethodPost, c + "/documents"}, {fiber.MethodDelete, c + "/documents/" + itoaU(doc)}} {
		got := e.do(cse[0], cse[1], nina, map[string]any{"status": "paid", "occurred_on": "2026-09-23", "amount": 1, "document_id": doc})
		assert.Contains(t, []int{fiber.StatusNotFound}, got.Status, cse[0]+" "+cse[1]+" "+got.Raw)
	}
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, insP+"/claims/abc", sara, nil).Status)

	// Update + delete.
	upd := map[string]any{"cost_type": "visit", "service_date": "2026-09-19", "amount": 500000}
	r = e.do(fiber.MethodPut, c, sara, upd)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "visit_drug", r.data()["category"])
	assert.Len(t, r.data()["documents"], 1, "documents kept when not sent")
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodDelete, c, sara, nil).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, c, sara, nil).Status)

	// Deleting the policy removes everything under it.
	e.do(fiber.MethodPost, insP+"/claims", sara, map[string]any{"cost_type": "drug", "service_date": "2026-09-20", "amount": 10})
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodDelete, insP+"/policy", sara, nil).Status)
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM insurance_claims WHERE user_id = ?`, e.ids["Sara"]).Scan(&n))
	assert.Zero(t, n)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodDelete, insP+"/policy", sara, nil).Status)
}

// The questionnaire is prefilled from her own record only, confirmed by her, then advanced by her (CB-INS-01).
func TestInsurance_Questionnaire(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120009111")
	nina := e.user("Nina", "09120009112")
	e.insExec(`INSERT INTO user_life_profiles (user_id, life_mode, ivf_iui, chronic_illnesses, medications, created_at, updated_at)
		VALUES (?, 'cycle', 0, '["thyroid"]', '[]', NOW(), NOW())`, e.ids["Sara"])
	e.insExec(`INSERT INTO health_records (user_id, surgeries, family_history, created_at, updated_at)
		VALUES (?, '[{"title":"Appendix","date":"2024-05-01"}]', '[]', NOW(), NOW())`, e.ids["Sara"])
	e.insExec(`INSERT INTO user_life_profiles (user_id, life_mode, ivf_iui, chronic_illnesses, created_at, updated_at)
		VALUES (?, 'pregnancy', 1, '["diabetes"]', NOW(), NOW())`, e.ids["Nina"])

	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, insP+"/questionnaires", sara, map[string]any{"purpose": "x"}).Status)
	r := e.do(fiber.MethodPost, insP+"/questionnaires", sara, map[string]any{"purpose": "renewal", "label": "Renewal 1405"})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	qid := idOf(r.data()["id"])
	byCode := map[string]map[string]any{}
	for _, q := range r.data()["questions"].([]any) {
		m := q.(map[string]any)
		byCode[m["code"].(string)] = m
	}
	assert.Equal(t, "yes", byCode["chronic_disease"]["answer"])
	assert.Equal(t, true, byCode["chronic_disease"]["from_record"])
	assert.Equal(t, "yes", byCode["hospitalized_5y"]["answer"])
	assert.Equal(t, "no", byCode["regular_medication"]["answer"])
	assert.Equal(t, "no", byCode["pregnant"]["answer"], "her record, not Nina's")
	assert.Equal(t, "no", byCode["infertility_treatment"]["answer"])
	assert.Nil(t, byCode["smoker"]["answer"])
	assert.NotContains(t, r.Raw, "diabetes")

	q := insP + "/questionnaires/" + itoaU(qid)
	assert.Equal(t, fiber.StatusConflict, e.do(fiber.MethodPost, q+"/status", sara, map[string]any{"status": "submitted", "occurred_on": "2026-09-23"}).Status)
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, q+"/confirm", sara, nil).Status, "unanswered rows")
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPut, q, sara, map[string]any{"answers": map[string]any{"smoker": "maybe"}}).Status)
	r = e.do(fiber.MethodPut, q, sara, map[string]any{"answers": map[string]any{"smoker": "no", "disability": "no",
		"gyn_surgery": "no", "regular_medication": "yes"}, "explanation": "Thyroid: hypothyroid since 1401"})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, true, r.data()["complete"])
	assert.Equal(t, "Thyroid: hypothyroid since 1401", r.data()["explanation"])
	var enc string
	require.NoError(t, e.db.QueryRow(`SELECT explanation_enc FROM insurance_questionnaires WHERE id = ?`, qid).Scan(&enc))
	assert.NotContains(t, enc, "hypothyroid")

	// The evidence copied from her record is sealed at rest (00093): the API shows it, the row holds no plaintext.
	assert.Contains(t, r.Raw, "thyroid")
	var answers, evidence string
	require.NoError(t, e.db.QueryRow(`SELECT answers, evidence_enc FROM insurance_questionnaires WHERE id = ?`, qid).Scan(&answers, &evidence))
	for _, plain := range []string{"thyroid", "Appendix"} {
		assert.NotContains(t, answers, plain)
		assert.NotContains(t, evidence, plain)
	}
	assert.NotContains(t, answers, "evidence")
	assert.NotEmpty(t, evidence)

	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPost, q+"/confirm", sara, nil).Status)
	r = e.do(fiber.MethodPost, q+"/status", sara, map[string]any{"status": "submitted", "occurred_on": "2026-09-23"})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "submitted", r.data()["status"])
	assert.Len(t, r.data()["timeline"], 1)
	assert.Equal(t, fiber.StatusConflict, e.do(fiber.MethodPut, q, sara, map[string]any{"answers": map[string]any{"smoker": "yes"}}).Status)

	for _, cse := range [][2]string{{fiber.MethodGet, q}, {fiber.MethodPut, q}, {fiber.MethodDelete, q},
		{fiber.MethodPost, q + "/confirm"}, {fiber.MethodPost, q + "/status"}} {
		assert.Equal(t, fiber.StatusNotFound, e.do(cse[0], cse[1], nina, map[string]any{"status": "reviewing", "occurred_on": "2026-09-23"}).Status, cse[1])
	}
	assert.Empty(t, e.do(fiber.MethodGet, insP+"/questionnaires", nina, nil).data()["questionnaires"])
	assert.Len(t, e.do(fiber.MethodGet, insP+"/questionnaires", sara, nil).data()["questionnaires"], 1)
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodDelete, q, sara, nil).Status)
}
