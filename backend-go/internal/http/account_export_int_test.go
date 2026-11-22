package http

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/accountexport"
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

	r := e.doStep(fiber.MethodGet, "/api/v1/profile/export", sara, e.mintStepUp(sara), nil)
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
	r = e.doStep(fiber.MethodGet, "/api/v1/profile/export", nina, e.mintStepUp(nina), nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.NotContains(t, r.Raw, "saratag")
	assert.Contains(t, r.Raw, "ninatag-claim-note")
}

// ---------------------------------------------------------------------------------------------------------------
// T-M12-14 (finding T-04): the export covers every user table — proved from the live schema. The user-table set,
// the schema reader and the row seeder are copied from internal/profile/delete_complete_int_test.go (T-M12-13; a
// _test package there, so not importable, and internal/profile is outside this task's paths). The exclusion list and
// the profile-package keys are parsed from docs/quality-2026-11/user-tables.md → "Account export".

const axUserTablesDoc = "../../../docs/quality-2026-11/user-tables.md"

type axCol struct {
	name, dataType, colType string
	nullable, hasDefault    bool
	autoInc, generated      bool
	maxLen                  int64
}

type axFK struct{ refTable, refCol string }

type axSchema struct {
	tables []string
	cols   map[string][]axCol
	fks    map[string]map[string]axFK
	pk     map[string][]string
	json   map[string]bool
}

func axEachRow(t *testing.T, db *sql.DB, q string, scan func(*sql.Rows)) {
	t.Helper()
	rows, err := db.Query(q)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		scan(rows)
	}
	require.NoError(t, rows.Err())
}

func axLoadSchema(t *testing.T, db *sql.DB) *axSchema {
	t.Helper()
	s := &axSchema{cols: map[string][]axCol{}, fks: map[string]map[string]axFK{}, pk: map[string][]string{}, json: map[string]bool{}}
	axEachRow(t, db, `SELECT c.table_name, c.column_name, c.data_type, c.column_type, c.is_nullable = 'YES',
			c.column_default IS NOT NULL, c.extra, COALESCE(c.character_maximum_length, 0)
		FROM information_schema.columns c
		JOIN information_schema.tables t ON t.table_schema = c.table_schema AND t.table_name = c.table_name AND t.table_type = 'BASE TABLE'
		WHERE c.table_schema = DATABASE() ORDER BY c.table_name, c.ordinal_position`, func(r *sql.Rows) {
		var tbl, extra string
		var c axCol
		require.NoError(t, r.Scan(&tbl, &c.name, &c.dataType, &c.colType, &c.nullable, &c.hasDefault, &extra, &c.maxLen))
		c.autoInc = strings.Contains(extra, "auto_increment")
		c.generated = strings.Contains(extra, "GENERATED")
		if _, ok := s.cols[tbl]; !ok {
			s.tables = append(s.tables, tbl)
		}
		s.cols[tbl] = append(s.cols[tbl], c)
	})
	axEachRow(t, db, `SELECT k.table_name, k.column_name, k.referenced_table_name, k.referenced_column_name
		FROM information_schema.key_column_usage k
		WHERE k.table_schema = DATABASE() AND k.referenced_table_name IS NOT NULL`, func(r *sql.Rows) {
		var tbl, col string
		var fk axFK
		require.NoError(t, r.Scan(&tbl, &col, &fk.refTable, &fk.refCol))
		if s.fks[tbl] == nil {
			s.fks[tbl] = map[string]axFK{}
		}
		s.fks[tbl][col] = fk
	})
	axEachRow(t, db, `SELECT table_name, column_name FROM information_schema.key_column_usage
		WHERE table_schema = DATABASE() AND constraint_name = 'PRIMARY' ORDER BY table_name, ordinal_position`, func(r *sql.Rows) {
		var tbl, col string
		require.NoError(t, r.Scan(&tbl, &col))
		s.pk[tbl] = append(s.pk[tbl], col)
	})
	jsonRe := regexp.MustCompile("^json_valid\\(`([^`]+)`\\)$")
	axEachRow(t, db, `SELECT table_name, check_clause FROM information_schema.check_constraints WHERE constraint_schema = DATABASE()`, func(r *sql.Rows) {
		var tbl, clause string
		require.NoError(t, r.Scan(&tbl, &clause))
		if m := jsonRe.FindStringSubmatch(clause); m != nil {
			s.json[tbl+"."+m[1]] = true
		}
	})
	return s
}

func axUserColumnName(col string) bool {
	switch col {
	case "user_id", "owner_id", "member_id", "child_owner_id":
		return true
	}
	return strings.HasSuffix(col, "_user_id")
}

// userRef: the column holds a user id — by a foreign key to users, or by name when it has no foreign key.
func (s *axSchema) userRef(tbl, col string) bool {
	if fk, ok := s.fks[tbl][col]; ok {
		return fk.refTable == "users"
	}
	return axUserColumnName(col)
}

// userTables is the T-M12-13 set: a user column, or a foreign key to a user table (transitively).
func (s *axSchema) userTables() map[string]bool {
	scope := map[string]bool{}
	for _, tbl := range s.tables {
		if tbl == "users" {
			continue
		}
		for _, c := range s.cols[tbl] {
			if s.userRef(tbl, c.name) {
				scope[tbl] = true
			}
		}
	}
	for changed := true; changed; {
		changed = false
		for _, tbl := range s.tables {
			if scope[tbl] || tbl == "users" {
				continue
			}
			for _, fk := range s.fks[tbl] {
				if scope[fk.refTable] {
					scope[tbl], changed = true, true
					break
				}
			}
		}
	}
	return scope
}

type axRow struct {
	table string
	vals  map[string]any
}

type axSeeder struct {
	t      *testing.T
	db     *sql.DB
	s      *axSchema
	scope  map[string]bool
	n      int
	memo   map[string]*axRow
	active map[string]bool
	rows   []*axRow
}

func (sd *axSeeder) value(tbl string, c axCol, attempt int) any {
	sd.n++
	switch c.dataType {
	case "tinyint", "smallint", "mediumint", "int", "bigint", "decimal", "float", "double":
		return 1 + attempt
	case "date":
		return "2026-09-20"
	case "datetime", "timestamp":
		return "2026-09-20 10:00:00"
	case "time":
		return "10:00:00"
	case "year":
		return 2026
	case "enum", "set":
		inner := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(c.colType, "enum("), "set("), ")")
		return strings.Trim(strings.SplitN(inner, ",", 2)[0], "'")
	}
	if sd.s.json[tbl+"."+c.name] {
		return "{}"
	}
	v := fmt.Sprintf("%dt14%d", sd.n, attempt)
	if c.maxLen > 0 && int64(len(v)) > c.maxLen {
		v = v[:c.maxLen]
	}
	return v
}

// row seeds one row of tbl; users(col) is the id of each user column.
func (sd *axSeeder) row(tbl string, owner uint64, users func(col string) any) *axRow {
	t := sd.t
	t.Helper()
	r := &axRow{table: tbl}
	for attempt := 0; attempt < 8; attempt++ {
		r.vals = map[string]any{}
		var cols []string
		var args []any
		for _, c := range sd.s.cols[tbl] {
			if c.autoInc || c.generated {
				continue
			}
			var v any
			fk, hasFK := sd.s.fks[tbl][c.name]
			switch {
			case sd.s.userRef(tbl, c.name):
				v = users(c.name)
			case hasFK && fk.refTable == tbl:
				require.True(t, c.nullable, "NOT NULL self reference %s.%s", tbl, c.name)
			case hasFK && (!c.nullable || sd.scope[fk.refTable]):
				pOwner := owner
				if !sd.scope[fk.refTable] {
					pOwner = 0
				}
				p := sd.ensure(fk.refTable, pOwner)
				if p == nil {
					require.True(t, c.nullable, "FK cycle at %s.%s", tbl, c.name)
					continue
				}
				v = p.vals[fk.refCol]
			case c.nullable || c.hasDefault:
				continue
			default:
				v = sd.value(tbl, c, attempt)
			}
			if v == nil {
				continue
			}
			cols = append(cols, "`"+c.name+"`")
			args = append(args, v)
			r.vals[c.name] = v
		}
		q := "INSERT INTO `" + tbl + "` (" + strings.Join(cols, ", ") + ") VALUES (" +
			strings.TrimSuffix(strings.Repeat("?, ", len(cols)), ", ") + ")"
		if len(cols) == 0 {
			q = "INSERT INTO `" + tbl + "` () VALUES ()"
		}
		res, err := sd.db.Exec(q, args...)
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1062 {
			continue
		}
		require.NoError(t, err, "seed %s", tbl)
		for _, c := range sd.s.cols[tbl] {
			if c.autoInc {
				id, err := res.LastInsertId()
				require.NoError(t, err)
				r.vals[c.name] = id
			}
		}
		sd.rows = append(sd.rows, r)
		return r
	}
	t.Fatalf("seed %s: unique key collisions", tbl)
	return nil
}

func (sd *axSeeder) ensure(tbl string, owner uint64) *axRow {
	key := fmt.Sprintf("%d/%s", owner, tbl)
	if r, ok := sd.memo[key]; ok {
		return r
	}
	if sd.active[key] {
		return nil
	}
	sd.active[key] = true
	defer delete(sd.active, key)
	r := sd.row(tbl, owner, func(string) any { return owner })
	sd.memo[key] = r
	return r
}

// axExportDoc is the "Account export" part of user-tables.md: tables the profile package exports (→ key) and the
// exclusion list (→ reason).
func axExportDoc(t *testing.T) (profileKeys, excluded map[string]string) {
	t.Helper()
	f, err := os.Open(axUserTablesDoc)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	row := regexp.MustCompile("^\\| `([a-z0-9_]+)` \\| ([^|]+) \\|$")
	profileKeys, excluded = map[string]string{}, map[string]string{}
	var cur map[string]string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "### Exported by `internal/profile`"):
			cur = profileKeys
		case line == "### Not exported":
			cur = excluded
		case strings.HasPrefix(line, "#"):
			cur = nil
		}
		if m := row.FindStringSubmatch(line); cur != nil && m != nil {
			cur[m[1]] = strings.Trim(strings.TrimSpace(m[2]), "`")
		}
	}
	require.NoError(t, sc.Err())
	require.NotEmpty(t, profileKeys, "profile-package table not found in %s", axUserTablesDoc)
	require.NotEmpty(t, excluded, "exclusion list not found in %s", axUserTablesDoc)
	return profileKeys, excluded
}

// exportCoverageViolations: every user table is exported or excluded (with a reason), never both, and no list names
// a table that is not a user table.
func exportCoverageViolations(scope map[string]bool, exported, excluded map[string]string) []string {
	var out []string
	for tbl := range scope {
		_, ex := exported[tbl]
		_, no := excluded[tbl]
		switch {
		case ex && no:
			out = append(out, tbl+": exported and on the exclusion list")
		case !ex && !no:
			out = append(out, tbl+": not in the export — add a section (accountexport) or an exclusion row with a reason")
		}
	}
	for tbl := range exported {
		if !scope[tbl] {
			out = append(out, tbl+": exported but not a user table — stale coverage entry")
		}
	}
	for tbl, reason := range excluded {
		if !scope[tbl] {
			out = append(out, tbl+": excluded but not a user table — stale exclusion row")
		}
		if reason == "" {
			out = append(out, tbl+": excluded without a reason")
		}
	}
	sort.Strings(out)
	return out
}

func axExported(t *testing.T) map[string]string {
	t.Helper()
	profileKeys, _ := axExportDoc(t)
	out := accountexport.Coverage()
	for tbl, key := range profileKeys {
		_, dup := out[tbl]
		require.False(t, dup, "%s: listed for both internal/profile and accountexport", tbl)
		out[tbl] = key
	}
	return out
}

// axAt returns the objects at a dotted export path (a list's elements, or the single object).
func axAt(data map[string]any, path string) (objs []map[string]any, present bool) {
	var node any = data
	for _, k := range strings.Split(path, ".") {
		m, ok := node.(map[string]any)
		if !ok {
			return nil, false
		}
		if node, ok = m[k]; !ok {
			return nil, false
		}
	}
	switch v := node.(type) {
	case map[string]any:
		return []map[string]any{v}, true
	case []any:
		for _, e := range v {
			if o, ok := e.(map[string]any); ok {
				objs = append(objs, o)
			}
		}
	}
	return objs, true
}

func axSame(got, want any) bool {
	if f, ok := got.(float64); ok {
		return strconv.FormatFloat(f, 'f', -1, 64) == fmt.Sprint(want)
	}
	return fmt.Sprint(got) == fmt.Sprint(want)
}

// axHas: some object at path is the seeded row (every primary-key column equal).
func axHas(objs []map[string]any, pk []string, r *axRow) bool {
	for _, o := range objs {
		match := true
		for _, c := range pk {
			if v, ok := o[c]; !ok || !axSame(v, r.vals[c]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// Profile keys whose rows carry no primary key in the export (consent codes, AI usage counts, the effective
// notification settings object, the one-per-user life profile): presence is checked instead.
var axNoPKKeys = map[string]bool{"consents": true, "ai_usage": true, "notification_settings": true, "life_profile": true}

func TestAccountExport_CoversEveryUserTable(t *testing.T) {
	e := newCompanionEnv(t)
	s := axLoadSchema(t, e.db)
	scope := s.userTables()
	require.Greater(t, len(scope), 100, "the user-table set comes from the schema")
	_, excluded := axExportDoc(t)
	exported := axExported(t)

	// 1. Schema: every user table is exported or excluded, nothing stale.
	assert.Empty(t, exportCoverageViolations(scope, exported, excluded))
	t.Run("checker", func(t *testing.T) {
		probe := map[string]bool{"t14_a": true, "t14_b": true, "t14_c": true}
		got := exportCoverageViolations(probe,
			map[string]string{"t14_a": "x.a", "t14_c": "x.c", "t14_gone": "x.g"},
			map[string]string{"t14_c": "derived", "t14_old": "queue", "t14_a2": ""})
		assert.Equal(t, []string{
			"t14_a2: excluded but not a user table — stale exclusion row",
			"t14_a2: excluded without a reason",
			"t14_b: not in the export — add a section (accountexport) or an exclusion row with a reason",
			"t14_c: exported and on the exclusion list",
			"t14_gone: exported but not a user table — stale coverage entry",
			"t14_old: excluded but not a user table — stale exclusion row",
		}, got)
	})

	// 2. Behaviour: one row per user table for A and the control user B, link rows joining A to third users.
	_, err := e.db.Exec(`INSERT INTO city_service_cities (slug, name, sort_order, is_active, created_at, updated_at) VALUES ('tehran', '{"en":"Tehran"}', 1, 1, NOW(), NOW())`)
	require.NoError(t, err)
	tokA := e.user("A", "09120001401")
	e.user("B", "09120001402")
	a, b := e.ids["A"], e.ids["B"]
	sd := &axSeeder{t: t, db: e.db, s: s, scope: scope, memo: map[string]*axRow{}, active: map[string]bool{}}
	var tables []string
	for tbl := range scope {
		tables = append(tables, tbl)
	}
	sort.Strings(tables)
	for _, tbl := range tables {
		sd.ensure(tbl, a)
		sd.ensure(tbl, b)
	}
	others := 0
	for _, tbl := range tables {
		var userFKs []string
		for _, c := range s.cols[tbl] {
			if fk, ok := s.fks[tbl][c.name]; ok && fk.refTable == "users" {
				userFKs = append(userFKs, c.name)
			}
		}
		if len(userFKs) < 2 {
			continue
		}
		for _, col := range userFKs {
			others++
			e.user(fmt.Sprintf("O%d", others), fmt.Sprintf("0912200%04d", others))
			other := e.ids[fmt.Sprintf("O%d", others)]
			sd.row(tbl, other, func(c string) any {
				if c == col {
					return a
				}
				return other
			})
		}
	}
	t.Logf("seeded %d rows in %d user tables (%d exported, %d excluded)", len(sd.rows), len(scope), len(exported), len(excluded))

	r := e.doStep(fiber.MethodGet, "/api/v1/profile/export", tokA, e.mintStepUp(tokA), nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	data := r.data()

	// Each exported table: A's row is at its path, B's is not.
	for _, tbl := range tables {
		path, ok := exported[tbl]
		if !ok {
			continue
		}
		objs, present := axAt(data, path)
		if !assert.True(t, present, "%s: export key %s missing", tbl, path) {
			continue
		}
		if axNoPKKeys[path] {
			assert.NotEmpty(t, objs, "%s: %s is empty", tbl, path)
			continue
		}
		pk := s.pk[tbl]
		require.NotEmpty(t, pk, "table %s has no primary key", tbl)
		assert.True(t, axHas(objs, pk, sd.memo[fmt.Sprintf("%d/%s", a, tbl)]), "%s: A's row not at %s", tbl, path)
		assert.False(t, axHas(objs, pk, sd.memo[fmt.Sprintf("%d/%s", b, tbl)]), "%s: B's row leaked into A's %s", tbl, path)
	}

	// Nothing of B anywhere: no user column holds B's id, none of B's seeded text appears.
	userCols := map[string]bool{}
	for _, tbl := range s.tables {
		for _, c := range s.cols[tbl] {
			if s.userRef(tbl, c.name) {
				userCols[c.name] = true
			}
		}
	}
	var walk func(path string, v any)
	walk = func(path string, v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, val := range x {
				if userCols[k] && axSame(val, b) {
					t.Errorf("%s.%s holds the control user's id", path, k)
				}
				walk(path+"."+k, val)
			}
		case []any:
			for _, val := range x {
				walk(path, val)
			}
		}
	}
	walk("data", data)
	for _, row := range sd.rows {
		if sd.memo[fmt.Sprintf("%d/%s", b, row.table)] != row {
			continue
		}
		for col, v := range row.vals {
			if sv, ok := v.(string); ok && strings.Contains(sv, "t14") {
				assert.NotContains(t, r.Raw, `"`+sv+`"`, "B's %s.%s in A's export", row.table, col)
			}
		}
	}
}

// T-M12-14: sealed text of the newly exported tables (assistant summaries and messages, telemed chat, partner-lab
// contact details) is opened for the owner with the domains' AD; a seal bound to another account (or a value that
// does not open) exports as null, never as ciphertext.
func TestAccountExport_NewSectionsOpenSealedText(t *testing.T) {
	e := newCompanionEnv(t)
	box, err := loss.NewNoteBox(nil, nil, false)
	require.NoError(t, err)
	seal := func(plain, ad string) string {
		s, err := box.SealWith(plain, []byte(ad))
		require.NoError(t, err)
		return s
	}
	x := e.insExec
	sara := e.user("Sara", "09120001411")
	e.user("Nina", "09120001412")
	saraID, ninaID := e.ids["Sara"], e.ids["Nina"]
	doctor := x(`INSERT INTO telemed_doctors (kind, name, specialty, licence_no, created_at, updated_at) VALUES ('doctor', '{"en":"Dr"}', 'gyn', 'L1', NOW(), NOW())`)
	service := x(`INSERT INTO partner_lab_services (code, kind, title, partner_name, created_at, updated_at) VALUES ('cbc', 'lab', '{"en":"CBC"}', 'Lab', NOW(), NOW())`)
	ids := func(n ...uint64) []string {
		out := make([]string, len(n))
		for i, v := range n {
			out[i] = strconv.FormatUint(v, 10)
		}
		return out
	}
	seed := func(uid uint64, tag string, foreign uint64) {
		u := strconv.FormatUint(uid, 10)
		conv := x(`INSERT INTO assistant_conversations (user_id, department_id, created_at, updated_at) VALUES (?, (SELECT MIN(id) FROM assistant_departments), NOW(), NOW())`, uid)
		c := strconv.FormatUint(conv, 10)
		_, err := e.db.Exec(`UPDATE assistant_conversations SET summary = ? WHERE id = ?`, seal(tag+"-summary", "assistant_conversations.summary:"+u+":"+c), conv)
		require.NoError(t, err)
		msg := x(`INSERT INTO assistant_messages (conversation_id, role, body, created_at, updated_at) VALUES (?, 'user', '', NOW(), NOW())`, conv)
		_, err = e.db.Exec(`UPDATE assistant_messages SET body = ? WHERE id = ?`, seal(tag+"-message", "assistant_messages.body:"+u+":"+c+":"+ids(msg)[0]), msg)
		require.NoError(t, err)
		// A seal bound to another account: does not open for her.
		x(`INSERT INTO assistant_messages (conversation_id, role, body, created_at, updated_at) VALUES (?, 'assistant', ?, NOW(), NOW())`,
			conv, seal(tag+"-foreign", "assistant_messages.body:"+ids(foreign)[0]+":"+c+":0"))
		booking := x(`INSERT INTO telemed_bookings (reference, user_id, doctor_id, mode, duration_minutes, starts_at, ends_at, status, for_whom, price_rials, total_rials, authority, created_at, updated_at)
			VALUES (?, ?, ?, 'chat', 15, '2026-09-20 10:00:00', '2026-09-20 10:15:00', 'completed', 'self', 1000, 1000, ?, NOW(), NOW())`, "R"+tag, uid, doctor, tag+"-authority")
		x(`INSERT INTO telemed_messages (booking_id, sender, body, created_at, updated_at) VALUES (?, 'user', ?, NOW(), NOW())`,
			booking, seal(tag+"-chat", "telemed-message:v1:"+ids(booking)[0]+":user"))
		x(`INSERT INTO assistant_handoffs (conversation_id, user_id, booking_id, summary, created_at, updated_at) VALUES (?, ?, ?, ?, NOW(), NOW())`,
			conv, uid, booking, seal(tag+"-handoff", "assistant_handoffs.summary:"+u+":"+c+":"+ids(booking)[0]))
		x(`INSERT INTO partner_lab_requests (reference, user_id, service_id, preferred_date, preferred_window, payload_enc, created_at, updated_at) VALUES (?, ?, ?, '2026-09-25', 'morning', ?, NOW(), NOW())`,
			"P"+tag, uid, service, seal(`{"full_name":"`+tag+`-name","phone":"0912"}`, "partner-lab:"+u+":P"+tag))
	}
	seed(saraID, "saratag", ninaID)
	seed(ninaID, "ninatag", saraID)

	r := e.doStep(fiber.MethodGet, "/api/v1/profile/export", sara, e.mintStepUp(sara), nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	part := func(path string) []map[string]any {
		objs, ok := axAt(r.data(), path)
		require.True(t, ok, path)
		return objs
	}
	conv := part("assistant.conversations")
	require.Len(t, conv, 1)
	assert.Equal(t, "saratag-summary", conv[0]["summary"])
	msgs := part("assistant.messages")
	require.Len(t, msgs, 2)
	assert.Equal(t, "saratag-message", msgs[0]["body"])
	assert.Nil(t, msgs[1]["body"], "a seal bound to another account exports as null")
	assert.Equal(t, "saratag-handoff", part("assistant.handoffs")[0]["summary"])
	assert.Equal(t, "saratag-chat", part("telemed.messages")[0]["body"])
	bk := part("telemed.bookings")[0]
	assert.NotContains(t, bk, "authority")
	pr := part("labs.partner_requests")[0]
	assert.Equal(t, "saratag-name", pr["payload"].(map[string]any)["full_name"])
	assert.NotContains(t, pr, "payload_enc")
	assert.NotContains(t, r.Raw, "v1:", "no ciphertext in the export")
	assert.NotContains(t, r.Raw, "ninatag")
	assert.NotContains(t, r.Raw, "saratag-foreign")
	assert.NotContains(t, r.Raw, "saratag-authority")
}
