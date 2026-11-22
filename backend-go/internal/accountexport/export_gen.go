package accountexport

// T-M12-14b: the typed readers of the generic export parts. One sqlc query per table (db/queries/profile/export_*.sql,
// explicit column list = the exported columns, hidden ones are not selected), registered here by table name.
// exportColumns lists the selected columns in order — the keys of each exported object — and is checked against the
// live schema by TestAccountExport_ColumnGuard (a new column must be exported here or hidden in tables.go).

import (
	"context"
	"database/sql"

	"github.com/ritme/backend-go/internal/profile/store"
)

// exportColumns are the columns each generic table exports, in SELECT order.
var exportColumns = map[string][]string{
	"cycle_preferences":              {"id", "user_id", "lengths_auto", "created_at", "updated_at"},
	"cycle_skips":                    {"id", "user_id", "expected_start", "created_at", "updated_at"},
	"engine_prediction_log":          {"id", "user_id", "cycle_start", "kind", "predicted_next", "basis_cycles", "ovulation_source", "confidence", "actual_start", "error_days", "created_at"},
	"fertility_logs":                 {"id", "user_id", "log_date", "lh_test", "cervical_mucus", "bbt_time", "created_at", "updated_at"},
	"health_log_custom_items":        {"id", "user_id", "category", "param", "label", "created_at", "updated_at", "deleted_at"},
	"health_log_preferences":         {"id", "user_id", "mode", "category_order", "hidden", "pinned", "created_at", "updated_at"},
	"pregnancy_alerts":               {"id", "user_id", "alert_level", "alert_type", "title", "message", "pregnancy_week", "trigger_symptoms", "medical_history_flags", "is_read", "is_dismissed", "read_at", "dismissed_at", "recommended_actions", "created_at", "updated_at"},
	"pregnancy_contraction_sessions": {"id", "user_id", "started_at", "ended_at", "alert_at", "active_lock", "created_at", "updated_at"},
	"pregnancy_contractions":         {"id", "session_id", "user_id", "started_at", "ended_at", "active_lock", "created_at", "updated_at"},
	"pregnancy_daily_extras":         {"id", "user_id", "log_date", "mood", "water_glasses", "heartburn_severity", "constipation_severity", "visit_note", "created_at", "updated_at"},
	"pregnancy_kick_sessions":        {"id", "user_id", "started_at", "ended_at", "kicks", "tenth_kick_at", "last_kick_at", "pregnancy_week", "active_lock", "created_at", "updated_at"},
	"pregnancy_week_user_state":      {"id", "user_id", "week", "done_task_keys", "created_at", "updated_at"},
	"health_records":                 {"id", "user_id", "blood_type", "allergies", "allergies_on_emergency_card", "surgeries", "family_history", "created_at", "updated_at"},
	"health_record_pregnancies":      {"id", "user_id", "outcome", "ended_on", "baby_count", "created_at", "updated_at"},
	"record_documents":               {"id", "user_id", "kind", "title", "document_date", "ended_on", "centre", "doctor", "note", "extracted", "review_state", "created_at", "updated_at"},
	"record_document_files":          {"id", "user_id", "document_id", "file_id", "position", "created_at", "updated_at"},
	"record_document_links":          {"id", "user_id", "document_id", "target_type", "target_id", "state", "created_at", "updated_at"},
	"emergency_cards":                {"id", "user_id", "show_on_lock_screen", "show_pregnancy", "contact_name", "contact_relation", "contact_phone", "insurance_label", "insurance_last4", "public_enabled_at", "public_view_count", "public_last_viewed_at", "created_at", "updated_at"},
	"health_share_links":             {"id", "user_id", "kind", "label", "payload", "sections", "include_losses", "range_from", "range_to", "expires_at", "revoked_at", "view_count", "last_viewed_at", "created_at", "updated_at"},
	"health_share_link_views":        {"id", "share_link_id", "user_id", "via", "device", "browser", "viewed_at"},
	"lab_reports":                    {"id", "user_id", "source", "category", "title", "taken_on", "fasting", "lab_name", "status", "progress", "error_code", "quota_at", "verified_at", "interpretation", "interpreted_at", "interpret_count", "feedback", "feedback_note", "feedback_at", "created_at", "updated_at"},
	"lab_markers":                    {"id", "lab_id", "user_id", "code", "name", "value", "value_text", "unit", "ref_low", "ref_high", "ref_text", "confidence", "source", "sort_order", "created_at", "updated_at", "confirmed_at"},
	"lab_files":                      {"id", "lab_id", "user_id", "page", "mime", "size_bytes", "created_at", "updated_at"},
	"partner_lab_requests":           {"id", "reference", "user_id", "service_id", "status", "partner_ref", "partner_status", "preferred_date", "preferred_window", "consent_share", "consent_import", "payload_enc", "imported_lab_id", "closed_at", "purged_at", "created_at", "updated_at"},
	"checkup_types":                  {"id", "key", "user_id", "category", "title", "subtitle", "why", "performed_by", "icon", "tone", "interval_months", "interval_months_max", "age_min", "age_max", "cycle_day_from", "cycle_day_to", "remind_lead_days", "prep_steps", "guide_steps", "finding_options", "hide_in_pregnancy", "audiences", "is_active", "sort_order", "source_note", "created_at", "updated_at", "informational"},
	"checkup_records":                {"id", "user_id", "checkup_type_id", "done_on", "result", "findings", "note", "has_attachment", "next_due_on", "created_at", "updated_at", "test_kind"},
	"user_checkup_settings":          {"id", "user_id", "checkup_type_id", "enabled", "remind", "created_at", "updated_at"},
	"vital_plan_items":               {"id", "user_id", "type", "slot", "days", "remind_at", "sort_order", "created_at", "updated_at"},
	"vital_readings":                 {"id", "user_id", "type", "measured_at", "systolic", "diastolic", "pulse", "arm", "position", "glucose_mg_dl", "glucose_unit", "context", "method", "note", "created_at", "updated_at"},
	"vital_targets":                  {"id", "user_id", "context", "low_mgdl", "high_mgdl", "set_by", "note", "created_at", "updated_at"},
	"reminder_intakes":               {"id", "user_id", "reminder_id", "intake_date", "slot", "taken_at", "created_at", "updated_at"},
	"children":                       {"id", "owner_id", "name", "birth_date", "sex", "birth_weight_kg", "birth_length_cm", "birth_head_cm", "delivery_type", "created_at", "updated_at", "created_by", "updated_by"},
	"child_guardians":                {"id", "child_id", "user_id", "level", "sub_scopes", "status", "invited_by", "link_id", "created_at", "updated_at"},
	"child_measurements":             {"id", "child_id", "measured_on", "weight_kg", "length_cm", "head_cm", "created_at", "updated_at", "created_by", "updated_by"},
	"child_milestone_checks":         {"id", "child_id", "milestone_code", "checked_on", "created_at", "updated_at", "created_by", "updated_by"},
	"child_vaccine_doses":            {"id", "child_id", "dose_code", "given_on", "note", "created_at", "updated_at", "created_by", "updated_by"},
	"baby_feeds":                     {"id", "child_id", "type", "started_at", "ended_at", "active_side", "side_started_at", "last_side", "left_seconds", "right_seconds", "duration_seconds", "amount_ml", "note", "active_lock", "created_at", "updated_at", "created_by", "updated_by"},
	"baby_sleeps":                    {"id", "child_id", "started_at", "ended_at", "note", "active_lock", "created_at", "updated_at", "created_by", "updated_by"},
	"baby_diapers":                   {"id", "child_id", "changed_at", "kind", "note", "created_at", "updated_at", "created_by", "updated_by"},
	"baby_med_doses":                 {"id", "child_id", "name", "dose_text", "given_at", "note", "created_by", "updated_by", "created_at", "updated_at"},
	"families":                       {"id", "owner_id", "spouse_user_id", "companion_id", "created_at", "updated_at"},
	"family_children":                {"id", "family_id", "child_id", "created_at", "updated_at"},
	"companions":                     {"id", "owner_id", "companion_user_id", "type", "status", "display_name", "invited_at", "accepted_at", "revoked_at", "revoked_by", "created_at", "updated_at", "role", "template", "expires_at", "urgent_sms", "owner_confirmed_at", "goal", "share_pain", "share_mood", "share_energy", "share_sleep", "action_prefs"},
	"companion_grants":               {"id", "companion_id", "section", "level", "created_at", "updated_at", "mode_epoch"},
	"companion_invites":              {"id", "companion_id", "owner_id", "phone", "attempts", "expires_at", "used_at", "used_by_id", "revoked_at", "created_at", "updated_at"},
	"companion_audit_logs":           {"id", "owner_id", "actor_id", "companion_id", "section", "action", "created_at", "subject_type", "subject_id", "diff_json", "payload_key_id", "undo_deadline", "undone_at", "undone_by", "proposal_id"},
	"family_attendance":              {"id", "reminder_id", "member_id", "created_at"},
	"family_member_alerts":           {"id", "member_id", "owner_id", "link_id", "source_alert_id", "source_hit", "level", "created_at", "seen_at", "sms_sent_at"},
	"family_member_notices":          {"id", "member_id", "owner_id", "link_id", "kind", "dedupe_key", "due_at", "sent_at", "deferred_reason", "notification_id", "created_at"},
	"family_nudges":                  {"id", "owner_id", "member_id", "link_id", "target_type", "target_id", "local_date", "slot", "sent_at"},
	"family_owner_state":             {"owner_id", "mode", "mode_epoch", "review_pending", "updated_at"},
	"family_proposals":               {"id", "subject_type", "subject_id", "link_id", "actor_id", "approver_id", "kind", "domain", "payload_key_id", "status", "expires_at", "decided_by", "decided_at", "applied_ref", "created_at", "updated_at"},
	"family_share_requests":          {"id", "from_user_id", "to_user_id", "link_id", "domains", "status", "expires_at", "decided_at", "created_at"},
	"todo_tasks":                     {"id", "user_id", "title", "note", "category", "due_date", "due_time", "remind", "done_at", "suggestion_key", "created_at", "updated_at", "created_by", "updated_by", "scope", "assigned_to"},
	"todo_items":                     {"id", "task_id", "user_id", "title", "due_date", "done_at", "sort_order", "created_at", "updated_at", "created_by", "updated_by"},
	"todo_suggestion_events":         {"id", "user_id", "suggestion_key", "ref_date", "action", "created_at", "updated_at"},
	"teen_profiles":                  {"id", "user_id", "age_band", "menarche", "parent_note", "created_at", "updated_at"},
	"teen_kit_checks":                {"id", "user_id", "item_code", "created_at", "updated_at"},
	"challenge_enrollments":          {"id", "user_id", "challenge_id", "started_on", "status", "completed_at", "created_at", "updated_at"},
	"challenge_task_completions":     {"id", "enrollment_id", "task_id", "day_number", "completed_at"},
	"challenge_reward_grants":        {"id", "user_id", "challenge_id", "enrollment_id", "badge_code", "badge_title", "plus_days", "subscription_id", "granted_at"},
	"user_challenge_completions":     {"id", "user_id", "challenge_id", "completion_date", "completed_at", "created_at", "updated_at"},
	"user_task_completions":          {"id", "user_id", "task_template_id", "completion_date", "completed_at", "created_at", "updated_at"},
	"assistant_conversations":        {"id", "user_id", "department_id", "title", "triage_level", "emergency", "message_count", "summary", "summary_at", "summary_message_count", "last_message_at", "created_at", "updated_at"},
	"assistant_messages":             {"id", "conversation_id", "role", "body", "photo_file_id", "card", "triage_level", "guardrail", "chips", "status", "created_at", "updated_at"},
	"assistant_handoffs":             {"id", "conversation_id", "user_id", "booking_id", "summary", "triage_level", "created_at", "updated_at"},
	"assistant_profile_settings":     {"id", "user_id", "shared", "created_at", "updated_at"},
	"telemed_bookings":               {"id", "reference", "user_id", "doctor_id", "mode", "duration_minutes", "starts_at", "ends_at", "status", "slot_key", "hold_expires_at", "for_whom", "child_id", "patient_name", "reason", "note", "price_rials", "discount_rials", "discount_source", "total_rials", "payment_status", "gateway", "ref_id", "card_pan", "paid_at", "refund_id", "refunded_rials", "refunded_at", "appointment_id", "reschedules", "cancelled_at", "completed_at", "created_at", "updated_at"},
	"telemed_booking_consents":       {"id", "booking_id", "scope", "granted_at", "revoked_at", "created_at", "updated_at"},
	"telemed_messages":               {"id", "booking_id", "sender", "admin_id", "body", "file_id", "read_at", "created_at", "updated_at"},
	"telemed_prescription_items":     {"id", "booking_id", "drug", "dose", "form", "times", "duration_days", "instructions", "reminder_id", "added_at", "created_at", "updated_at"},
	"telemed_reviews":                {"id", "doctor_id", "user_id", "booking_id", "rating", "body", "is_visible", "created_at", "updated_at"},
	"learning_instructors":           {"id", "user_id", "display_name", "title", "bio", "status", "approved_at", "approved_by", "revoked_at", "created_at", "updated_at"},
	"learning_grants":                {"id", "instructor_id", "phone", "user_id", "group_id", "course_id", "duration", "duration_days", "until_date", "status", "activated_at", "expires_at", "revoked_at", "seen_at", "created_at", "updated_at"},
	"learning_progress":              {"id", "user_id", "lesson_id", "course_id", "position_seconds", "percent", "completed_at", "last_seen_at", "created_at", "updated_at"},
	"plus_subscriptions":             {"id", "user_id", "plan_id", "invoice_id", "status", "source", "starts_at", "ends_at", "auto_renew", "canceled_at", "created_at", "updated_at"},
	"plus_trials":                    {"id", "user_id", "started_at", "ends_at", "created_at", "updated_at"},
	"plus_invoices":                  {"id", "reference", "user_id", "plan_id", "kind", "credit_pack_id", "credits", "duration_months", "duration_minutes", "status", "currency", "subtotal_rials", "discount_rials", "vat_rate_bps", "vat_rials", "total_rials", "discount_code_id", "discount_code", "gateway", "expires_at", "paid_at", "created_at", "updated_at"},
	"plus_receipts":                  {"id", "invoice_id", "user_id", "gateway", "ref_id", "card_pan", "amount_rials", "paid_at", "created_at", "updated_at"},
	"plus_store_purchases":           {"id", "user_id", "store", "sku", "order_id", "purchase_time", "created_at", "updated_at"},
	"plus_usage_counters":            {"id", "user_id", "feature", "period_start", "used", "created_at", "updated_at"},
	"ai_credit_grants":               {"id", "user_id", "kind", "subscription_id", "trial_id", "invoice_id", "amount", "remaining", "note", "created_at", "updated_at"},
	"ai_credit_spends":               {"id", "user_id", "feature", "credits", "plan_credits", "extra_credits", "allocations", "status", "created_at", "updated_at"},
	"user_notifications":             {"id", "user_id", "type", "title", "body", "action_url", "data", "read_at", "created_at", "updated_at", "link_id", "pushed_at"},
	"message_inbox":                  {"id", "user_id", "rule_id", "version", "rule_key", "slot", "kind", "level", "state", "open_flag", "evidence_key", "hit_count", "evidence", "first_at", "last_at", "acked_at", "noted_at", "resolved_at", "archived_at", "legacy_alert_id", "created_at", "updated_at"},
	"message_events":                 {"id", "rule_id", "user_id", "variant", "event", "created_at", "version", "slot"},
	"analytics_events":               {"id", "user_id", "name", "props", "occurred_on", "occurred_at", "created_at"},
	"user_activity_days":             {"user_id", "day", "session", "logged"},
	"consent_events":                 {"id", "user_id", "consent", "version", "action", "scope", "source", "created_at"},
	"files":                          {"id", "user_id", "purpose", "visibility", "mime", "size_bytes", "created_at", "updated_at"},
	"shop_carts":                     {"id", "user_id", "created_at", "updated_at", "discount_code_id"},
	"shop_cart_items":                {"id", "cart_id", "variant_id", "quantity", "created_at", "updated_at"},
	"city_service_listings":          {"id", "category_id", "city_id", "name", "district", "address", "phone", "description", "price_text", "hours_text", "photos", "booking_mode", "booking_url", "sort_order", "is_active", "created_at", "updated_at", "status", "verified_at", "owner_user_id", "lat", "lng", "about", "age_ranges", "amenities", "weekly_hours", "photo_file_ids", "website_url", "cancel_free_hours", "cancel_policy", "rules", "rating_avg", "rating_count"},
	"city_service_services":          {"id", "listing_id", "kind", "name", "description", "format", "duration_minutes", "age_min_months", "age_max_months", "capacity", "price_toman", "package_sessions", "package_validity_days", "sort_order", "is_active", "created_at", "updated_at"},
	"city_service_slots":             {"id", "listing_id", "service_id", "starts_at", "ends_at", "capacity", "booked_count", "status", "created_at", "updated_at"},
}

// jsonTextColumns are the exported columns the schema keeps as text with a json_valid CHECK (not spelled json in the
// migrations, so sqlc types them string): decoded like the JSON columns. Checked against the schema by the column guard.
var jsonTextColumns = map[string]bool{
	"pregnancy_alerts.trigger_symptoms":        true,
	"pregnancy_alerts.medical_history_flags":   true,
	"pregnancy_alerts.recommended_actions":     true,
	"checkup_types.title":                      true,
	"checkup_types.subtitle":                   true,
	"checkup_types.why":                        true,
	"checkup_types.prep_steps":                 true,
	"checkup_types.guide_steps":                true,
	"checkup_types.finding_options":            true,
	"checkup_types.audiences":                  true,
	"checkup_records.findings":                 true,
	"pregnancy_week_user_state.done_task_keys": true,
	"health_log_preferences.category_order":    true,
	"health_log_preferences.hidden":            true,
	"health_log_preferences.pinned":            true,
	"companion_audit_logs.diff_json":           true,
	"lab_reports.interpretation":               true,
	"health_records.allergies":                 true,
	"health_records.surgeries":                 true,
	"health_records.family_history":            true,
	"record_documents.extracted":               true,
	"health_share_links.sections":              true,
	"assistant_profile_settings.shared":        true,
	"assistant_messages.chips":                 true,
	"challenge_reward_grants.badge_title":      true,
	"analytics_events.props":                   true,
	"city_service_listings.photos":             true,
	"city_service_listings.age_ranges":         true,
	"city_service_listings.amenities":          true,
	"city_service_listings.weekly_hours":       true,
	"city_service_listings.photo_file_ids":     true,
	"consent_events.scope":                     true,
	"family_share_requests.domains":            true,
	"user_notifications.title":                 true,
	"user_notifications.body":                  true,
	"user_notifications.data":                  true,
	"child_guardians.sub_scopes":               true,
	"message_inbox.evidence":                   true,
	"ai_credit_spends.allocations":             true,
}

// fetchers are the sqlc readers by table (every query binds the owner's id as its only argument).
func fetchers(db *sql.DB) map[string]fetcher {
	q := store.New(db)
	return map[string]fetcher{
		"cycle_preferences":              fetch(q.AcctExportCyclePreferences),
		"cycle_skips":                    fetch(q.AcctExportCycleSkips),
		"engine_prediction_log":          fetch(q.AcctExportEnginePredictionLog),
		"fertility_logs":                 fetch(q.AcctExportFertilityLogs),
		"health_log_custom_items":        fetch(q.AcctExportHealthLogCustomItems),
		"health_log_preferences":         fetch(q.AcctExportHealthLogPreferences),
		"pregnancy_alerts":               fetch(q.AcctExportPregnancyAlerts),
		"pregnancy_contraction_sessions": fetch(q.AcctExportPregnancyContractionSessions),
		"pregnancy_contractions":         fetch(q.AcctExportPregnancyContractions),
		"pregnancy_daily_extras":         fetch(q.AcctExportPregnancyDailyExtras),
		"pregnancy_kick_sessions":        fetch(q.AcctExportPregnancyKickSessions),
		"pregnancy_week_user_state":      fetch(q.AcctExportPregnancyWeekUserState),
		"health_records":                 fetch(q.AcctExportHealthRecords),
		"health_record_pregnancies":      fetch(q.AcctExportHealthRecordPregnancies),
		"record_documents":               fetch(q.AcctExportRecordDocuments),
		"record_document_files":          fetch(q.AcctExportRecordDocumentFiles),
		"record_document_links":          fetch(q.AcctExportRecordDocumentLinks),
		"emergency_cards":                fetch(q.AcctExportEmergencyCards),
		"health_share_links":             fetch(q.AcctExportHealthShareLinks),
		"health_share_link_views":        fetch(q.AcctExportHealthShareLinkViews),
		"lab_reports":                    fetch(q.AcctExportLabReports),
		"lab_markers":                    fetch(q.AcctExportLabMarkers),
		"lab_files":                      fetch(q.AcctExportLabFiles),
		"partner_lab_requests":           fetch(q.AcctExportPartnerLabRequests),
		"checkup_types":                  fetch(q.AcctExportCheckupTypes),
		"checkup_records":                fetch(q.AcctExportCheckupRecords),
		"user_checkup_settings":          fetch(q.AcctExportUserCheckupSettings),
		"vital_plan_items":               fetch(q.AcctExportVitalPlanItems),
		"vital_readings":                 fetch(q.AcctExportVitalReadings),
		"vital_targets":                  fetch(q.AcctExportVitalTargets),
		"reminder_intakes":               fetch(q.AcctExportReminderIntakes),
		"children":                       fetch(q.AcctExportChildren),
		"child_guardians": fetchP(q.AcctExportChildGuardians, func(u int64) store.AcctExportChildGuardiansParams {
			return store.AcctExportChildGuardiansParams{Uid: u}
		}),
		"child_measurements":     fetch(q.AcctExportChildMeasurements),
		"child_milestone_checks": fetch(q.AcctExportChildMilestoneChecks),
		"child_vaccine_doses":    fetch(q.AcctExportChildVaccineDoses),
		"baby_feeds":             fetch(q.AcctExportBabyFeeds),
		"baby_sleeps":            fetch(q.AcctExportBabySleeps),
		"baby_diapers":           fetch(q.AcctExportBabyDiapers),
		"baby_med_doses":         fetch(q.AcctExportBabyMedDoses),
		"families":               fetchP(q.AcctExportFamilies, func(u int64) store.AcctExportFamiliesParams { return store.AcctExportFamiliesParams{Uid: u} }),
		"family_children":        fetch(q.AcctExportFamilyChildren),
		"companions":             fetchP(q.AcctExportCompanions, func(u int64) store.AcctExportCompanionsParams { return store.AcctExportCompanionsParams{Uid: u} }),
		"companion_grants": fetchP(q.AcctExportCompanionGrants, func(u int64) store.AcctExportCompanionGrantsParams {
			return store.AcctExportCompanionGrantsParams{Uid: u}
		}),
		"companion_invites":     fetch(q.AcctExportCompanionInvites),
		"companion_audit_logs":  fetch(q.AcctExportCompanionAuditLogs),
		"family_attendance":     fetch(q.AcctExportFamilyAttendance),
		"family_member_alerts":  fetch(q.AcctExportFamilyMemberAlerts),
		"family_member_notices": fetch(q.AcctExportFamilyMemberNotices),
		"family_nudges":         fetchP(q.AcctExportFamilyNudges, func(u int64) store.AcctExportFamilyNudgesParams { return store.AcctExportFamilyNudgesParams{Uid: u} }),
		"family_owner_state":    fetch(q.AcctExportFamilyOwnerState),
		"family_proposals": fetchP(q.AcctExportFamilyProposals, func(u int64) store.AcctExportFamilyProposalsParams {
			return store.AcctExportFamilyProposalsParams{Uid: u}
		}),
		"family_share_requests": fetchP(q.AcctExportFamilyShareRequests, func(u int64) store.AcctExportFamilyShareRequestsParams {
			return store.AcctExportFamilyShareRequestsParams{Uid: u}
		}),
		"todo_tasks":                 fetch(q.AcctExportTodoTasks),
		"todo_items":                 fetch(q.AcctExportTodoItems),
		"todo_suggestion_events":     fetch(q.AcctExportTodoSuggestionEvents),
		"teen_profiles":              fetch(q.AcctExportTeenProfiles),
		"teen_kit_checks":            fetch(q.AcctExportTeenKitChecks),
		"challenge_enrollments":      fetch(q.AcctExportChallengeEnrollments),
		"challenge_task_completions": fetch(q.AcctExportChallengeTaskCompletions),
		"challenge_reward_grants":    fetch(q.AcctExportChallengeRewardGrants),
		"user_challenge_completions": fetch(q.AcctExportUserChallengeCompletions),
		"user_task_completions":      fetch(q.AcctExportUserTaskCompletions),
		"assistant_conversations":    fetch(q.AcctExportAssistantConversations),
		"assistant_messages":         fetch(q.AcctExportAssistantMessages),
		"assistant_handoffs":         fetch(q.AcctExportAssistantHandoffs),
		"assistant_profile_settings": fetch(q.AcctExportAssistantProfileSettings),
		"telemed_bookings":           fetch(q.AcctExportTelemedBookings),
		"telemed_booking_consents":   fetch(q.AcctExportTelemedBookingConsents),
		"telemed_messages":           fetch(q.AcctExportTelemedMessages),
		"telemed_prescription_items": fetch(q.AcctExportTelemedPrescriptionItems),
		"telemed_reviews":            fetch(q.AcctExportTelemedReviews),
		"learning_instructors":       fetch(q.AcctExportLearningInstructors),
		"learning_grants":            fetch(q.AcctExportLearningGrants),
		"learning_progress":          fetch(q.AcctExportLearningProgress),
		"plus_subscriptions":         fetch(q.AcctExportPlusSubscriptions),
		"plus_trials":                fetch(q.AcctExportPlusTrials),
		"plus_invoices":              fetch(q.AcctExportPlusInvoices),
		"plus_receipts":              fetch(q.AcctExportPlusReceipts),
		"plus_store_purchases":       fetch(q.AcctExportPlusStorePurchases),
		"plus_usage_counters":        fetch(q.AcctExportPlusUsageCounters),
		"ai_credit_grants":           fetch(q.AcctExportAiCreditGrants),
		"ai_credit_spends":           fetch(q.AcctExportAiCreditSpends),
		"user_notifications":         fetch(q.AcctExportUserNotifications),
		"message_inbox":              fetch(q.AcctExportMessageInbox),
		"message_events":             fetch(q.AcctExportMessageEvents),
		"analytics_events":           fetch(q.AcctExportAnalyticsEvents),
		"user_activity_days":         fetch(q.AcctExportUserActivityDays),
		"consent_events":             fetch(q.AcctExportConsentEvents),
		"files":                      fetch(q.AcctExportFiles),
		"shop_carts":                 fetch(q.AcctExportShopCarts),
		"shop_cart_items":            fetch(q.AcctExportShopCartItems),
		"city_service_listings":      fetch(q.AcctExportCityServiceListings),
		"city_service_services":      fetch(q.AcctExportCityServiceServices),
		"city_service_slots":         fetch(q.AcctExportCityServiceSlots),
	}
}

// fetch adapts a sqlc reader taking the owner's id.
func fetch[R any](f func(context.Context, int64) ([]R, error)) fetcher {
	return func(ctx context.Context, uid uint64) ([]any, error) {
		list, err := f(ctx, int64(uid)) //nolint:gosec // G115: ids are BIGINT
		return toAny(list), err
	}
}

// fetchP adapts a sqlc reader whose parameters are a struct (a predicate naming the owner more than once).
func fetchP[P, R any](f func(context.Context, P) ([]R, error), mk func(int64) P) fetcher {
	return func(ctx context.Context, uid uint64) ([]any, error) {
		list, err := f(ctx, mk(int64(uid))) //nolint:gosec // G115: ids are BIGINT
		return toAny(list), err
	}
}

func toAny[R any](list []R) []any {
	out := make([]any, len(list))
	for i := range list {
		out[i] = list[i]
	}
	return out
}
