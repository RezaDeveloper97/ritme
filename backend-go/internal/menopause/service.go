// Package menopause is the menopause API (CB-MENO-02, canvas boards nbl_Meno_Home / _Stage / _HotFlash / _Score and
// the dark Main home): the stage profile, the home's "today" read model, the hot-flash timer, the monthly symptom
// score and the patterns found in her own logs.
//
// Built on bloom (roadmap/DECISIONS.md #2), nothing duplicated:
//   - the profile is bloom's user_life_profiles.menopause_* columns (B-N2-01), written here column-wise;
//   - the daily symptoms, the bleeding choice and the triggers are log taxonomy slots in health_log_entries (B-N3-01),
//     read through healthlog.Service;
//   - patterns are φ cards of bloom's analysis engine (B-N3-07, analysis.Binary) with its minimum-data rules;
//   - upcoming checkups are the M4 checkups plan (checkups.LoadPlan);
//   - every list and clinical text (score questions and domains, bands, the bleeding alert, stage tips, the patterns
//     disclaimer, the checkup groups) is admin-editable catalog content (CB-MENO-01 seeds, needs_review).
//
// Health data: every read and write is scoped to the authenticated user; nothing is logged.
package menopause

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/menopause/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/vitals"
)

// Catalog groups the package reads (CB-MENO-01 seeds, audience menopause).
const (
	GroupScoreItems    = "meno_score_items"
	GroupScoreBands    = "meno_score_bands"
	GroupAlerts        = "meno_alerts"
	GroupTips          = "meno_tips"
	GroupCheckupGroups = "meno_checkup_groups"
)

// Catalog codes the package looks up.
const (
	AlertPostmenopausalBleeding = "postmenopausal_bleeding"
	TipPatternsDisclaimer       = "patterns_disclaimer"
	tipStagePrefix              = "stage_"
)

// CatalogSource reads catalog groups (catalog.Reader: active items in order, cached per group).
type CatalogSource interface {
	Items(ctx context.Context, group string) ([]catalog.Item, error)
}

// DB is the database the service reads and writes through (a *sql.DB; the checkups plan reads through it too).
type DB interface {
	store.DBTX
}

// Service is the menopause API's logic.
type Service struct {
	db      DB
	q       *store.Queries
	logs    *healthlog.Service
	catalog CatalogSource
	vitals  *vitals.Service // the report's blood pressure (bloom B-N6-01 merged readings); nil on a transaction
}

// NewService returns a Service on db.
func NewService(db DB, cat CatalogSource) *Service {
	s := &Service{db: db, q: store.New(db), logs: healthlog.NewService(db), catalog: cat}
	if conn, ok := db.(*sql.DB); ok {
		s.vitals = vitals.NewService(conn)
	}
	return s
}

func tehranNow(now time.Time) sql.NullTime {
	return sql.NullTime{Time: now.In(civildate.Tehran).Truncate(time.Second), Valid: true}
}

// item is the active catalog item `code` of group (nil when inactive or missing).
func (s *Service) item(ctx context.Context, group, code string) (*catalog.Item, error) {
	items, err := s.catalog.Items(ctx, group)
	if err != nil {
		return nil, fmt.Errorf("menopause: load catalog %s: %w", group, err)
	}
	i := slices.IndexFunc(items, func(it catalog.Item) bool { return it.Code == code })
	if i < 0 {
		return nil, nil
	}
	return &items[i], nil
}

// Profile is the user's stored menopause answers (bloom B-N2-01 columns) with the stage rule applied.
func (s *Service) Profile(ctx context.Context, userID uint64, today civildate.Date) (Stage, error) {
	row, err := s.q.GetMenopauseProfile(ctx, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Stage{}, fmt.Errorf("menopause: load profile: %w", err)
	}
	return ResolveStage(answersOf(row), today), nil
}

func answersOf(row store.GetMenopauseProfileRow) Answers {
	a := Answers{}
	if row.MenopauseStage.Valid {
		a.Stage = row.MenopauseStage.String
	}
	if row.MenopauseLastPeriod.Valid {
		d := row.MenopauseLastPeriod.Date
		a.LastPeriod = &d
	}
	if row.MenopauseSurgical.Valid {
		b := row.MenopauseSurgical.Bool
		a.Surgical = &b
	}
	if row.MenopauseHrt.Valid {
		b := row.MenopauseHrt.Bool
		a.HRT = &b
	}
	return a
}

// ProfileChange is a validated PUT /menopause/profile: Set lists the keys sent (null clears).
type ProfileChange struct {
	Set        map[string]bool
	Stage      string
	LastPeriod *civildate.Date
	Surgical   *bool
	HRT        *bool
}

func nullBool(b *bool) sql.NullBool {
	if b == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *b, Valid: true}
}

// SaveProfile applies a partial change to the menopause columns and returns the resolved profile.
func (s *Service) SaveProfile(ctx context.Context, userID uint64, ch ProfileChange, now time.Time) (Stage, error) {
	row, err := s.q.GetMenopauseProfile(ctx, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Stage{}, fmt.Errorf("menopause: load profile: %w", err)
	}
	a := answersOf(row)
	if ch.Set["stage"] {
		a.Stage = ch.Stage
	}
	if ch.Set["last_period"] {
		a.LastPeriod = ch.LastPeriod
	}
	if ch.Set["surgical"] {
		a.Surgical = ch.Surgical
	}
	if ch.Set["hrt"] {
		a.HRT = ch.HRT
	}
	p := store.UpsertMenopauseProfileParams{
		UserID: userID, MenopauseSurgical: nullBool(a.Surgical), MenopauseHrt: nullBool(a.HRT), Now: tehranNow(now),
	}
	if a.Stage != "" {
		p.MenopauseStage = sql.NullString{String: a.Stage, Valid: true}
	}
	if a.LastPeriod != nil {
		p.MenopauseLastPeriod = civildate.NullDate{Date: *a.LastPeriod, Valid: true}
	}
	if err := s.q.UpsertMenopauseProfile(ctx, p); err != nil {
		return Stage{}, fmt.Errorf("menopause: save profile: %w", err)
	}
	return ResolveStage(a, civildate.InTehran(now)), nil
}

// StageTip is the catalog meno_tips `stage_<stage>` item (nil when there is no stage or the tip is inactive).
func (s *Service) StageTip(ctx context.Context, st Stage) (*catalog.Item, error) {
	if st.Stage == "" {
		return nil, nil
	}
	return s.item(ctx, GroupTips, tipStagePrefix+st.Stage)
}
