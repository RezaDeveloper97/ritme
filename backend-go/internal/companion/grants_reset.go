package companion

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	rootdb "github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/companion/store"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// ResetReason is why ResetModeScopedGrants ran (kept in the owner notice's data, never a health payload).
type ResetReason string

// Reset reasons (T-M9-02, FAM-H3): every start and end of a pregnancy episode.
const (
	ResetPregnancyActivated   ResetReason = "pregnancy_activated"
	ResetPregnancyDeactivated ResetReason = "pregnancy_deactivated"
	ResetPregnancyLoss        ResetReason = "pregnancy_loss"
	ResetPostpartumStarted    ResetReason = "postpartum_started"
)

// EventPregnancySharingReset is data.event of the owner notice ResetModeScopedGrants writes.
const EventPregnancySharingReset = "pregnancy_sharing_reset"

// ModeScopedSections are the grants that belong to one pregnancy episode: a member who could see one pregnancy must
// never see the next one from its first day (SPEC v6 §1.7, gap report 05 FAM-H3). M11 replaces this with the mode
// epoch; until then the pregnancy grant is the only mode-scoped grant.
var ModeScopedSections = []Section{SectionPregnancy}

// ResetModeScopedGrants sets every mode-scoped (pregnancy) grant of ownerID's links to none. For each link that held
// one it writes a grants_changed audit row on the section (actor = the owner, whose mode change caused it) and one
// owner-inbox notice («اشتراک بارداری با همراهت متوقف شد…») whose data carries the diff {pregnancy: [old, none]}
// (the audit table has no payload column). Pass the caller's transaction as db when there is one, so the reset
// commits or rolls back with the mode change; otherwise call it right after the commit. Reports how many links lost
// a grant. at is the caller's clock (the mode change's timestamp).
func ResetModeScopedGrants(ctx context.Context, db store.DBTX, ownerID uint64, reason ResetReason, at time.Time) (int, error) {
	if ownerID == 0 {
		return 0, nil
	}
	q := store.New(db)
	now := at.In(civildate.Tehran).Truncate(time.Second)
	links, err := q.ListOwnerCompanions(ctx, ownerID)
	if err != nil {
		return 0, fmt.Errorf("companion: reset grants: links: %w", err)
	}
	var languages []string
	reset := 0
	for _, c := range links {
		grants, err := q.ListGrants(ctx, c.ID)
		if err != nil {
			return reset, fmt.Errorf("companion: reset grants: grants: %w", err)
		}
		diff := map[string][2]string{}
		for _, g := range grants {
			sec := Section(g.Section)
			if !isModeScoped(sec) || !Level(g.Level).Valid() || Level(g.Level) == LevelNone {
				continue
			}
			if err := q.DeleteGrant(ctx, store.DeleteGrantParams{CompanionID: c.ID, Section: g.Section}); err != nil {
				return reset, fmt.Errorf("companion: reset grants: delete: %w", err)
			}
			if err := audit(ctx, q, ownerID, ownerID, c.ID, sec, ActionGrantsChanged, now); err != nil {
				return reset, err
			}
			diff[g.Section] = [2]string{g.Level, string(LevelNone)}
		}
		if len(diff) == 0 {
			continue
		}
		reset++
		if languages == nil {
			if languages, err = activeLanguages(ctx, db); err != nil {
				return reset, err
			}
		}
		if err := resetNotice(ctx, q, ownerID, c.ID, reason, diff, languages, nt(now)); err != nil {
			return reset, err
		}
	}
	return reset, nil
}

func isModeScoped(s Section) bool { return slices.Contains(ModeScopedSections, s) }

// activeLanguages are the codes of the `languages` table (the notice carries one title per language); the companion
// copy falls back to English for a language without its own line.
func activeLanguages(ctx context.Context, db store.DBTX) ([]string, error) {
	rows, err := i18nstore.New(db).ListActiveLanguages(ctx)
	if err != nil {
		return nil, fmt.Errorf("companion: reset grants: languages: %w", err)
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Code)
	}
	return out, nil
}

func resetNotice(ctx context.Context, q *store.Queries, ownerID, companionID uint64, reason ResetReason,
	diff map[string][2]string, languages []string, now sql.NullTime) error {
	title, body := map[string]string{}, map[string]string{}
	for _, code := range languages {
		title[code] = T("notices.pregnancy_reset_title", code)
		body[code] = T("notices.pregnancy_reset_body", code)
	}
	rawTitle, err := json.Marshal(title)
	if err != nil {
		return fmt.Errorf("companion: reset notice title: %w", err)
	}
	rawBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("companion: reset notice body: %w", err)
	}
	data, err := json.Marshal(map[string]any{
		"companion_id": companionID, "event": EventPregnancySharingReset, "reason": string(reason), "diff": diff,
	})
	if err != nil {
		return fmt.Errorf("companion: reset notice data: %w", err)
	}
	if err := q.InsertUserNotification(ctx, store.InsertUserNotificationParams{
		UserID: ownerID, Type: NotificationType, Title: json.RawMessage(rawTitle),
		Body: rootdb.NullRawJSON{V: rawBody, Valid: true}, ActionUrl: sql.NullString{String: pathCompanions, Valid: true},
		Data: rootdb.NullRawJSON{V: data, Valid: true}, Now: now,
	}); err != nil {
		return fmt.Errorf("companion: reset notice: %w", err)
	}
	return nil
}
