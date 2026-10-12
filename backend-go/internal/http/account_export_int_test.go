package http

import (
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/insurance"
	"github.com/ritme/backend-go/internal/loss"
)

// seedCanvasData gives the user one row (or a small family of rows) in every canvas table the account export reads;
// every free text carries tag, and the sealed text is sealed the way the domains seal it (development key, owner AD).
func seedCanvasData(t *testing.T, e *companionEnv, uid uint64, tag string) {
	t.Helper()
	box, err := loss.NewNoteBox(nil, nil, false)
	require.NoError(t, err)
	u := strconv.FormatUint(uid, 10)
	seal := func(plain, ad string) string {
		s, err := box.SealWith(plain, []byte(ad))
		require.NoError(t, err)
		return s
	}
	x := e.insExec

	// pelvic floor
	x(`INSERT INTO pelvic_programs (user_id, started_on, created_at, updated_at) VALUES (?, '2026-09-01', NOW(), NOW())`, uid)
	x(`INSERT INTO pelvic_sessions (user_id, session_date, sessions_count, sets_completed, duration_sec, level_code, created_at, updated_at) VALUES (?, '2026-09-02', 1, 3, 300, ?, NOW(), NOW())`, uid, tag+"-level")
	x(`INSERT INTO pelvic_bladder_logs (user_id, log_date, leak, night_voids, uti_symptoms, created_at, updated_at) VALUES (?, '2026-09-02', 'none', 1, '["burning"]', NOW(), NOW())`, uid)
	// contraception
	x(`INSERT INTO contraception_methods (user_id, method, pack_type, pack_started_on, followup_done, created_at, updated_at) VALUES (?, 'pill', '21', '2026-09-01', 1, NOW(), NOW())`, uid)
	x(`INSERT INTO contraception_pill_logs (user_id, log_date, status, logged_at, created_at, updated_at) VALUES (?, '2026-09-02', 'taken', '2026-09-02 09:00:00', NOW(), NOW())`, uid)
	// menopause
	x(`INSERT INTO user_life_profiles (user_id, life_mode, menopause_stage, menopause_surgical, created_at, updated_at) VALUES (?, 'menopause', 'peri', 0, NOW(), NOW())`, uid)
	x(`INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_text, created_at, updated_at) VALUES (?, '2026-09-02', 'symptoms', 'general', 'hot_flashes', ?, NOW(), NOW())`, uid, tag+"-log")
	x(`INSERT INTO hot_flashes (user_id, started_at, duration_s, severity, night, sweat, triggers, created_at, updated_at) VALUES (?, '2026-09-02 03:00:00', 120, 2, 1, 0, '["coffee"]', NOW(), NOW())`, uid)
	x(`INSERT INTO menopause_scores (user_id, month, answers, total, somatic, psychological, urogenital, created_at, updated_at) VALUES (?, '2026-09-01', '{"q1":2}', 2, 2, 0, 0, NOW(), NOW())`, uid)
	item := x(`INSERT INTO treatment_items (user_id, kind, name, created_at, updated_at) VALUES (?, 'medication', ?, NOW(), NOW())`, uid, tag+"-hrt")
	x(`INSERT INTO treatment_intakes (user_id, treatment_item_id, intake_date, amount, taken_at, created_at, updated_at) VALUES (?, ?, '2026-09-02', 1, '2026-09-02 08:00:00', NOW(), NOW())`, uid, item)
	x(`INSERT INTO side_effect_logs (user_id, treatment_item_id, log_date, code, created_at, updated_at) VALUES (?, ?, '2026-09-02', 'nausea', NOW(), NOW())`, uid, item)
	// IVF
	cycle := x(`INSERT INTO ivf_cycles (user_id, number, protocol, stage, started_on, outcome, created_at, updated_at) VALUES (?, 1, 'antagonist', 'stim', '2026-09-01', NULL, NOW(), NOW())`, uid)
	rem := x(`INSERT INTO reminders (user_id, type, title, recurrence, is_active, meta, created_at, updated_at) VALUES (?, 'medication', ?, 'daily', 1, '{"v":1}', NOW(), NOW())`, uid, tag+"-med")
	med := x(`INSERT INTO ivf_meds (user_id, cycle_id, reminder_id, role, route, stock_units, created_at, updated_at) VALUES (?, ?, ?, 'stim', 'subcut', 4, NOW(), NOW())`, uid, cycle, rem)
	x(`INSERT INTO ivf_dose_logs (user_id, ivf_med_id, dose_date, slot, site, created_at, updated_at) VALUES (?, ?, '2026-09-02', '08:00', 'belly_left', NOW(), NOW())`, uid, med)
	x(`INSERT INTO ivf_scans (user_id, cycle_id, scan_date, right_10_14, left_18_plus, endometrium_mm, notes, created_at, updated_at) VALUES (?, ?, '2026-09-05', 3, 1, 8.5, ?, NOW(), NOW())`, uid, cycle, tag+"-scan")
	x(`INSERT INTO ivf_tww_logs (user_id, cycle_id, log_date, mood, created_at, updated_at) VALUES (?, ?, '2026-09-06', 'hopeful', NOW(), NOW())`, uid, cycle)
	// condition programs
	x(`INSERT INTO condition_enrolments (user_id, program, enrolled_on, created_at, updated_at) VALUES (?, 'endometriosis', '2026-09-01', NOW(), NOW())`, uid)
	x(`INSERT INTO condition_pain_entries (user_id, entry_date, pain_types, missed_activity, analgesic, created_at, updated_at) VALUES (?, '2026-09-02', '["cramps"]', 1, ?, NOW(), NOW())`, uid, tag+"-pill")
	x(`INSERT INTO pmdd_entries (user_id, entry_date, scores, created_at, updated_at) VALUES (?, '2026-09-02', '{"mood":3}', NOW(), NOW())`, uid)
	x(`INSERT INTO pbac_entries (user_id, entry_date, light_count, flooding, created_at, updated_at) VALUES (?, '2026-09-02', 2, 1, NOW(), NOW())`, uid)
	// loss path: the private note is sealed per owner + loss
	lossID := x(`INSERT INTO pregnancy_losses (user_id, loss_type, occurred_on, paused_reminders, created_at, updated_at) VALUES (?, 'chemical', '2026-08-20', '[]', NOW(), NOW())`, uid)
	note, err := box.Seal(tag+"-loss-note", uid, lossID)
	require.NoError(t, err)
	_, err = e.db.Exec(`UPDATE pregnancy_losses SET private_note = ? WHERE id = ?`, note, lossID)
	require.NoError(t, err)
	x(`INSERT INTO pregnancy_loss_moods (user_id, loss_id, log_date, mood, created_at, updated_at) VALUES (?, ?, '2026-08-25', 'low', NOW(), NOW())`, uid, lossID)
	// insurance: notes, step notes, explanation and evidence sealed (AD insurance:<kind>:<uid>)
	pol := x(`INSERT INTO insurance_policies (user_id, insurer_name, plan_name, valid_from, valid_to, created_at, updated_at) VALUES (?, ?, 'Gold', '2026-03-21', '2027-03-20', NOW(), NOW())`, uid, tag+"-insurer")
	x(`INSERT INTO insurance_policy_members (policy_id, kind, name, sort_order, created_at, updated_at) VALUES (?, 'self', ?, 1, NOW(), NOW())`, pol, tag+"-member")
	x(`INSERT INTO insurance_coverages (policy_id, category, status, cap_amount, sort_order, created_at, updated_at) VALUES (?, 'paraclinic', 'covered', 1000, 1, NOW(), NOW())`, pol)
	claim := x(`INSERT INTO insurance_claims (user_id, policy_id, cost_type, service_date, amount, status, notes_enc, created_at, updated_at) VALUES (?, ?, 'visit', '2026-09-01', 500, 'submitted', ?, NOW(), NOW())`,
		uid, pol, seal(tag+"-claim-note", "insurance:claim:"+u))
	x(`INSERT INTO insurance_claim_events (claim_id, status, occurred_on, note_enc, created_at, updated_at) VALUES (?, 'submitted', '2026-09-01', ?, NOW(), NOW())`,
		claim, seal(tag+"-step-note", "insurance:claim_event:"+u))
	code := insurance.Questions[0].Code
	x(`INSERT INTO insurance_questionnaires (user_id, purpose, status, answers, evidence_enc, explanation_enc, created_at, updated_at) VALUES (?, 'new', 'draft', ?, ?, ?, NOW(), NOW())`,
		uid, `[{"code":"`+code+`","answer":"no","record_answer":"no"}]`, seal(`{"`+code+`":["`+tag+`-evidence"]}`, "insurance:questionnaire:"+u),
		seal(tag+"-explanation", "insurance:questionnaire:"+u))
	// shop: buyer PII sealed (AD shop-address:<uid> / shop-order:<uid>), plaintext columns ''
	x(`INSERT INTO shop_addresses (user_id, recipient_name, recipient_phone, city, address_line, pii_enc, is_default, created_at, updated_at) VALUES (?, '', '', 'Tehran', '', ?, 1, NOW(), NOW())`,
		uid, seal(`{"name":"`+tag+`-recipient","phone":"0912","line":"`+tag+`-street"}`, "shop-address:"+u))
	order := x(`INSERT INTO shop_orders (code, user_id, recipient_name, recipient_phone, city, address_text, pii_enc, payment_method, payment_status, status, items_total, payable_total, idempotency_key, created_at, updated_at) VALUES (?, ?, '', '', 'Tehran', '', ?, 'cod', 'pending', 'placed', 100, 100, ?, NOW(), NOW())`,
		"O"+u, uid, seal(`{"name":"`+tag+`-buyer","phone":"0912","address":"`+tag+`-addr"}`, "shop-order:"+u), tag+"-idem")
	x(`INSERT INTO shop_order_events (order_id, status, note, created_at) VALUES (?, 'placed', ?, NOW())`, order, tag+"-order-event")
	// directory: an application with its documents, the files it references, and a file of another purpose
	licence := x(`INSERT INTO files (user_id, purpose, visibility, mime, size_bytes, sha256, path, created_at, updated_at) VALUES (?, 'place_licence', 'private', 'application/pdf', 1234, REPEAT('a', 64), ?, NOW(), NOW())`, uid, "private/"+tag+"-licence.pdf")
	x(`INSERT INTO files (user_id, purpose, visibility, mime, size_bytes, sha256, path, created_at, updated_at) VALUES (?, 'lab', 'private', 'application/pdf', 99, REPEAT('b', 64), ?, NOW(), NOW())`, uid, "private/"+tag+"-lab.pdf")
	x(`INSERT INTO city_service_applications (user_id, status, step, business_name, documents, photo_file_ids, owner_confirmed, created_at, updated_at) VALUES (?, 'draft', 2, ?, ?, '[]', 1, NOW(), NOW())`,
		uid, tag+"-business", `{"licence":`+strconv.FormatUint(licence, 10)+`}`)
	listing := x(`INSERT INTO city_service_listings (category_id, city_id, name, booking_mode, sort_order, is_active, status, created_at, updated_at)
		VALUES ((SELECT id FROM city_service_categories WHERE slug = 'pool'), (SELECT MIN(id) FROM city_service_cities), ?, 'request', 1, 1, 'published', NOW(), NOW())`, tag+"-place")
	booking := x(`INSERT INTO city_service_bookings (user_id, listing_id, contact_name, contact_phone, preferred_date, status, created_at, updated_at) VALUES (?, ?, ?, '0912', '2026-09-30', 'completed', NOW(), NOW())`, uid, listing, tag+"-contact")
	x(`INSERT INTO city_service_reviews (listing_id, booking_id, user_id, rating, body, created_at, updated_at) VALUES (?, ?, ?, 5, ?, NOW(), NOW())`, listing, booking, uid, tag+"-review")
}

// User decision 2026-10-12 (D-96): GET /profile/export carries the canvas health data — every section, the sealed
// text opened for the owner, file metadata without paths — and never another user's row.
func TestAccountExport_CanvasSections(t *testing.T) {
	e := newCompanionEnv(t)
	_, err := e.db.Exec(`INSERT INTO city_service_cities (slug, name, sort_order, is_active, created_at, updated_at) VALUES ('tehran', '{"en":"Tehran"}', 1, 1, NOW(), NOW())`)
	require.NoError(t, err)
	sara := e.user("Sara", "09120009301")
	nina := e.user("Nina", "09120009302")
	saraID, ninaID := e.ids["Sara"], e.ids["Nina"]
	seedCanvasData(t, e, saraID, "saratag")
	seedCanvasData(t, e, ninaID, "ninatag")

	r := e.do(fiber.MethodGet, "/api/v1/profile/export", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	data := r.data()

	// Every section and part is there, with her rows.
	want := map[string][]string{
		"pelvic_floor":       {"program", "sessions", "bladder_diary"},
		"contraception":      {"method", "pill_pack", "reminders"},
		"menopause":          {"profile", "logs", "hot_flashes", "scores", "treatments", "treatment_intakes", "side_effects"},
		"ivf":                {"cycles", "meds", "injections", "scans", "tww", "reminders"},
		"condition_programs": {"enrolments", "pain_diary", "pmdd", "pbac"},
		"loss":               {"losses", "moods"},
		"insurance":          {"policies", "members", "coverages", "claims", "claim_events", "claim_documents", "questionnaires"},
		"shop":               {"addresses", "orders", "shipments", "order_items", "order_events", "returns", "reviews", "checklist", "supply_reminder"},
		"directory":          {"bookings", "reviews", "reports", "applications", "application_files"},
	}
	for section, parts := range want {
		obj, ok := data[section].(map[string]any)
		require.True(t, ok, section)
		for _, p := range parts {
			assert.Contains(t, obj, p, section+"."+p)
		}
	}
	sec := func(s, p string) any { return data[s].(map[string]any)[p] }
	list := func(s, p string) []any { l, _ := sec(s, p).([]any); return l }
	row := func(s, p string) map[string]any {
		l := list(s, p)
		require.Len(t, l, 1, s+"."+p)
		return l[0].(map[string]any)
	}

	assert.Equal(t, float64(saraID), sec("pelvic_floor", "program").(map[string]any)["user_id"])
	assert.Equal(t, []any{"burning"}, row("pelvic_floor", "bladder_diary")["uti_symptoms"])
	assert.Equal(t, true, sec("contraception", "method").(map[string]any)["followup_done"])
	assert.Equal(t, "peri", sec("menopause", "profile").(map[string]any)["menopause_stage"])
	assert.Equal(t, false, sec("menopause", "profile").(map[string]any)["menopause_surgical"])
	assert.Equal(t, true, row("menopause", "hot_flashes")["night"])
	assert.Len(t, list("menopause", "treatment_intakes"), 1)
	scan := row("ivf", "scans")
	assert.Equal(t, float64(3), scan["right_10_14"])
	assert.Equal(t, float64(1), scan["left_18_plus"])
	assert.Equal(t, "8.5", scan["endometrium_mm"])
	assert.NotContains(t, row("ivf", "cycles"), "active_user_id")
	assert.Equal(t, "belly_left", row("ivf", "injections")["site"])
	assert.Equal(t, true, row("condition_programs", "pbac")["flooding"])

	// Sealed text is opened for her; the ciphertext columns are not exported.
	assert.Equal(t, "saratag-loss-note", row("loss", "losses")["private_note"])
	claim := row("insurance", "claims")
	assert.Equal(t, "saratag-claim-note", claim["note"])
	assert.NotContains(t, claim, "notes_enc")
	assert.Equal(t, "saratag-step-note", row("insurance", "claim_events")["note"])
	q := row("insurance", "questionnaires")
	assert.Equal(t, "saratag-explanation", q["explanation"])
	assert.NotContains(t, q, "evidence_enc")
	assert.Equal(t, []any{"saratag-evidence"}, q["answers"].([]any)[0].(map[string]any)["evidence"])
	addr := row("shop", "addresses")
	assert.Equal(t, "saratag-recipient", addr["recipient_name"])
	assert.Equal(t, "saratag-street", addr["address_line"])
	assert.NotContains(t, addr, "pii_enc")
	ord := row("shop", "orders")
	assert.Equal(t, "saratag-buyer", ord["recipient_name"])
	assert.NotContains(t, ord, "idempotency_key")
	assert.Len(t, list("shop", "order_events"), 1)

	// Directory: the application with its document references, file metadata only (no path / hash / other purposes).
	app := row("directory", "applications")
	assert.Equal(t, map[string]any{"licence": float64(idOfF(row("directory", "application_files")))}, app["documents"])
	file := row("directory", "application_files")
	assert.Equal(t, "place_licence", file["purpose"])
	assert.NotContains(t, file, "path")
	assert.NotContains(t, file, "sha256")
	assert.Len(t, list("directory", "bookings"), 1)
	assert.Len(t, list("directory", "reviews"), 1)

	// Owner-only: nothing of Nina's anywhere (her tagged text, sealed or not), every user_id is Sara's.
	assert.NotContains(t, r.Raw, "ninatag")
	assert.NotContains(t, r.Raw, "licence.pdf")
	var walk func(v any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, val := range x {
				if k == "user_id" {
					assert.Equal(t, float64(saraID), val)
				}
				walk(val)
			}
		case []any:
			for _, val := range x {
				walk(val)
			}
		}
	}
	for section := range want {
		walk(data[section])
	}

	// And the other way round.
	r = e.do(fiber.MethodGet, "/api/v1/profile/export", nina, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.NotContains(t, r.Raw, "saratag")
	assert.Contains(t, r.Raw, "ninatag-claim-note")
}
