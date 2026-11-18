package accountexport

// T-M12-14 (finding T-04): every user table is in the export or on the documented exclusion list
// (docs/quality-2026-11/user-tables.md → "Account export"). The tables below had no export section; they are read
// with one fixed statement each — table, owner predicate, order and hidden columns are constants in this file (never
// request input) and the owner's id is the only bound value — and serialised from the column types: integers as
// numbers, DECIMAL as the DB string, DATE as "Y-m-d", DATETIME / TIMESTAMP as the Laravel datetime, JSON columns
// (CHECK json_valid) decoded, everything else as text. Text a domain seals at rest is opened for the owner (the AD
// binds each seal to her, so another account's blob never opens; one that does not open exports as null). Link
// tables (family, companions, guardians) export only the link rows she is a party to, never the other side's data.
// Credentials, token / code hashes, storage paths and internal keys are hidden.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/profile/model"
)

// spec is one part read from one table.
type spec struct {
	key    string // the part's key inside the section
	table  string
	where  string // owner predicate; every `?` binds the owner's id
	order  string // ORDER BY (default "id")
	hidden []string
	each   func(uid uint64, o *jsonx.OrderedMap) // opens sealed text, adjusts the row
}

// group is one section of generic parts.
type group struct {
	key   string
	parts []spec
}

const (
	ownChildren  = "child_id IN (SELECT id FROM `children` WHERE owner_id = ?)"
	ownCompanion = "companion_id IN (SELECT id FROM `companions` WHERE owner_id = ? OR companion_user_id = ?)"
	ownBooking   = "booking_id IN (SELECT id FROM `telemed_bookings` WHERE user_id = ?)"
	byUser       = "user_id = ?"
)

// tableGroups are the sections (and parts of existing sections) read through the generic loader, in output order.
func tableGroups(box Box) []group {
	return []group{
		{key: "cycle", parts: []spec{
			{key: "preferences", table: "cycle_preferences", where: byUser},
			{key: "skips", table: "cycle_skips", where: byUser},
			{key: "predictions", table: "engine_prediction_log", where: byUser},
		}},
		{key: "fertility", parts: []spec{
			{key: "logs", table: "fertility_logs", where: byUser, order: "log_date, id"},
		}},
		{key: "health_log", parts: []spec{
			{key: "custom_items", table: "health_log_custom_items", where: byUser},
			{key: "preferences", table: "health_log_preferences", where: byUser},
		}},
		{key: "pregnancy_tools", parts: []spec{
			{key: "alerts", table: "pregnancy_alerts", where: byUser},
			{key: "contraction_sessions", table: "pregnancy_contraction_sessions", where: byUser},
			{key: "contractions", table: "pregnancy_contractions", where: byUser},
			{key: "daily_extras", table: "pregnancy_daily_extras", where: byUser, order: "log_date, id"},
			{key: "kick_sessions", table: "pregnancy_kick_sessions", where: byUser},
			{key: "week_state", table: "pregnancy_week_user_state", where: byUser, order: "week, id"},
		}},
		{key: "health_record", parts: []spec{
			{key: "record", table: "health_records", where: byUser},
			{key: "pregnancies", table: "health_record_pregnancies", where: byUser},
			{key: "documents", table: "record_documents", where: byUser},
			{key: "document_files", table: "record_document_files", where: byUser},
			{key: "document_links", table: "record_document_links", where: byUser},
			{key: "emergency_card", table: "emergency_cards", where: byUser, hidden: []string{"public_token_hash"}},
			{key: "share_links", table: "health_share_links", where: byUser, hidden: []string{"token_hash", "code_hash", "code_payload"}},
			{key: "share_link_views", table: "health_share_link_views", where: byUser},
		}},
		{key: "labs", parts: []spec{
			{key: "reports", table: "lab_reports", where: byUser},
			{key: "markers", table: "lab_markers", where: byUser, order: "lab_id, sort_order, id"},
			{key: "files", table: "lab_files", where: byUser, hidden: []string{"path"}},
			{key: "partner_requests", table: "partner_lab_requests", where: byUser, each: partnerPayload(box)},
		}},
		{key: "checkups", parts: []spec{
			{key: "types", table: "checkup_types", where: byUser}, // her own checkup types (catalog rows have no user)
			{key: "records", table: "checkup_records", where: byUser},
			{key: "settings", table: "user_checkup_settings", where: byUser},
		}},
		{key: "vitals", parts: []spec{
			{key: "plan_items", table: "vital_plan_items", where: byUser},
			{key: "readings", table: "vital_readings", where: byUser, order: "measured_at, id"},
			{key: "targets", table: "vital_targets", where: byUser},
		}},
		{key: "care", parts: []spec{
			{key: "reminder_intakes", table: "reminder_intakes", where: byUser, order: "intake_date, id"},
		}},
		{key: "children", parts: []spec{
			{key: "children", table: "children", where: "owner_id = ?"},
			{key: "guardians", table: "child_guardians", where: "user_id = ? OR " + ownChildren},
			{key: "measurements", table: "child_measurements", where: ownChildren},
			{key: "milestones", table: "child_milestone_checks", where: ownChildren},
			{key: "vaccines", table: "child_vaccine_doses", where: ownChildren},
			{key: "feeds", table: "baby_feeds", where: ownChildren},
			{key: "sleeps", table: "baby_sleeps", where: ownChildren},
			{key: "diapers", table: "baby_diapers", where: ownChildren},
			{key: "med_doses", table: "baby_med_doses", where: ownChildren},
		}},
		{key: "family", parts: []spec{
			{key: "families", table: "families", where: "owner_id = ? OR spouse_user_id = ?"},
			{key: "family_children", table: "family_children", where: ownChildren},
			{key: "companions", table: "companions", where: "owner_id = ? OR companion_user_id = ?"},
			{key: "companion_grants", table: "companion_grants", where: ownCompanion},
			{key: "invites", table: "companion_invites", where: "owner_id = ?", hidden: []string{"code_hash"}},
			{key: "activity", table: "companion_audit_logs", where: "owner_id = ?", hidden: []string{"payload_sealed"}},
			{key: "attendance", table: "family_attendance", where: "member_id = ?"},
			// Alerts and notices about her own data (her side as the owner); a member's copy is the owner's data.
			{key: "member_alerts", table: "family_member_alerts", where: "owner_id = ?"},
			{key: "member_notices", table: "family_member_notices", where: "owner_id = ?"},
			{key: "nudges", table: "family_nudges", where: "owner_id = ? OR member_id = ?"},
			{key: "owner_state", table: "family_owner_state", where: "owner_id = ?", order: "owner_id"},
			{key: "proposals", table: "family_proposals", where: "actor_id = ? OR approver_id = ?",
				hidden: []string{"payload_sealed", "dedupe_key", "pending_key"}},
			{key: "share_requests", table: "family_share_requests", where: "from_user_id = ? OR to_user_id = ?"},
		}},
		{key: "todo", parts: []spec{
			{key: "tasks", table: "todo_tasks", where: byUser},
			{key: "items", table: "todo_items", where: byUser},
			{key: "suggestion_events", table: "todo_suggestion_events", where: byUser},
		}},
		{key: "teen", parts: []spec{
			{key: "profile", table: "teen_profiles", where: byUser},
			{key: "kit_checks", table: "teen_kit_checks", where: byUser},
		}},
		{key: "challenges", parts: []spec{
			{key: "enrollments", table: "challenge_enrollments", where: byUser},
			{key: "task_completions", table: "challenge_task_completions",
				where: "enrollment_id IN (SELECT id FROM `challenge_enrollments` WHERE user_id = ?)"},
			{key: "reward_grants", table: "challenge_reward_grants", where: byUser},
			{key: "daily_completions", table: "user_challenge_completions", where: byUser},
			{key: "task_template_completions", table: "user_task_completions", where: byUser},
		}},
		{key: "assistant", parts: []spec{
			{key: "conversations", table: "assistant_conversations", where: byUser, each: assistantSummary(box)},
			{key: "messages", table: "assistant_messages", order: "conversation_id, id",
				where: "conversation_id IN (SELECT id FROM `assistant_conversations` WHERE user_id = ?)", each: assistantBody(box)},
			{key: "handoffs", table: "assistant_handoffs", where: byUser, each: assistantHandoff(box)},
			{key: "settings", table: "assistant_profile_settings", where: byUser},
		}},
		{key: "telemed", parts: []spec{
			{key: "bookings", table: "telemed_bookings", where: byUser, hidden: []string{"authority"}},
			{key: "consents", table: "telemed_booking_consents", where: ownBooking},
			{key: "messages", table: "telemed_messages", where: ownBooking, order: "booking_id, id", each: telemedBody(box)},
			{key: "prescription_items", table: "telemed_prescription_items", where: ownBooking},
			{key: "reviews", table: "telemed_reviews", where: byUser},
		}},
		{key: "learning", parts: []spec{
			{key: "instructor", table: "learning_instructors", where: byUser},
			{key: "grants", table: "learning_grants", where: byUser},
			{key: "progress", table: "learning_progress", where: byUser},
		}},
		{key: "plus", parts: []spec{
			{key: "subscriptions", table: "plus_subscriptions", where: byUser},
			{key: "trials", table: "plus_trials", where: byUser},
			{key: "invoices", table: "plus_invoices", where: byUser, hidden: []string{"authority"}},
			{key: "receipts", table: "plus_receipts", where: byUser},
			{key: "store_purchases", table: "plus_store_purchases", where: byUser, hidden: []string{"purchase_token"}},
			{key: "usage_counters", table: "plus_usage_counters", where: byUser},
		}},
		{key: "ai_credits", parts: []spec{
			{key: "grants", table: "ai_credit_grants", where: byUser},
			{key: "spends", table: "ai_credit_spends", where: byUser},
		}},
		{key: "notifications", parts: []spec{
			{key: "inbox", table: "user_notifications", where: byUser},
			{key: "smart_messages", table: "message_inbox", where: byUser},
			{key: "message_events", table: "message_events", where: byUser},
		}},
		{key: "activity", parts: []spec{
			{key: "events", table: "analytics_events", where: byUser},
			{key: "days", table: "user_activity_days", where: byUser, order: "day"},
		}},
		{key: "account_records", parts: []spec{
			{key: "consent_events", table: "consent_events", where: byUser},
			{key: "profile_backups", table: "user_profiles_dedupe_backup", where: byUser},
			// Metadata of every stored file — never the bytes, the storage path or the hash.
			{key: "files", table: "files", where: byUser, hidden: []string{"path", "sha256"}},
		}},
	}
}

// shopExtra / directoryExtra are generic parts appended to the existing shop and directory sections.
var (
	shopExtra = []spec{
		{key: "carts", table: "shop_carts", where: byUser},
		{key: "cart_items", table: "shop_cart_items", where: "cart_id IN (SELECT id FROM `shop_carts` WHERE user_id = ?)"},
	}
	directoryExtra = []spec{
		// A venue listing she claimed as its owner, with its services and slots.
		{key: "owned_listings", table: "city_service_listings", where: "owner_user_id = ?"},
		{key: "owned_listing_services", table: "city_service_services",
			where: "listing_id IN (SELECT id FROM `city_service_listings` WHERE owner_user_id = ?)"},
		{key: "owned_listing_slots", table: "city_service_slots",
			where: "listing_id IN (SELECT id FROM `city_service_listings` WHERE owner_user_id = ?)"},
	}
)

// handWritten are the tables the typed sections read (accountexport.go / health.go / commerce.go).
var handWritten = map[string]string{
	"pelvic_programs": "pelvic_floor.program", "pelvic_sessions": "pelvic_floor.sessions", "pelvic_bladder_logs": "pelvic_floor.bladder_diary",
	"contraception_methods": "contraception.method", "contraception_pill_logs": "contraception.pill_pack",
	"contraception_reminders": "contraception.reminders",
	"health_log_entries":      "menopause.logs", "hot_flashes": "menopause.hot_flashes", "menopause_scores": "menopause.scores",
	"treatment_items": "menopause.treatments", "treatment_intakes": "menopause.treatment_intakes", "side_effect_logs": "menopause.side_effects",
	"ivf_cycles": "ivf.cycles", "ivf_meds": "ivf.meds", "ivf_dose_logs": "ivf.injections", "ivf_scans": "ivf.scans",
	"ivf_tww_logs": "ivf.tww", "ivf_reminders": "ivf.reminders",
	"condition_enrolments": "condition_programs.enrolments", "condition_pain_entries": "condition_programs.pain_diary",
	"pmdd_entries": "condition_programs.pmdd", "pbac_entries": "condition_programs.pbac",
	"pregnancy_losses": "loss.losses", "pregnancy_loss_moods": "loss.moods",
	"insurance_policies": "insurance.policies", "insurance_policy_members": "insurance.members",
	"insurance_coverages": "insurance.coverages", "insurance_claims": "insurance.claims",
	"insurance_claim_events": "insurance.claim_events", "insurance_claim_documents": "insurance.claim_documents",
	"insurance_questionnaires": "insurance.questionnaires",
	"shop_addresses":           "shop.addresses", "shop_orders": "shop.orders", "shop_order_shipments": "shop.shipments",
	"shop_order_items": "shop.order_items", "shop_order_events": "shop.order_events", "shop_returns": "shop.returns",
	"shop_product_reviews": "shop.reviews", "shop_user_checklist_items": "shop.checklist", "shop_supply_reminders": "shop.supply_reminder",
	"city_service_bookings": "directory.bookings", "city_service_reviews": "directory.reviews", "city_service_reports": "directory.reports",
	"city_service_applications": "directory.applications",
	"postpartum_profiles":       "postpartum.profile", "postpartum_recovery_logs": "postpartum.recovery_logs",
}

// Coverage maps every table this package exports to its "section.part" in GET /profile/export (the T-M12-14 test
// checks it against the schema and the exclusion list; the profile package's own keys are listed there).
func Coverage() map[string]string {
	out := make(map[string]string, len(handWritten)+64)
	for t, p := range handWritten {
		out[t] = p
	}
	for _, g := range tableGroups(nil) {
		for _, s := range g.parts {
			out[s.table] = g.key + "." + s.key
		}
	}
	for _, s := range shopExtra {
		out[s.table] = "shop." + s.key
	}
	for _, s := range directoryExtra {
		out[s.table] = "directory." + s.key
	}
	return out
}

// tables reads generic parts; the JSON columns are learnt once from the schema's json_valid CHECKs.
type tables struct {
	db       *sql.DB
	once     sync.Once
	jsonCols map[string]bool // "table.column"
	jsonErr  error
}

var jsonValid = regexp.MustCompile("^json_valid\\(`([^`]+)`\\)$")

func (x *tables) loadJSONCols(ctx context.Context) error {
	x.once.Do(func() {
		x.jsonCols = map[string]bool{}
		rows, err := x.db.QueryContext(ctx,
			`SELECT table_name, check_clause FROM information_schema.check_constraints WHERE constraint_schema = DATABASE()`)
		if err != nil {
			x.jsonErr = fmt.Errorf("accountexport: json columns: %w", err)
			return
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var tbl, clause string
			if err := rows.Scan(&tbl, &clause); err != nil {
				x.jsonErr = fmt.Errorf("accountexport: json columns: %w", err)
				return
			}
			if m := jsonValid.FindStringSubmatch(clause); m != nil {
				x.jsonCols[tbl+"."+m[1]] = true
			}
		}
		if err := rows.Err(); err != nil {
			x.jsonErr = fmt.Errorf("accountexport: json columns: %w", err)
		}
	})
	return x.jsonErr
}

// part is a spec as a section part for uid.
func (x *tables) part(ctx context.Context, uid uint64, s spec) part {
	return part{key: s.key, load: func() ([]any, error) { return x.rows(ctx, uid, s) }}
}

func (x *tables) rows(ctx context.Context, uid uint64, s spec) ([]any, error) {
	if err := x.loadJSONCols(ctx); err != nil {
		return nil, err
	}
	order := s.order
	if order == "" {
		order = "id"
	}
	args := make([]any, strings.Count(s.where, "?"))
	for i := range args {
		args[i] = uid
	}
	// Table, predicate and order are constants of this file; the owner's id is the only bound value.
	q := "SELECT * FROM `" + s.table + "` WHERE " + s.where + " ORDER BY " + order //nolint:gosec // G202: constants only
	rows, err := x.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("accountexport: %s: %w", s.table, err)
	}
	defer func() { _ = rows.Close() }()
	types, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("accountexport: %s: %w", s.table, err)
	}
	vals := make([]any, len(types))
	ptrs := make([]any, len(types))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	out := []any{}
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("accountexport: %s: %w", s.table, err)
		}
		o := jsonx.NewObject()
		for i, ct := range types {
			col := ct.Name()
			if contains(s.hidden, col) {
				continue
			}
			o.Set(col, x.value(s.table, col, ct.DatabaseTypeName(), vals[i]))
		}
		if s.each != nil {
			s.each(uid, o)
		}
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("accountexport: %s: %w", s.table, err)
	}
	return out, nil
}

// value serialises one cell by its column type (see the file comment).
func (x *tables) value(table, col, dbType string, v any) any {
	switch t := v.(type) {
	case nil:
		return nil
	case time.Time:
		if dbType == "DATE" {
			return t.Format(time.DateOnly)
		}
		return jsonx.DateTime(t)
	case []byte:
		if x.jsonCols[table+"."+col] {
			return model.DecodeJSONColumn(json.RawMessage(t))
		}
		return string(t)
	case string:
		if x.jsonCols[table+"."+col] {
			return model.DecodeJSONColumn(json.RawMessage(t))
		}
		return t
	}
	return v // int64, uint64, float64, bool
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// generic is a section of generic parts.
func (x *tables) generic(g group) loader {
	return func(ctx context.Context, uid uint64) (any, error) {
		parts := make([]part, 0, len(g.parts))
		for _, s := range g.parts {
			parts = append(parts, x.part(ctx, uid, s))
		}
		return object(parts...)
	}
}

// withExtra appends generic parts to a typed section's object.
func (x *tables) withExtra(base loader, extra []spec) loader {
	return func(ctx context.Context, uid uint64) (any, error) {
		v, err := base(ctx, uid)
		if err != nil {
			return nil, err
		}
		o, ok := v.(*jsonx.OrderedMap)
		if !ok {
			return v, nil
		}
		parts := make([]part, 0, len(extra))
		for _, s := range extra {
			parts = append(parts, x.part(ctx, uid, s))
		}
		more, err := object(parts...)
		if err != nil {
			return nil, err
		}
		m := more.(*jsonx.OrderedMap) //nolint:forcetypeassert // object always returns *OrderedMap
		for _, s := range extra {
			val, _ := m.Get(s.key)
			o.Set(s.key, val)
		}
		return o, nil
	}
}

// ---------------------------------------------------------------------------------------------------------------
// Sealed text, opened for the owner (the AD formats are the owning domains').

func u64(o *jsonx.OrderedMap, col string) uint64 {
	v, _ := o.Get(col)
	switch n := v.(type) {
	case int64:
		if n > 0 {
			return uint64(n)
		}
	case uint64:
		return n
	}
	return 0
}

func str(o *jsonx.OrderedMap, col string) string {
	v, _ := o.Get(col)
	s, _ := v.(string)
	return s
}

// sealedPrefix starts every NoteBox value ("v1:<kid>:<base64>").
const sealedPrefix = "v1:"

// openOrPlain is internal/assistant's read rule: legacy plaintext (no prefix) as is, a sealed value opened, one that
// does not open null — never the ciphertext.
func openOrPlain(box Box, stored string, ad []byte) any {
	if stored == "" {
		return nil
	}
	if !strings.HasPrefix(stored, sealedPrefix) {
		return stored
	}
	if box == nil || box.Disabled() {
		return nil
	}
	return opened(box.OpenWith(stored, ad))
}

func assistantAD(parts ...any) []byte {
	s := make([]string, len(parts))
	for i, p := range parts {
		switch v := p.(type) {
		case string:
			s[i] = v
		case uint64:
			s[i] = strconv.FormatUint(v, 10)
		}
	}
	return []byte(strings.Join(s, ":"))
}

func assistantSummary(box Box) func(uint64, *jsonx.OrderedMap) {
	return func(uid uint64, o *jsonx.OrderedMap) {
		o.Set("summary", openOrPlain(box, str(o, "summary"), assistantAD("assistant_conversations.summary", uid, u64(o, "id"))))
	}
}

func assistantBody(box Box) func(uint64, *jsonx.OrderedMap) {
	return func(uid uint64, o *jsonx.OrderedMap) {
		o.Set("body", openOrPlain(box, str(o, "body"),
			assistantAD("assistant_messages.body", uid, u64(o, "conversation_id"), u64(o, "id"))))
	}
}

func assistantHandoff(box Box) func(uint64, *jsonx.OrderedMap) {
	return func(uid uint64, o *jsonx.OrderedMap) {
		o.Set("summary", openOrPlain(box, str(o, "summary"),
			assistantAD("assistant_handoffs.summary", uid, u64(o, "conversation_id"), u64(o, "booking_id"))))
	}
}

// telemedBody opens a chat message (internal/telemed messageAD); a body that does not open is null.
func telemedBody(box Box) func(uint64, *jsonx.OrderedMap) {
	return func(_ uint64, o *jsonx.OrderedMap) {
		var body any
		if s := str(o, "body"); s != "" && box != nil && !box.Disabled() {
			ad := "telemed-message:v1:" + strconv.FormatUint(u64(o, "booking_id"), 10) + ":" + str(o, "sender")
			body = opened(box.OpenWith(s, []byte(ad)))
		}
		o.Set("body", body)
	}
}

// partnerPayload opens a partner lab request's contact details (internal/partners payloadAD) as an object in place of
// payload_enc; null once purged or when it does not open.
func partnerPayload(box Box) func(uint64, *jsonx.OrderedMap) {
	return func(uid uint64, o *jsonx.OrderedMap) {
		enc := str(o, "payload_enc")
		o.Delete("payload_enc")
		var payload any
		if enc != "" && box != nil && !box.Disabled() {
			ad := "partner-lab:" + strconv.FormatUint(uid, 10) + ":" + str(o, "reference")
			if plain, err := box.OpenWith(enc, []byte(ad)); err == nil {
				payload = model.DecodeJSONColumn(json.RawMessage(plain))
			}
		}
		o.Set("payload", payload)
	}
}
