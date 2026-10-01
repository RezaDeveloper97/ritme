package search

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	carestore "github.com/ritme/backend-go/internal/care/store"
	checkupstore "github.com/ritme/backend-go/internal/checkups/store"
	contentstore "github.com/ritme/backend-go/internal/content/store"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// The sources are read through the owning domains' services and sqlc stores (no SQL of its own): search
// is a read model over them. Every user-scoped read takes the authenticated user id.

// LogSource is the user's life mode and her v2 log entries (internal/healthlog, B-N3-01).
type LogSource interface {
	LifeMode(ctx context.Context, userID uint64) (string, error)
	Range(ctx context.Context, userID uint64, from, to civildate.Date) ([]healthlog.DayEntries, error)
}

// PeriodSource is the start of the user's latest period (cycle_histories, else the profile's
// last_period_start); ok false when she has none.
type PeriodSource interface {
	LatestPeriodStart(ctx context.Context, userID uint64) (civildate.Date, bool, error)
}

// Article is a published article's searchable fields (no body).
type Article struct {
	Slug     string
	Title    json.RawMessage // translatable
	Excerpt  json.RawMessage // translatable, may be nil
	Category string
	ReadTime int // minutes, 0 = unknown
}

// ArticleSource lists the published articles.
type ArticleSource interface {
	PublishedArticles(ctx context.Context) ([]Article, error)
}

// Checkup is an active checkup type the user can see.
type Checkup struct {
	ID       uint64
	Key      string
	Custom   bool // the user's own checkup
	Category string
	Title    json.RawMessage // translatable (custom: the user's own text under her keys)
	Subtitle json.RawMessage
}

// CheckupSource lists the checkup types visible to the user (shared catalog plus her own).
type CheckupSource interface {
	VisibleCheckups(ctx context.Context, userID uint64) ([]Checkup, error)
}

// Reminder is one of the user's care reminders (medication or appointment).
type Reminder struct {
	ID       uint64
	Kind     string // medication | appointment
	Title    string
	Subtitle string
	Active   bool
}

// ReminderSource lists the user's care reminders.
type ReminderSource interface {
	CareReminders(ctx context.Context, userID uint64) ([]Reminder, error)
}

// maxArticles caps one search's article scan (the catalogue is admin-curated and small).
const maxArticles = 1000

// NewSources builds the production sources over db.
func NewSources(db *sql.DB) (LogSource, PeriodSource, ArticleSource, CheckupSource, ReminderSource) {
	return healthlog.NewService(db), periods{cyclestore.New(db)}, articles{contentstore.New(db)},
		checkups{checkupstore.New(db)}, reminders{carestore.New(db)}
}

type periods struct{ q *cyclestore.Queries }

func (p periods) LatestPeriodStart(ctx context.Context, userID uint64) (civildate.Date, bool, error) {
	latest, err := p.q.GetLatestPeriod(ctx, userID)
	switch {
	case err == nil:
		return latest.PeriodStartDate, true, nil
	case !errors.Is(err, sql.ErrNoRows):
		return civildate.Date{}, false, fmt.Errorf("latest period: %w", err)
	}
	profile, err := p.q.GetProfileByUserID(ctx, userID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return civildate.Date{}, false, nil
	case err != nil:
		return civildate.Date{}, false, fmt.Errorf("profile: %w", err)
	}
	return profile.LastPeriodStart.Date, profile.LastPeriodStart.Valid, nil
}

type articles struct{ q *contentstore.Queries }

func (a articles) PublishedArticles(ctx context.Context) ([]Article, error) {
	rows, err := a.q.ListPublishedArticles(ctx, contentstore.ListPublishedArticlesParams{
		LangA: "", LangB: "", Pattern: "", Limit: maxArticles,
	})
	if err != nil {
		return nil, fmt.Errorf("articles: %w", err)
	}
	out := make([]Article, 0, len(rows))
	for _, r := range rows {
		art := Article{Slug: r.Slug, Title: r.Title, Category: r.Category.String}
		if r.Excerpt.Valid {
			art.Excerpt = r.Excerpt.V
		}
		if r.ReadTimeMinutes.Valid {
			art.ReadTime = int(r.ReadTimeMinutes.Int16)
		}
		out = append(out, art)
	}
	return out, nil
}

type checkups struct{ q *checkupstore.Queries }

func (c checkups) VisibleCheckups(ctx context.Context, userID uint64) ([]Checkup, error) {
	rows, err := c.q.ListCheckupPlanRows(ctx, checkupstore.ListCheckupPlanRowsParams{UserID: int64(userID)}) //nolint:gosec // G115: user ids fit int64
	if err != nil {
		return nil, fmt.Errorf("checkups: %w", err)
	}
	out := make([]Checkup, 0, len(rows))
	for _, r := range rows {
		t := r.CheckupType
		ck := Checkup{ID: t.ID, Key: t.Key.String, Custom: t.UserID.Valid, Category: t.Category, Title: t.Title}
		if t.Subtitle.Valid {
			ck.Subtitle = t.Subtitle.V
		}
		out = append(out, ck)
	}
	return out, nil
}

type reminders struct{ q *carestore.Queries }

func (r reminders) CareReminders(ctx context.Context, userID uint64) ([]Reminder, error) {
	meds, err := r.q.ListMedications(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("medications: %w", err)
	}
	visits, err := r.q.ListAppointments(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("appointments: %w", err)
	}
	out := make([]Reminder, 0, len(meds)+len(visits))
	for _, list := range [][]carestore.Reminder{meds, visits} {
		for _, m := range list {
			out = append(out, Reminder{ID: m.ID, Kind: m.Type, Title: m.Title, Subtitle: m.Subtitle.String, Active: m.IsActive})
		}
	}
	return out, nil
}
