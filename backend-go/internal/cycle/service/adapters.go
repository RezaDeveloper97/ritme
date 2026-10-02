package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ritme/backend-go/internal/cycle/legacy"
	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/cycle/recommendation"
	"github.com/ritme/backend-go/internal/cycle/store"
	hlmodel "github.com/ritme/backend-go/internal/healthlog/model"
	hlstore "github.com/ritme/backend-go/internal/healthlog/store"
)

// Row → engine-model converters. The engine packages (metrics, resolver, view, legacy) are pure;
// these are the only place the sqlc rows are mapped onto their inputs.

// HistoryFromRow maps a cycle_histories row onto model.History (CycleHistory casts).
func HistoryFromRow(r store.CycleHistory) model.History {
	h := model.History{
		ID:          int64(r.ID), //nolint:gosec // G115: auto-increment ids fit int64
		PeriodStart: r.PeriodStartDate,
		IsConfirmed: r.IsConfirmed,
		IsEstimated: r.IsEstimated,
		Source:      r.Source,
	}
	if r.PeriodEndDate.Valid {
		h.PeriodEnd = r.PeriodEndDate.Date
	}
	if r.CycleLength.Valid {
		h.CycleLength = model.Int(int(r.CycleLength.Int32))
	}
	if r.BleedingLength.Valid {
		h.BleedingLength = model.Int(int(r.BleedingLength.Int32))
	}
	if r.DataQualityFlags.Valid {
		var flags []string
		if json.Unmarshal(r.DataQualityFlags.V, &flags) == nil {
			h.DataQualityFlags = flags
		}
	}
	return h
}

// HistoriesFromRows maps rows in their order; the newest row, when the bleeding log created it, counts
// as confirmed (model.PromoteLoggedStart, B-N3-14b / D-55).
func HistoriesFromRows(rows []store.CycleHistory) []model.History {
	out := make([]model.History, len(rows))
	for i, r := range rows {
		out[i] = HistoryFromRow(r)
	}
	model.PromoteLoggedStart(out)
	return out
}

// ProfileFromRow maps a user_profiles row onto model.Profile (nil row = no profile).
func ProfileFromRow(r *store.UserProfile) *model.Profile {
	if r == nil {
		return nil
	}
	p := &model.Profile{Goal: r.UserGoal}
	if r.LastPeriodStart.Valid {
		p.LastPeriodStart = r.LastPeriodStart.Date
	}
	if r.Birthday.Valid {
		p.Birthday = r.Birthday.Date
	}
	if r.CycleDuration.Valid {
		p.CycleDuration = model.Int(int(r.CycleDuration.Int16))
	}
	if r.PeriodDuration.Valid {
		p.PeriodDuration = model.Int(int(r.PeriodDuration.Int16))
	}
	return p
}

// DailyLogFromRow builds the legacy engine's view of a daily_health_logs row; Source is
// DailyHealthLog::toArray() (emitted as `source_daily_log_data`).
func DailyLogFromRow(r store.DailyHealthLog) *legacy.DailyLog {
	l := hlmodel.FromRow(hlstore.DailyHealthLog(r))
	return &legacy.DailyLog{
		Spotting:             l.Bool("spotting"),
		VaginalDryness:       l.Bool("vaginal_dryness"),
		Fatigue:              l.Bool("fatigue"),
		DischargeTexture:     l.Str("discharge_texture"),
		OvarianPainIntensity: l.Str("ovarian_pain_intensity"),
		BloatingIntensity:    l.Str("bloating_intensity"),
		HeadacheIntensity:    l.Str("headache_intensity"),
		PelvicPainIntensity:  l.Str("pelvic_pain_intensity"),
		StomachAcheIntensity: l.Str("stomach_ache_intensity"),
		SleepQuality:         l.Str("sleep_quality"),
		SexualDesire:         l.Str("sexual_desire"),
		SexualActivities:     l.Strings("sexual_activities"),
		Moods:                l.Strings("moods"),
		Source:               l.ToArray(),
	}
}

// RecommendationSource is the sqlc adapter behind recommendation.Repository.
type RecommendationSource struct{ Q store.Querier }

var _ recommendation.Source = RecommendationSource{}

// ActiveRows implements recommendation.Source: active rows ordered by sort_order, id.
func (s RecommendationSource) ActiveRows(ctx context.Context) ([]recommendation.Row, error) {
	rows, err := s.Q.ListActiveRecommendations(ctx)
	if err != nil {
		return nil, fmt.Errorf("cycle: load recommendations: %w", err)
	}
	out := make([]recommendation.Row, len(rows))
	for i, r := range rows {
		out[i] = recommendation.Row{
			ID:             int64(r.ID), //nolint:gosec // G115: auto-increment ids fit int64
			Key:            nullString(r.Key.String, r.Key.Valid),
			Type:           r.Type,
			Title:          nullJSON(r.Title.V, r.Title.Valid),
			Text:           r.Text,
			CyclePhase:     nullString(r.CyclePhase.String, r.CyclePhase.Valid),
			CycleSubphases: nullJSON(r.CycleSubphases.V, r.CycleSubphases.Valid),
			SymptomTrigger: nullString(r.SymptomTrigger.String, r.SymptomTrigger.Valid),
			IsActive:       r.IsActive,
			SortOrder:      int(r.SortOrder),
		}
	}
	return out, nil
}

// AnyExists implements recommendation.Source: any row at all (inactive rows count).
func (s RecommendationSource) AnyExists(ctx context.Context) (bool, error) {
	ok, err := s.Q.AnyRecommendationExists(ctx)
	if err != nil {
		return false, fmt.Errorf("cycle: recommendations exist: %w", err)
	}
	return ok, nil
}

func nullString(s string, valid bool) *string {
	if !valid {
		return nil
	}
	return &s
}

func nullJSON(v json.RawMessage, valid bool) json.RawMessage {
	if !valid {
		return nil
	}
	return v
}
