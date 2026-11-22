package accountexport

// T-M12-14 (finding T-04): every user table is in the export or on the documented exclusion list
// (docs/quality-2026-11/user-tables.md → "Account export"). The tables below had no export section; they are read
// with one sqlc query each (db/queries/profile/export_*.sql: explicit column list, owner's id the only bound value;
// T-M12-14b) — hidden columns are simply not selected — and serialised from the generated field types: integers as
// numbers, DECIMAL as the DB string, DATE as "Y-m-d", DATETIME / TIMESTAMP as the Laravel datetime, JSON columns
// (CHECK json_valid) decoded, everything else as text. Text a domain seals at rest is opened for the owner (the AD
// binds each seal to her, so another account's blob never opens; one that does not open exports as null). Link
// tables (family, companions, guardians) export only the link rows she is a party to, never the other side's data.
// Credentials, token / code hashes, storage paths and internal keys are hidden.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/profile/model"
)

// spec is one part read from one table.
type spec struct {
	key    string // the part's key inside the section
	table  string
	hidden []string                              // columns the export never carries: not selected by the sqlc query, checked by the column guard
	each   func(uid uint64, o *jsonx.OrderedMap) // opens sealed text, adjusts the row
}

// group is one section of generic parts.
type group struct {
	key   string
	parts []spec
}

// tableGroups are the sections (and parts of existing sections) read through the generic loader, in output order.
func tableGroups(box Box) []group {
	return []group{
		{key: "cycle", parts: []spec{
			{key: "preferences", table: "cycle_preferences"},
			{key: "skips", table: "cycle_skips"},
			{key: "predictions", table: "engine_prediction_log"},
		}},
		{key: "fertility", parts: []spec{
			{key: "logs", table: "fertility_logs"},
		}},
		{key: "health_log", parts: []spec{
			{key: "custom_items", table: "health_log_custom_items"},
			{key: "preferences", table: "health_log_preferences"},
		}},
		{key: "pregnancy_tools", parts: []spec{
			{key: "alerts", table: "pregnancy_alerts"},
			{key: "contraction_sessions", table: "pregnancy_contraction_sessions"},
			{key: "contractions", table: "pregnancy_contractions"},
			{key: "daily_extras", table: "pregnancy_daily_extras"},
			{key: "kick_sessions", table: "pregnancy_kick_sessions"},
			{key: "week_state", table: "pregnancy_week_user_state"},
		}},
		{key: "health_record", parts: []spec{
			{key: "record", table: "health_records"},
			{key: "pregnancies", table: "health_record_pregnancies"},
			{key: "documents", table: "record_documents"},
			{key: "document_files", table: "record_document_files"},
			{key: "document_links", table: "record_document_links"},
			{key: "emergency_card", table: "emergency_cards", hidden: []string{"public_token_hash"}},
			{key: "share_links", table: "health_share_links", hidden: []string{"token_hash", "code_hash", "code_payload"}},
			{key: "share_link_views", table: "health_share_link_views"},
		}},
		{key: "labs", parts: []spec{
			{key: "reports", table: "lab_reports"},
			{key: "markers", table: "lab_markers"},
			{key: "files", table: "lab_files", hidden: []string{"path"}},
			{key: "partner_requests", table: "partner_lab_requests", each: partnerPayload(box)},
		}},
		{key: "checkups", parts: []spec{
			{key: "types", table: "checkup_types"}, // her own checkup types (catalog rows have no user)
			{key: "records", table: "checkup_records"},
			{key: "settings", table: "user_checkup_settings"},
		}},
		{key: "vitals", parts: []spec{
			{key: "plan_items", table: "vital_plan_items"},
			{key: "readings", table: "vital_readings"},
			{key: "targets", table: "vital_targets"},
		}},
		{key: "care", parts: []spec{
			{key: "reminder_intakes", table: "reminder_intakes"},
		}},
		{key: "children", parts: []spec{
			{key: "children", table: "children"},
			{key: "guardians", table: "child_guardians"},
			{key: "measurements", table: "child_measurements"},
			{key: "milestones", table: "child_milestone_checks"},
			{key: "vaccines", table: "child_vaccine_doses"},
			{key: "feeds", table: "baby_feeds"},
			{key: "sleeps", table: "baby_sleeps"},
			{key: "diapers", table: "baby_diapers"},
			{key: "med_doses", table: "baby_med_doses"},
		}},
		{key: "family", parts: []spec{
			{key: "families", table: "families"},
			{key: "family_children", table: "family_children"},
			{key: "companions", table: "companions"},
			{key: "companion_grants", table: "companion_grants"},
			{key: "invites", table: "companion_invites", hidden: []string{"code_hash"}},
			{key: "activity", table: "companion_audit_logs", hidden: []string{"payload_sealed"}},
			{key: "attendance", table: "family_attendance"},
			// Alerts and notices about her own data (her side as the owner); a member's copy is the owner's data.
			{key: "member_alerts", table: "family_member_alerts"},
			{key: "member_notices", table: "family_member_notices"},
			{key: "nudges", table: "family_nudges"},
			{key: "owner_state", table: "family_owner_state"},
			{key: "proposals", table: "family_proposals",
				hidden: []string{"payload_sealed", "dedupe_key", "pending_key"}},
			{key: "share_requests", table: "family_share_requests"},
		}},
		{key: "todo", parts: []spec{
			{key: "tasks", table: "todo_tasks"},
			{key: "items", table: "todo_items"},
			{key: "suggestion_events", table: "todo_suggestion_events"},
		}},
		{key: "teen", parts: []spec{
			{key: "profile", table: "teen_profiles"},
			{key: "kit_checks", table: "teen_kit_checks"},
		}},
		{key: "challenges", parts: []spec{
			{key: "enrollments", table: "challenge_enrollments"},
			{key: "task_completions", table: "challenge_task_completions"},
			{key: "reward_grants", table: "challenge_reward_grants"},
			{key: "daily_completions", table: "user_challenge_completions"},
			{key: "task_template_completions", table: "user_task_completions"},
		}},
		{key: "assistant", parts: []spec{
			{key: "conversations", table: "assistant_conversations", each: assistantSummary(box)},
			{key: "messages", table: "assistant_messages", each: assistantBody(box)},
			{key: "handoffs", table: "assistant_handoffs", each: assistantHandoff(box)},
			{key: "settings", table: "assistant_profile_settings"},
		}},
		{key: "telemed", parts: []spec{
			{key: "bookings", table: "telemed_bookings", hidden: []string{"authority"}},
			{key: "consents", table: "telemed_booking_consents"},
			{key: "messages", table: "telemed_messages", each: telemedBody(box)},
			{key: "prescription_items", table: "telemed_prescription_items"},
			{key: "reviews", table: "telemed_reviews"},
		}},
		{key: "learning", parts: []spec{
			{key: "instructor", table: "learning_instructors"},
			{key: "grants", table: "learning_grants"},
			{key: "progress", table: "learning_progress"},
		}},
		{key: "plus", parts: []spec{
			{key: "subscriptions", table: "plus_subscriptions"},
			{key: "trials", table: "plus_trials"},
			{key: "invoices", table: "plus_invoices", hidden: []string{"authority"}},
			{key: "receipts", table: "plus_receipts"},
			{key: "store_purchases", table: "plus_store_purchases", hidden: []string{"purchase_token"}},
			{key: "usage_counters", table: "plus_usage_counters"},
		}},
		{key: "ai_credits", parts: []spec{
			{key: "grants", table: "ai_credit_grants"},
			{key: "spends", table: "ai_credit_spends"},
		}},
		{key: "notifications", parts: []spec{
			{key: "inbox", table: "user_notifications"},
			{key: "smart_messages", table: "message_inbox"},
			{key: "message_events", table: "message_events"},
		}},
		{key: "activity", parts: []spec{
			{key: "events", table: "analytics_events"},
			{key: "days", table: "user_activity_days"},
		}},
		{key: "account_records", parts: []spec{
			{key: "consent_events", table: "consent_events"},
			{key: "profile_backups", table: "user_profiles_dedupe_backup"},
			// Metadata of every stored file — never the bytes, the storage path or the hash.
			{key: "files", table: "files", hidden: []string{"path", "sha256"}},
		}},
	}
}

// shopExtra / directoryExtra are generic parts appended to the existing shop and directory sections.
var (
	shopExtra = []spec{
		{key: "carts", table: "shop_carts"},
		{key: "cart_items", table: "shop_cart_items"},
	}
	directoryExtra = []spec{
		// A venue listing she claimed as its owner, with its services and slots.
		{key: "owned_listings", table: "city_service_listings"},
		{key: "owned_listing_services", table: "city_service_services"},
		{key: "owned_listing_slots", table: "city_service_slots"},
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

// tables reads generic parts through the sqlc readers of export_gen.go.
type tables struct {
	db   *sql.DB
	once sync.Once
	read map[string]fetcher
}

// fetcher runs one table's sqlc query for the owner.
type fetcher func(ctx context.Context, uid uint64) ([]any, error)

// part is a spec as a section part for uid.
func (x *tables) part(ctx context.Context, uid uint64, s spec) part {
	return part{key: s.key, load: func() ([]any, error) { return x.rows(ctx, uid, s) }}
}

func (x *tables) rows(ctx context.Context, uid uint64, s spec) ([]any, error) {
	x.once.Do(func() { x.read = fetchers(x.db) })
	var list []any
	var err error
	cols := exportColumns[s.table]
	if f, ok := x.read[s.table]; ok {
		list, err = f(ctx, uid)
	} else if s.table == dedupeBackupTable {
		cols, list, err = dedupeBackupRows(ctx, x.db, uid)
	} else {
		err = errors.New("no reader")
	}
	if err != nil {
		return nil, fmt.Errorf("accountexport: %s: %w", s.table, err)
	}
	out := make([]any, 0, len(list))
	for _, r := range list {
		o := jsonx.NewObject()
		v := reflect.ValueOf(r)
		if len(cols) != v.NumField() {
			return nil, fmt.Errorf("accountexport: %s: %d columns, %d fields", s.table, len(cols), v.NumField())
		}
		for i, col := range cols {
			c := cell(v.Field(i))
			if str, isStr := c.(string); isStr && jsonTextColumns[s.table+"."+col] {
				c = model.DecodeJSONColumn(json.RawMessage(str))
			}
			o.Set(col, c)
		}
		if s.each != nil {
			s.each(uid, o)
		}
		out = append(out, o)
	}
	return out, nil
}

var (
	timeType = reflect.TypeOf(time.Time{})
	rawType  = reflect.TypeOf(json.RawMessage(nil))
	dateType = reflect.TypeOf(civildate.Date{})
)

// cell serialises one sqlc field like the former column-type switch: DATE "Y-m-d", DATETIME the Laravel datetime,
// JSON decoded, tinyint(1) as 0/1, DECIMAL / text as strings, integers and floats as numbers, NULL as null.
func cell(v reflect.Value) any {
	t := v.Type()
	if v.Kind() == reflect.Struct && t != timeType && t != dateType && v.FieldByName("Valid").IsValid() { // sql.Null*, civildate.NullDate
		if !v.FieldByName("Valid").Bool() {
			return nil
		}
		return cell(v.Field(0))
	}
	switch t {
	case timeType:
		return jsonx.DateTime(v.Interface().(time.Time)) //nolint:forcetypeassert // checked by type
	case dateType:
		return v.Interface().(civildate.Date).String() //nolint:forcetypeassert // checked by type
	case rawType:
		return model.DecodeJSONColumn(v.Interface().(json.RawMessage)) //nolint:forcetypeassert // checked by type
	}
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		if v.Bool() {
			return int64(1)
		}
		return int64(0)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint()
	case reflect.Float32, reflect.Float64:
		return v.Float()
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return string(v.Bytes())
		}
	}
	return v.Interface()
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
