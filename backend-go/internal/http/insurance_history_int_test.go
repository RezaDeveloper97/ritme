package http

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Policy history (user decision 2026-10-12, 00093): renewal creates a new policy, the previous one turns past and
// read-only together with its claims, usage is per policy, and every policy id is owner-only (no IDOR).
func TestInsurance_PolicyHistory(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120009121")
	nina := e.user("Nina", "09120009122")

	// Her first policy covers last year only: with nothing else it is still her current policy.
	first := map[string]any{"insurer_name": "Alpha", "plan_name": "Silver", "policy_number": "77-1234",
		"valid_from": "2025-03-21", "valid_to": "2026-03-20", "claim_deadline_days": 15,
		"members":   []any{map[string]any{"kind": "self", "name": "Sara"}, map[string]any{"kind": "spouse", "name": "Ali"}},
		"coverages": []any{map[string]any{"category": "paraclinic", "cap_amount": 20000000, "deductible_pct": 30}}}
	r := e.do(fiber.MethodPut, insP+"/policy", sara, first)
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	p1 := idOf(r.data()["policy"].(map[string]any)["id"])
	assert.Equal(t, "current", r.data()["policy"].(map[string]any)["status"])

	r = e.do(fiber.MethodPost, insP+"/claims", sara, map[string]any{"cost_type": "paraclinic", "service_date": "2026-01-10", "amount": 1000000})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	c1 := idOf(r.data()["id"])
	assert.Equal(t, float64(p1), r.data()["policy_id"])
	c := insP + "/claims/" + itoaU(c1)
	require.Equal(t, fiber.StatusCreated, e.do(fiber.MethodPost, c+"/events", sara,
		map[string]any{"status": "paid", "occurred_on": "2026-09-23", "amount": 700000}).Status)

	// Renew: the next period (day after the old end, same length), members and coverages copied.
	r = e.do(fiber.MethodPost, insP+"/policies/"+itoaU(p1)+"/renew", sara, map[string]any{})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	pol := r.data()["policy"].(map[string]any)
	p2 := idOf(pol["id"])
	assert.NotEqual(t, p1, p2)
	assert.Equal(t, "2026-03-21", pol["valid_from"])
	assert.Equal(t, "2027-03-20", pol["valid_to"])
	assert.Equal(t, "1234", pol["number_last4"])
	assert.Equal(t, "current", pol["status"])
	assert.Len(t, pol["members"], 2)
	require.Len(t, pol["coverages"], 1)
	assert.Equal(t, float64(0), pol["coverages"].([]any)[0].(map[string]any)["used"], "usage is per policy period")

	// History: two policies, the old one past / read-only with its claim.
	r = e.do(fiber.MethodGet, insP+"/policies", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, float64(p2), r.data()["current_id"])
	items := r.data()["policies"].([]any)
	require.Len(t, items, 2)
	newest, oldest := items[0].(map[string]any), items[1].(map[string]any)
	assert.Equal(t, float64(p2), newest["id"])
	assert.Equal(t, true, newest["can_renew"])
	assert.Equal(t, float64(0), newest["claims_count"])
	assert.Equal(t, "past", oldest["status"])
	assert.Equal(t, true, oldest["read_only"])
	assert.Equal(t, float64(1), oldest["claims_count"])

	assert.Equal(t, float64(p2), idOfF(e.do(fiber.MethodGet, insP+"/policy", sara, nil).data()["policy"]), "the home reads the current one")
	r = e.do(fiber.MethodGet, insP+"/policies/"+itoaU(p1), sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	old := r.data()["policy"].(map[string]any)
	assert.Equal(t, true, old["read_only"])
	assert.Equal(t, float64(700000), old["coverages"].([]any)[0].(map[string]any)["used"])

	// The past policy and its claims are read-only; only the newest can be renewed.
	assert.Equal(t, "insurance_policy_renewed", e.do(fiber.MethodPost, insP+"/policies/"+itoaU(p1)+"/renew", sara, nil).Body["error_code"])
	assert.Equal(t, "insurance_policy_read_only", e.do(fiber.MethodPut, insP+"/policies/"+itoaU(p1), sara, first).Body["error_code"])
	r = e.do(fiber.MethodGet, c, sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, float64(p1), r.data()["policy_id"])
	assert.Equal(t, true, r.data()["read_only"])
	evID := uint64(idOfF(r.data()["events"].([]any)[0]))
	for _, cse := range [][2]string{{fiber.MethodPut, c}, {fiber.MethodDelete, c}, {fiber.MethodDelete, c + "/events/" + itoaU(evID)},
		{fiber.MethodPost, c + "/documents"}, {fiber.MethodDelete, c + "/documents/1"}} {
		got := e.do(cse[0], cse[1], sara, map[string]any{"cost_type": "visit", "service_date": "2026-01-10", "amount": 1,
			"status": "reviewing", "occurred_on": "2026-01-21", "document_id": 1})
		assert.Equal(t, fiber.StatusConflict, got.Status, cse[0]+" "+cse[1]+" "+got.Raw)
		assert.Equal(t, "insurance_policy_read_only", got.Body["error_code"])
	}
	// User decision 2026-10-12: her past policy's claim still takes timeline steps (a late claim paid after renewal).
	r = e.do(fiber.MethodPost, c+"/events", sara, map[string]any{"status": "appealed", "occurred_on": "2026-09-23", "note": "late step"})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	assert.Equal(t, true, r.data()["read_only"], "the claim itself stays read-only")
	assert.Len(t, r.data()["events"], 3)
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, c+"/events", sara,
		map[string]any{"status": "paid", "occurred_on": "2026-09-22"}).Status, "steps are still validated")
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPost, c+"/events", nina,
		map[string]any{"status": "paid", "occurred_on": "2026-09-22", "amount": 1}).Status, "owner-only")

	// Claims stay with their policy: the current list is empty, the old one keeps its claim; new claims go to p2.
	r = e.do(fiber.MethodGet, insP+"/claims", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Empty(t, r.data()["claims"])
	assert.Equal(t, float64(p2), idOfF(r.data()["policy"]))
	r = e.do(fiber.MethodGet, insP+"/claims?policy_id="+itoaU(p1), sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Len(t, r.data()["claims"], 1)
	assert.Equal(t, true, r.data()["policy"].(map[string]any)["read_only"])
	r = e.do(fiber.MethodPost, insP+"/claims", sara, map[string]any{"cost_type": "visit", "service_date": "2026-09-20", "amount": 300000})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	assert.Equal(t, float64(p2), r.data()["policy_id"])
	assert.Equal(t, false, r.data()["read_only"])

	// Owner-only: Nina sees no history and every one of Sara's policy ids answers 404.
	r = e.do(fiber.MethodGet, insP+"/policies", nina, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Empty(t, r.data()["policies"])
	assert.Nil(t, r.data()["current_id"])
	for _, id := range []uint64{p1, p2} {
		pp := insP + "/policies/" + itoaU(id)
		for _, cse := range [][2]string{{fiber.MethodGet, pp}, {fiber.MethodPut, pp}, {fiber.MethodDelete, pp},
			{fiber.MethodPost, pp + "/renew"}, {fiber.MethodGet, insP + "/claims?policy_id=" + itoaU(id)}} {
			got := e.do(cse[0], cse[1], nina, first)
			assert.Equal(t, fiber.StatusNotFound, got.Status, cse[0]+" "+cse[1]+" "+got.Raw)
		}
	}
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, insP+"/claims?policy_id=abc", sara, nil).Status)

	// Early renewal: p2 still contains today so it stays current; p3 is upcoming and editable; p2 can't be renewed twice.
	r = e.do(fiber.MethodPost, insP+"/policies/"+itoaU(p2)+"/renew", sara, map[string]any{"valid_from": "2027-03-21", "valid_to": "2027-01-01"})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	r = e.do(fiber.MethodPost, insP+"/policies/"+itoaU(p2)+"/renew", sara, nil)
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	p3 := idOf(r.data()["policy"].(map[string]any)["id"])
	assert.Equal(t, "upcoming", r.data()["policy"].(map[string]any)["status"])
	assert.Equal(t, float64(p2), idOfF(e.do(fiber.MethodGet, insP+"/policy", sara, nil).data()["policy"]))
	upd := map[string]any{"insurer_name": "Beta", "valid_from": "2027-03-21", "valid_to": "2028-03-19",
		"members": []any{map[string]any{"kind": "self", "name": "Sara"}}}
	r = e.do(fiber.MethodPut, insP+"/policies/"+itoaU(p3), sara, upd)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "Beta", r.data()["policy"].(map[string]any)["insurer_name"])
	assert.Equal(t, "insurance_policy_renewed", e.do(fiber.MethodPost, insP+"/policies/"+itoaU(p2)+"/renew", sara, nil).Body["error_code"])

	// A policy added by hand (another insurer) joins the history.
	r = e.do(fiber.MethodPost, insP+"/policies", sara, map[string]any{"insurer_name": "Gamma", "valid_from": "2024-03-20", "valid_to": "2025-03-20"})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	assert.Equal(t, "past", r.data()["policy"].(map[string]any)["status"])
	assert.Len(t, e.do(fiber.MethodGet, insP+"/policies", sara, nil).data()["policies"], 4)

	// Deleting a past policy is her right: it goes with its claims, the others keep theirs.
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodDelete, insP+"/policies/"+itoaU(p1), sara, nil).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, c, sara, nil).Status)
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM insurance_claims WHERE user_id = ?`, e.ids["Sara"]).Scan(&n))
	assert.Equal(t, 1, n)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodDelete, insP+"/policies/"+itoaU(p1), sara, nil).Status)
}

func idOfF(v any) float64 {
	m, ok := v.(map[string]any)
	if !ok {
		return 0
	}
	f, _ := m["id"].(float64)
	return f
}

// A questionnaire written before 00093 (plain evidence in answers) stays readable and is sealed on its next save.
func TestInsurance_QuestionnaireLegacyEvidence(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120009131")
	nina := e.user("Nina", "09120009132")
	qid := e.insExec(`INSERT INTO insurance_questionnaires (user_id, purpose, status, answers, created_at, updated_at)
		VALUES (?, 'issue', 'draft', '[{"code":"chronic_disease","answer":"yes","record_answer":"yes","evidence":["legacy-asthma"]}]', NOW(), NOW())`, e.ids["Sara"])
	q := insP + "/questionnaires/" + itoaU(qid)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, q, nina, nil).Status)
	r := e.do(fiber.MethodGet, q, sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Contains(t, r.Raw, "legacy-asthma")

	r = e.do(fiber.MethodPut, q, sara, map[string]any{"answers": map[string]any{"smoker": "no"}})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Contains(t, r.Raw, "legacy-asthma")
	var answers, evidence string
	require.NoError(t, e.db.QueryRow(`SELECT answers, evidence_enc FROM insurance_questionnaires WHERE id = ?`, qid).Scan(&answers, &evidence))
	assert.NotContains(t, answers, "legacy-asthma")
	assert.NotContains(t, evidence, "legacy-asthma")
	assert.NotEmpty(t, evidence)
}
