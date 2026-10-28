package accountexport

import (
	"context"
	"database/sql"

	conditionsstore "github.com/ritme/backend-go/internal/conditions/store"
	contrastore "github.com/ritme/backend-go/internal/contraception/store"
	ivfstore "github.com/ritme/backend-go/internal/ivf/store"
	lossstore "github.com/ritme/backend-go/internal/loss/store"
	menostore "github.com/ritme/backend-go/internal/menopause/store"
	pelvicstore "github.com/ritme/backend-go/internal/pelvic/store"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	ppstore "github.com/ritme/backend-go/internal/postpartum/store"
	"github.com/ritme/backend-go/internal/profile/model"
)

// pelvicFloor: {program, sessions, bladder_diary} (CB-PELV).
func pelvicFloor(db *sql.DB) loader {
	q := pelvicstore.New(db)
	return func(ctx context.Context, uid uint64) (any, error) {
		return object(
			part{key: "program", one: true, load: func() ([]any, error) {
				l, err := q.ExportPelvicProgram(ctx, uid)
				return rows("pelvic program", l, err, nil, nil, nil)
			}},
			part{key: "sessions", load: func() ([]any, error) {
				l, err := q.ExportPelvicSessions(ctx, uid)
				return rows("pelvic sessions", l, err, nil, nil, nil)
			}},
			part{key: "bladder_diary", load: func() ([]any, error) {
				l, err := q.ExportPelvicBladderLogs(ctx, uid)
				return rows("pelvic bladder diary", l, err, model.Casts{"uti_symptoms": model.CastArray}, nil, nil)
			}},
		)
	}
}

// contraception: {method, pill_pack (the daily pill log), reminders} (CB-CONTRA).
func contraception(db *sql.DB) loader {
	q := contrastore.New(db)
	return func(ctx context.Context, uid uint64) (any, error) {
		return object(
			part{key: "method", one: true, load: func() ([]any, error) {
				l, err := q.ExportContraceptionMethods(ctx, uid)
				return rows("contraception method", l, err, model.Casts{"followup_done": model.CastBoolean}, nil, nil)
			}},
			part{key: "pill_pack", load: func() ([]any, error) {
				l, err := q.ExportContraceptionPillLogs(ctx, uid)
				return rows("contraception pill log", l, err, model.Casts{"logged_at": model.CastDateTime}, nil, nil)
			}},
			part{key: "reminders", load: func() ([]any, error) {
				l, err := q.ExportContraceptionReminders(ctx, uid)
				return rows("contraception reminders", l, err, nil, nil, nil)
			}},
		)
	}
}

// menopause: {profile (user_life_profiles.menopause_*), logs (the daily symptom log, health_log_entries), hot_flashes,
// scores, treatments, treatment_intakes (adherence), side_effects} (CB-MENO).
func menopause(db *sql.DB) loader {
	q := menostore.New(db)
	return func(ctx context.Context, uid uint64) (any, error) {
		return object(
			part{key: "profile", one: true, load: func() ([]any, error) {
				l, err := q.ExportMenopauseProfile(ctx, uid)
				return rows("menopause profile", l, err,
					model.Casts{"menopause_surgical": model.CastBoolean, "menopause_hrt": model.CastBoolean}, nil, nil)
			}},
			part{key: "logs", load: func() ([]any, error) {
				l, err := q.ExportHealthLogEntries(ctx, uid)
				return rows("health log entries", l, err, nil, nil, nil)
			}},
			part{key: "hot_flashes", load: func() ([]any, error) {
				l, err := q.ExportHotFlashes(ctx, uid)
				return rows("hot flashes", l, err, model.Casts{"started_at": model.CastDateTime, "night": model.CastBoolean,
					"sweat": model.CastBoolean, "triggers": model.CastArray}, nil, nil)
			}},
			part{key: "scores", load: func() ([]any, error) {
				l, err := q.ExportMenopauseScores(ctx, uid)
				return rows("menopause scores", l, err, model.Casts{"answers": model.CastArray}, nil, nil)
			}},
			part{key: "treatments", load: func() ([]any, error) {
				l, err := q.ExportTreatmentItems(ctx, uid)
				return rows("treatment items", l, err, nil, nil, nil)
			}},
			part{key: "treatment_intakes", load: func() ([]any, error) {
				l, err := q.ExportTreatmentIntakes(ctx, uid)
				return rows("treatment intakes", l, err, model.Casts{"taken_at": model.CastDateTime}, nil, nil)
			}},
			part{key: "side_effects", load: func() ([]any, error) {
				l, err := q.ExportSideEffectLogs(ctx, uid)
				return rows("side effect logs", l, err, nil, nil, nil)
			}},
		)
	}
}

// ivf: {cycles (stage, outcome), meds (inventory), injections (dose logs with their sites), scans, tww (two-week wait
// moods), reminders} (CB-IVF).
func ivf(db *sql.DB) loader {
	q := ivfstore.New(db)
	return func(ctx context.Context, uid uint64) (any, error) {
		return object(
			part{key: "cycles", load: func() ([]any, error) {
				l, err := q.ExportIvfCycles(ctx, uid)
				// active_user_id only enforces one open cycle per user (a unique key helper), not her data.
				return rows("ivf cycles", l, err, model.Casts{"retrieval_at": model.CastDateTime, "transfer_at": model.CastDateTime,
					"next_scan_at": model.CastDateTime, "closed_at": model.CastDateTime, "notify_companion": model.CastBoolean},
					[]string{"active_user_id"}, nil)
			}},
			part{key: "meds", load: func() ([]any, error) {
				l, err := q.ExportIvfMeds(ctx, uid)
				return rows("ivf meds", l, err, model.Casts{"trigger_at": model.CastDateTime, "stock_counted_at": model.CastDateTime}, nil, nil)
			}},
			part{key: "injections", load: func() ([]any, error) {
				l, err := q.ExportIvfDoseLogs(ctx, uid)
				return rows("ivf dose logs", l, err, nil, nil, nil)
			}},
			part{key: "scans", load: func() ([]any, error) {
				l, err := q.ExportIvfScans(ctx, uid)
				if err != nil {
					return rows("ivf scans", l, err, nil, nil, nil)
				}
				out := make([]any, 0, len(l))
				for _, s := range l { // the follicle-bucket columns have digits sqlc folds (right_10_14 → Right1014)
					out = append(out, jsonx.Obj("id", s.ID, "user_id", s.UserID, "cycle_id", s.CycleID, "scan_date", s.ScanDate.String(),
						"right_lt_10", s.RightLt10, "right_10_14", s.Right1014, "right_15_17", s.Right1517, "right_18_plus", s.Right18Plus,
						"left_lt_10", s.LeftLt10, "left_10_14", s.Left1014, "left_15_17", s.Left1517, "left_18_plus", s.Left18Plus,
						"endometrium_mm", nullString(s.EndometriumMm), "e2", nullString(s.E2), "e2_unit", nullString(s.E2Unit),
						"notes", nullString(s.Notes), "created_at", nullDateTime(s.CreatedAt), "updated_at", nullDateTime(s.UpdatedAt)))
				}
				return out, nil
			}},
			part{key: "tww", load: func() ([]any, error) {
				l, err := q.ExportIvfTwwLogs(ctx, uid)
				return rows("ivf tww logs", l, err, nil, nil, nil)
			}},
			part{key: "reminders", load: func() ([]any, error) {
				l, err := q.ExportIvfReminders(ctx, uid)
				return rows("ivf reminders", l, err, nil, nil, nil)
			}},
		)
	}
}

// conditionPrograms: {enrolments, pain_diary, pmdd, pbac} (CB-COND).
func conditionPrograms(db *sql.DB) loader {
	q := conditionsstore.New(db)
	return func(ctx context.Context, uid uint64) (any, error) {
		return object(
			part{key: "enrolments", load: func() ([]any, error) {
				l, err := q.ExportConditionEnrolments(ctx, uid)
				return rows("condition enrolments", l, err, nil, nil, nil)
			}},
			part{key: "pain_diary", load: func() ([]any, error) {
				l, err := q.ExportConditionPainEntries(ctx, uid)
				return rows("condition pain diary", l, err, model.Casts{"pain_types": model.CastArray, "associated": model.CastArray,
					"missed_activity": model.CastBoolean}, nil, nil)
			}},
			part{key: "pmdd", load: func() ([]any, error) {
				l, err := q.ExportPmddEntries(ctx, uid)
				return rows("pmdd entries", l, err, model.Casts{"scores": model.CastArray}, nil, nil)
			}},
			part{key: "pbac", load: func() ([]any, error) {
				l, err := q.ExportPbacEntries(ctx, uid)
				return rows("pbac entries", l, err, model.Casts{"flooding": model.CastBoolean}, nil, nil)
			}},
		)
	}
}

// lossPath: {losses (the private note opened for her), moods} (CB-LOSS).
func lossPath(db *sql.DB, box Box) loader {
	q := lossstore.New(db)
	return func(ctx context.Context, uid uint64) (any, error) {
		return object(
			part{key: "losses", load: func() ([]any, error) {
				l, err := q.ExportPregnancyLosses(ctx, uid)
				return rows("pregnancy losses", l, err, model.Casts{"notify_companion": model.CastBoolean,
					"paused_reminders": model.CastArray, "companion_notified_at": model.CastDateTime,
					"content_stopped_at": model.CastDateTime, "next_step_at": model.CastDateTime, "note_updated_at": model.CastDateTime},
					nil, func(r *lossstore.PregnancyLoss, o *jsonx.OrderedMap) {
						var note any
						if r.PrivateNote.Valid && r.PrivateNote.String != "" && !box.Disabled() {
							note = opened(box.Open(r.PrivateNote.String, r.UserID, r.ID))
						}
						o.Set("private_note", note)
					})
			}},
			part{key: "moods", load: func() ([]any, error) {
				l, err := q.ExportPregnancyLossMoods(ctx, uid)
				return rows("pregnancy loss moods", l, err, nil, nil, nil)
			}},
		)
	}
}

// postpartumData: {profile (the birth, null when none), recovery_logs (the daily red-flag / postpartum-visit rows)}
// (B-N5-01, T-M9-06; D-96 extended by T-M9-06c). EPDS checks are left out on purpose (D-54, user decision 2026-10-23:
// neither the answers nor the totals). The recovery sheet's lochia / pain / breasts / feeds / sleep answers are
// health_log_entries rows, already exported under menopause.logs (every health_log_entries row of the owner).
func postpartumData(db *sql.DB) loader {
	q := ppstore.New(db)
	return func(ctx context.Context, uid uint64) (any, error) {
		return object(
			part{key: "profile", one: true, load: func() ([]any, error) {
				l, err := q.ExportPostpartumProfile(ctx, uid)
				return rows("postpartum profile", l, err, model.Casts{"pregnancy_closed_at": model.CastDateTime,
					"breastfeeding": model.CastBoolean}, nil, nil)
			}},
			part{key: "recovery_logs", load: func() ([]any, error) {
				l, err := q.ExportPostpartumRecoveryLogs(ctx, uid)
				return rows("postpartum recovery logs", l, err, model.Casts{"pad_soaked_within_hour": model.CastBoolean,
					"severe_pain": model.CastBoolean, "breathless": model.CastBoolean, "postpartum_visit": model.CastBoolean}, nil, nil)
			}},
		)
	}
}

func nullString(s sql.NullString) any {
	if !s.Valid {
		return nil
	}
	return s.String
}

func nullDateTime(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return jsonx.DateTime(t.Time)
}
