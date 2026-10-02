// Package companionhome is the companion panel home (bloom B-N4-03, canvas nbl_Hamdam_Home): GET /api/v1/companion/home
// for a male (companion) account. Per active link it shows what the owner shared — her cycle day / phase / next
// period, pregnancy, symptoms, medications and appointments, each only with a view or edit grant and each read
// audited (companion.Service.Audit) before it is built through the access-filtered section views of
// internal/companion/shared — plus «امروز چه کار کنی؟» tips for her phase (admin content, tips.go), reading
// suggestions and the shared child card (null until bloom B-N5). A companion without links gets an empty state that
// points to «کد همدم را وارد کن»; a woman's account gets 403 not_companion_account.
package companionhome

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/companion/store"
	"github.com/ritme/backend-go/internal/content"
	"github.com/ritme/backend-go/internal/content/sanitizer"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// ActionEnterCode is the empty state's action: the companion code entry (B-N4-05).
const ActionEnterCode = "enter_code"

// homeSections are the sections the home reads, in card order (the artboard's shared block).
var homeSections = []companion.Section{
	companion.SectionCycle, companion.SectionPregnancy, companion.SectionSymptoms,
	companion.SectionMeds, companion.SectionAppointments,
}

// Options wires the handlers.
type Options struct {
	Reader companion.SectionReader // internal/companion/shared
	AppURL string                  // APP_URL (article image URLs)
	Logger *slog.Logger
}

// Handlers serve GET /companion/home. Mount behind the locale middleware and auth RequireUser.
type Handlers struct {
	conn     companion.Conn
	base     clock.Clock
	reader   companion.SectionReader
	appURL   string
	accounts *companion.Accounts
	logger   *slog.Logger
}

// NewHandlers wires the handlers; base is the fallback clock (tests and the contract suite pin it per request).
func NewHandlers(conn companion.Conn, base clock.Clock, opt Options) *Handlers {
	h := &Handlers{conn: conn, base: base, reader: opt.Reader, appURL: opt.AppURL,
		accounts: companion.NewAccounts(conn), logger: opt.Logger}
	if h.logger == nil {
		h.logger = slog.Default()
	}
	return h
}

// Show is GET /companion/home.
func (h *Handlers) Show(c fiber.Ctx) error {
	u := auth.CurrentUser(c)
	if u == nil {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	locale := i18n.Locale(c)
	ctx := c.Context()
	ok, err := h.accounts.IsCompanion(ctx, u.ID)
	if err != nil {
		return err
	}
	if !ok {
		return companion.NonCompanionForbidden(locale)
	}
	clk := clock.FromContext(c, h.base)
	data, err := h.build(ctx, u, locale, i18n.LanguagesOf(c).DefaultCode(), clk)
	if err != nil {
		return err
	}
	return httpx.OK(c, data)
}

func (h *Handlers) build(ctx context.Context, u *auth.User, locale, deflt string, clk clock.Clock) (*jsonx.OrderedMap, error) {
	now := clk.Now()
	svc := companion.NewService(h.conn, clk, companion.Options{})
	q := store.New(h.conn)
	links, err := svc.ListForCompanion(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	ownerIDs := make([]uint64, 0, len(links))
	for _, l := range links {
		ownerIDs = append(ownerIDs, l.OwnerID)
	}
	names, err := svc.Names(ctx, ownerIDs...)
	if err != nil {
		return nil, err
	}
	tc, err := loadTipCopy(ctx, q, locale, deflt)
	if err != nil {
		return nil, err
	}

	partners := make([]*jsonx.OrderedMap, 0, len(links))
	articlePhase := ""
	for _, l := range links {
		p, phase, err := h.partner(ctx, svc, tc, l, u.ID, names, locale, now)
		if err != nil {
			return nil, err
		}
		if articlePhase == "" && phase != PhaseGeneral {
			articlePhase = phase
		}
		partners = append(partners, p)
	}
	articles, err := h.articles(ctx, q, articlePhase, locale, deflt)
	if err != nil {
		return nil, err
	}

	var empty any
	if len(links) == 0 {
		empty = jsonx.Obj(
			"title", T("empty.title", locale),
			"body", T("empty.body", locale),
			"action", ActionEnterCode,
			"action_label", T("empty.action", locale),
		)
	}
	var viewerName any
	if u.Name.Valid && strings.TrimSpace(u.Name.String) != "" {
		viewerName = strings.TrimSpace(u.Name.String)
	}
	return jsonx.Obj(
		"mode", string(enums.LifeModeCompanion),
		"viewer", jsonx.Obj("id", u.ID, "name", viewerName),
		"has_partners", len(links) > 0,
		"partners", partners,
		"empty_state", empty,
		"articles", articles,
	), nil
}

// partner is one link's card set and its tip phase. A section is read only with a view/edit grant on the link
// (loaded in the same request), and the read is audited before the view is built.
func (h *Handlers) partner(ctx context.Context, svc *companion.Service, tc *tipCopy, l companion.Link, viewerID uint64,
	names map[uint64]string, locale string, now time.Time) (*jsonx.OrderedMap, string, error) {
	views := map[companion.Section]any{}
	for _, s := range homeSections {
		if !l.Grants.Of(s).CanRead() {
			continue
		}
		if err := svc.Audit(ctx, l.OwnerID, viewerID, l.ID, s, companion.ActionRead); err != nil {
			return nil, "", err
		}
		v, err := h.reader.Read(ctx, l.OwnerID, s, locale, now)
		if err != nil {
			return nil, "", err
		}
		views[s] = v
	}
	phase := tipPhase(views)
	name := names[l.OwnerID]
	shownName := name
	if shownName == "" {
		shownName = T("someone", locale)
	}
	tips := make([]*jsonx.OrderedMap, 0, TipsPerPhase)
	for _, t := range tc.tips(phase, shownName) {
		tips = append(tips, jsonx.Obj("key", t.Key, "title", t.Title, "body", nullable(t.Body)))
	}
	return jsonx.Obj(
		"link", companion.ViewerLinkJSON(l, names),
		"partner_name", nullable(name),
		"cycle", views[companion.SectionCycle],
		"pregnancy", views[companion.SectionPregnancy],
		"symptoms", views[companion.SectionSymptoms],
		"meds", views[companion.SectionMeds],
		"appointments", views[companion.SectionAppointments],
		"phase", phase,
		"note", nullable(tc.note(phase, shownName)),
		"tips", tips,
		"child", nil, // shared child card: bloom B-N5 (children)
	), phase, nil
}

// tipPhase is pregnancy when a shared pregnancy is active, else the shared cycle's phase, else general.
func tipPhase(views map[companion.Section]any) string {
	if m, ok := views[companion.SectionPregnancy].(*jsonx.OrderedMap); ok {
		if active, _ := m.Get("is_active"); active == true {
			return PhasePregnancy
		}
	}
	if m, ok := views[companion.SectionCycle].(*jsonx.OrderedMap); ok {
		if hasData, _ := m.Get("has_data"); hasData != true {
			return PhaseGeneral
		}
		if v, _ := m.Get("main_phase"); v != nil {
			return PhaseOf(enums.MainPhase(fmt.Sprint(v)))
		}
	}
	return PhaseGeneral
}

// articles are up to 4 published articles: category companion first, then ones tagged with the phase (the legacy
// cycle phase the articles admin tags with).
func (h *Handlers) articles(ctx context.Context, q *store.Queries, phase, locale, deflt string) ([]*jsonx.OrderedMap, error) {
	var tags []string
	if lp, ok := legacyPhase(phase); ok {
		tags = []string{lp}
	}
	raw, err := json.Marshal(append([]string{}, tags...))
	if err != nil {
		return nil, err
	}
	hasPhases := int64(0)
	if len(tags) > 0 {
		hasPhases = 1
	}
	rows, err := q.ListCompanionArticles(ctx, store.ListCompanionArticlesParams{HasPhases: hasPhases, Phases: string(raw)})
	if err != nil {
		return nil, fmt.Errorf("companion home: articles: %w", err)
	}
	out := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, a := range rows {
		var excerpt, image, category, minutes any
		if a.Excerpt.Valid {
			if s := pickString(a.Excerpt.V, locale, deflt); s != "" {
				if txt, ok := sanitizer.PlainText(s); ok {
					excerpt = txt
				}
			}
		}
		switch {
		case a.ImagePath.Valid && a.ImagePath.String != "":
			image = content.PublicURL(h.appURL, a.ImagePath.String)
		case a.ImageUrl.Valid && a.ImageUrl.String != "":
			image = a.ImageUrl.String
		}
		if a.Category.Valid {
			category = a.Category.String
		}
		if a.ReadTimeMinutes.Valid {
			minutes = int(a.ReadTimeMinutes.Int16)
		}
		out = append(out, jsonx.Obj(
			"id", a.ID,
			"slug", a.Slug,
			"title", nullable(pickString(a.Title, locale, deflt)),
			"excerpt", excerpt,
			"read_time_minutes", minutes,
			"image_url", image,
			"category", category,
		))
	}
	return out, nil
}

// legacyPhase is the articles' cycle_phases tag of a tip phase (enums.CyclePhase values); none for pregnancy and
// general.
func legacyPhase(phase string) (string, bool) {
	switch phase {
	case PhaseMenstrual:
		return string(enums.CyclePhaseMenstruation), true
	case PhaseFollicular:
		return string(enums.CyclePhaseFollicular), true
	case PhaseFertile:
		return string(enums.CyclePhaseOvulation), true
	case PhaseLuteal:
		return string(enums.CyclePhaseLuteal), true
	}
	return "", false
}

func pickString(raw json.RawMessage, locale, deflt string) string {
	var s string
	if json.Unmarshal(i18n.Pick(raw, locale, deflt), &s) != nil {
		return ""
	}
	return strings.TrimSpace(s)
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
