package teen

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/cycle/resolver"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/teen/store"
)

// CatalogSource reads catalog groups (catalog.Reader: active items in order, cached per group).
type CatalogSource interface {
	Items(ctx context.Context, group string) ([]catalog.Item, error)
}

// Service is the teen domain. Every read and write is scoped to one user id; the parent card reads another
// user's data only through an active parent link and its grants (companion.Service), audited first.
type Service struct {
	conn    companion.Conn
	q       *store.Queries
	catalog CatalogSource
	cycle   *cycleservice.Service
}

// NewService returns a Service on conn (a *sql.DB).
func NewService(conn companion.Conn, cat CatalogSource) *Service {
	return &Service{conn: conn, q: store.New(conn), catalog: cat, cycle: cycleservice.New(conn, nil)}
}

// Profile is the teen's stored answers.
type Profile struct {
	AgeBand    AgeBand
	Menarche   Menarche
	ParentNote string
	UpdatedAt  time.Time
}

func toProfile(r store.TeenProfile) *Profile {
	return &Profile{AgeBand: AgeBand(r.AgeBand), Menarche: Menarche(r.Menarche), ParentNote: r.ParentNote.String,
		UpdatedAt: r.UpdatedAt.Time}
}

// Profile is userID's profile (nil before onboarding).
func (s *Service) Profile(ctx context.Context, userID uint64) (*Profile, error) {
	r, err := s.q.GetTeenProfile(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil //nolint:nilnil // no profile yet is not an error
	}
	if err != nil {
		return nil, fmt.Errorf("teen: profile: %w", err)
	}
	return toProfile(r), nil
}

// SaveProfile stores the onboarding answers (the parent note is kept).
func (s *Service) SaveProfile(ctx context.Context, userID uint64, age AgeBand, m Menarche, now time.Time) (*Profile, error) {
	if err := s.q.UpsertTeenProfile(ctx, store.UpsertTeenProfileParams{
		UserID: userID, AgeBand: string(age), Menarche: string(m), Now: nt(now),
	}); err != nil {
		return nil, fmt.Errorf("teen: save profile: %w", err)
	}
	return s.Profile(ctx, userID)
}

// SetParentNote replaces the note for her parent ("" clears it). ErrProfileRequired before onboarding.
func (s *Service) SetParentNote(ctx context.Context, userID uint64, note string, now time.Time) (*Profile, error) {
	note = strings.TrimSpace(note)
	n, err := s.q.SetTeenParentNote(ctx, store.SetTeenParentNoteParams{
		ParentNote: sql.NullString{String: note, Valid: note != ""}, Now: nt(now), UserID: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("teen: parent note: %w", err)
	}
	if n == 0 {
		return nil, ErrProfileRequired
	}
	return s.Profile(ctx, userID)
}

// LifeMode is userID's stored life-stage mode ("" when none).
func (s *Service) LifeMode(ctx context.Context, userID uint64) (enums.LifeMode, error) {
	m, err := s.q.GetTeenLifeMode(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("teen: life mode: %w", err)
	}
	return enums.LifeMode(m.String), nil
}

// ---------------------------------------------------------------- content

// signMeta is the documented meta of a `teen_signs` item.
type signMeta struct {
	Kind     string   `json:"kind"`
	Menarche []string `json:"menarche"`
	AgeBands []string `json:"age_bands"`
}

func metaOf(it catalog.Item) signMeta {
	var m signMeta
	if len(it.Meta) > 0 {
		_ = json.Unmarshal(it.Meta, &m) // admin JSON; a malformed meta just matches nothing
	}
	return m
}

// matches reports whether an item's answer filters fit the profile (no profile: only unfiltered items).
func (m signMeta) matches(p *Profile) bool {
	if p == nil {
		return len(m.Menarche) == 0 && len(m.AgeBands) == 0
	}
	if len(m.Menarche) > 0 && !slices.Contains(m.Menarche, string(p.Menarche)) {
		return false
	}
	return len(m.AgeBands) == 0 || slices.Contains(m.AgeBands, string(p.AgeBand))
}

func (s *Service) items(ctx context.Context, group string) ([]catalog.Item, error) {
	all, err := s.catalog.Items(ctx, group)
	if err != nil {
		return nil, fmt.Errorf("teen: catalog %s: %w", group, err)
	}
	out := make([]catalog.Item, 0, len(all))
	for _, it := range all {
		if it.For(Audience) {
			out = append(out, it)
		}
	}
	return out, nil
}

// Content is the Teen_Home copy for a profile: the readiness estimate matching her answers, the signs card items,
// the «when to talk» note and the FAQ.
type Content struct {
	Readiness *catalog.Item
	Signs     []catalog.Item
	Talk      *catalog.Item
	FAQ       []catalog.Item
}

// Content picks the catalog copy for p (nil before onboarding: no estimate, only unfiltered signs).
func (s *Service) Content(ctx context.Context, p *Profile) (Content, error) {
	signs, err := s.items(ctx, GroupSigns)
	if err != nil {
		return Content{}, err
	}
	faq, err := s.items(ctx, GroupFAQ)
	if err != nil {
		return Content{}, err
	}
	out := Content{Signs: []catalog.Item{}, FAQ: faq}
	for i := range signs {
		m := metaOf(signs[i])
		switch m.Kind {
		case KindEstimate:
			if out.Readiness == nil && p != nil && m.matches(p) {
				out.Readiness = &signs[i]
			}
		case KindTalk:
			if out.Talk == nil {
				out.Talk = &signs[i]
			}
		case KindSign:
			if m.matches(p) {
				out.Signs = append(out.Signs, signs[i])
			}
		}
	}
	return out, nil
}

// KitItem is one checklist row.
type KitItem struct {
	Item    catalog.Item
	Checked bool
}

// Kit is the school emergency-kit checklist: the active catalog items with her ticks.
type Kit struct {
	Items   []KitItem
	Checked int
}

// Ready reports whether every active kit item is checked (false for an empty list).
func (k Kit) Ready() bool { return len(k.Items) > 0 && k.Checked == len(k.Items) }

// Kit is userID's checklist. Ticks of items an admin deactivated are ignored.
func (s *Service) Kit(ctx context.Context, userID uint64) (Kit, error) {
	items, err := s.items(ctx, GroupKit)
	if err != nil {
		return Kit{}, err
	}
	checks, err := s.q.ListTeenKitChecks(ctx, userID)
	if err != nil {
		return Kit{}, fmt.Errorf("teen: kit: %w", err)
	}
	k := Kit{Items: make([]KitItem, 0, len(items))}
	for _, it := range items {
		c := slices.Contains(checks, it.Code)
		if c {
			k.Checked++
		}
		k.Items = append(k.Items, KitItem{Item: it, Checked: c})
	}
	return k, nil
}

// SetKitItem ticks or unticks one active kit item. ErrUnknownKitItem for any other code.
func (s *Service) SetKitItem(ctx context.Context, userID uint64, code string, checked bool, now time.Time) (Kit, error) {
	items, err := s.items(ctx, GroupKit)
	if err != nil {
		return Kit{}, err
	}
	if !slices.ContainsFunc(items, func(it catalog.Item) bool { return it.Code == code }) {
		return Kit{}, ErrUnknownKitItem
	}
	if checked {
		err = s.q.CheckTeenKitItem(ctx, store.CheckTeenKitItemParams{UserID: userID, ItemCode: code, Now: nt(now)})
	} else {
		err = s.q.UncheckTeenKitItem(ctx, store.UncheckTeenKitItemParams{UserID: userID, ItemCode: code})
	}
	if err != nil {
		return Kit{}, fmt.Errorf("teen: kit item: %w", err)
	}
	return s.Kit(ctx, userID)
}

// ---------------------------------------------------------------- next period as a week

// NextPeriodWeek buckets userID's predicted next period start into this / next / later week (Saturday-start weeks,
// Tehran). unknown before the first period, without period data, or without a prediction. An overdue prediction is
// rolled forward by the cycle length (WeekBucket), so the bucket never reveals lateness.
func (s *Service) NextPeriodWeek(ctx context.Context, userID uint64, p *Profile, today civildate.Date) (PeriodWeek, error) {
	if p == nil || p.Menarche == MenarcheNotYet {
		return WeekUnknown, nil
	}
	sn, err := s.cycle.Load(ctx, userID, today, today, today)
	if err != nil {
		return "", fmt.Errorf("teen: cycle: %w", err)
	}
	if len(sn.Histories) == 0 && sn.Profile == nil {
		return WeekUnknown, nil
	}
	profile := sn.EngineProfile()
	st := resolver.Resolve(sn.Histories, profile, today, today, metrics.Calculate(sn.Histories, profile))
	return WeekBucket(st.PredictedNextPeriodStart, st.EffectiveCycleLength, today), nil
}

// OverdueGraceDays is how long an overdue prediction may still show as «this week» (an ordinary few-days variation).
const OverdueGraceDays = 3

// WeekBucket is the week of the predicted start next relative to today (zero next = unknown). Lateness is never
// revealed: up to OverdueGraceDays overdue counts as today; beyond that the prediction is rolled forward by
// cycleLen days until it is ≥ today (no usable cycle length → unknown), so a late period does not stay «this week».
func WeekBucket(next civildate.Date, cycleLen int, today civildate.Date) PeriodWeek {
	if next.IsZero() {
		return WeekUnknown
	}
	if next.Before(today) {
		switch {
		case next.DiffDays(today) <= OverdueGraceDays: // days overdue
			next = today
		case cycleLen <= 0:
			return WeekUnknown
		default:
			for next.Before(today) {
				next = next.AddDays(cycleLen)
			}
		}
	}
	start := today.StartOfWeek()
	switch {
	case next.Before(start.AddDays(7)):
		return WeekThis
	case next.Before(start.AddDays(14)):
		return WeekNext
	}
	return WeekLater
}

// ---------------------------------------------------------------- teen home

// Today is the Teen_Home read model.
type Today struct {
	Profile  *Profile
	TeenMode bool
	Allows   Allows
	Content  Content
	Kit      Kit
	// Preview is what a parent would see with every teen section granted (the live preview card of Teen_Parent);
	// the client masks it with the link's current grants.
	Preview ParentView
	// ParentLinks are her parent links (invited or active).
	ParentLinks []companion.Link
}

// Today builds the teen home for userID.
func (s *Service) Today(ctx context.Context, userID uint64, clk clock.Clock) (Today, error) {
	now := clk.Now()
	today := civildate.InTehran(now)
	p, err := s.Profile(ctx, userID)
	if err != nil {
		return Today{}, err
	}
	mode, err := s.LifeMode(ctx, userID)
	if err != nil {
		return Today{}, err
	}
	content, err := s.Content(ctx, p)
	if err != nil {
		return Today{}, err
	}
	kit, err := s.Kit(ctx, userID)
	if err != nil {
		return Today{}, err
	}
	week, err := s.NextPeriodWeek(ctx, userID, p, today)
	if err != nil {
		return Today{}, err
	}
	ready := kit.Ready()
	preview := ParentView{PeriodWeek: &week, KitReady: &ready}
	if p != nil && p.ParentNote != "" {
		note := p.ParentNote
		preview.Note = &note
	}
	links, err := companion.NewService(s.conn, clk, companion.Options{}).ListForOwner(ctx, userID)
	if err != nil {
		return Today{}, err
	}
	parents := make([]companion.Link, 0, len(links))
	for _, l := range links {
		if l.Type == companion.TypeParent {
			parents = append(parents, l)
		}
	}
	return Today{Profile: p, TeenMode: mode == enums.LifeModeTeen, Allows: AllowsFor(mode), Content: content, Kit: kit,
		Preview: preview, ParentLinks: parents}, nil
}

// ---------------------------------------------------------------- parent card

// ParentView is the card content; a nil part was not granted (and was never read).
type ParentView struct {
	PeriodWeek *PeriodWeek
	KitReady   *bool
	Note       *string // also nil when granted but empty
}

// ParentCard is one active parent link as the parent sees it.
type ParentCard struct {
	Link     companion.Link
	TeenName string
	View     ParentView
}

// ParentCards are the read-only cards of every active parent link in which viewerID is the parent and whose owner is
// still in teen mode. Each granted teen
// section is audited (companion read) before its part is built; an ungranted part is never read. A failed audit
// fails the request (no audit row → no access).
func (s *Service) ParentCards(ctx context.Context, viewerID uint64, clk clock.Clock) ([]ParentCard, error) {
	svc := companion.NewService(s.conn, clk, companion.Options{})
	links, err := svc.ListForCompanion(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	today := civildate.InTehran(clk.Now())
	cards := []ParentCard{}
	ownerIDs := []uint64{}
	for _, l := range links {
		if l.Type != companion.TypeParent {
			continue
		}
		// The teen sections exist only for a teen: once the owner left teen mode the card is gone (nothing read,
		// nothing audited). The link itself stays; ending it is the owner's choice.
		mode, err := s.LifeMode(ctx, l.OwnerID)
		if err != nil {
			return nil, err
		}
		if mode != enums.LifeModeTeen {
			continue
		}
		card := ParentCard{Link: l}
		granted, err := s.authorizeParent(ctx, svc, l, viewerID)
		if err != nil {
			return nil, err
		}
		var profile *Profile
		if len(granted) > 0 {
			if profile, err = s.Profile(ctx, l.OwnerID); err != nil {
				return nil, err
			}
		}
		for _, sec := range granted {
			switch sec {
			case companion.SectionTeenPeriodWeek:
				w, err := s.NextPeriodWeek(ctx, l.OwnerID, profile, today)
				if err != nil {
					return nil, err
				}
				card.View.PeriodWeek = &w
			case companion.SectionTeenKit:
				k, err := s.Kit(ctx, l.OwnerID)
				if err != nil {
					return nil, err
				}
				r := k.Ready()
				card.View.KitReady = &r
			case companion.SectionTeenNotes:
				if profile != nil && profile.ParentNote != "" {
					n := profile.ParentNote
					card.View.Note = &n
				}
			}
		}
		cards = append(cards, card)
		ownerIDs = append(ownerIDs, l.OwnerID)
	}
	names, err := svc.Names(ctx, ownerIDs...)
	if err != nil {
		return nil, err
	}
	for i := range cards {
		cards[i].TeenName = names[cards[i].Link.OwnerID]
	}
	return cards, nil
}

// authorizeParent is the teen sections viewerID may read through link l right now (the live grant, re-read, not the
// one loaded with the link), each audited before the caller reads anything.
func (s *Service) authorizeParent(ctx context.Context, svc *companion.Service, l companion.Link, viewerID uint64) ([]companion.Section, error) {
	granted := []companion.Section{}
	for _, sec := range companion.TeenSections {
		level, err := svc.Level(ctx, l.OwnerID, viewerID, sec)
		if err != nil {
			return nil, err
		}
		if !level.CanRead() {
			continue
		}
		if err := svc.Audit(ctx, l.OwnerID, viewerID, l.ID, sec, companion.ActionRead); err != nil {
			return nil, err
		}
		granted = append(granted, sec)
	}
	return granted, nil
}

func nt(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t.In(civildate.Tehran).Truncate(time.Second), Valid: true}
}
