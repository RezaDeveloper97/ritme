package content

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/content/sanitizer"
	"github.com/ritme/backend-go/internal/content/store"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

const (
	articlesDefaultPerPage = 12
	articlesMaxPerPage     = 50
)

var articleIndexRules = validation.Rules{
	validation.F("page", "nullable", "integer", "min:1"),
	validation.F("per_page", "nullable", "integer", "min:1", fmt.Sprintf("max:%d", articlesMaxPerPage)),
	validation.F("category", "nullable", "string", "max:255"),
	validation.F("q", "nullable", "string", "max:100"),
}

// articleRow is the column set every article query returns (body only for show).
type articleRow struct {
	ID              uint64
	Slug            string
	Title           json.RawMessage
	Excerpt         db.NullRawJSON
	Body            db.NullRawJSON
	CyclePhases     db.NullRawJSON
	Category        sql.NullString
	ReadTimeMinutes sql.NullInt16
	ImageURL        sql.NullString
	ImagePath       sql.NullString
	PublishedAt     sql.NullTime
}

// Articles is GET /articles (ArticleController::index).
func (h *Handlers) Articles(c fiber.Ctx) error {
	data := validation.Input(c)
	v := validation.Make(lang.Default(), i18n.Locale(c), data, articleIndexRules, validation.Now(h.now(c)))
	if v.Fails() {
		return v.Errors() // framework 422 ($request->validate)
	}
	filters := v.Validated()
	locale := i18n.ResolveLocale(c, "")

	perPage := articlesDefaultPerPage
	if pp, _ := filters.Get("per_page"); pp != nil {
		perPage = int(phpval.ToFloat(pp))
	}
	page := httpx.PageParam(c)

	var category string
	hasCategory := false
	if cat, _ := filters.Get("category"); cat != nil {
		category = phpval.ToString(cat)
		hasCategory = category != ""
	}
	var term string
	if q, _ := filters.Get("q"); q != nil {
		term = strings.Trim(phpval.ToString(q), " \t\n\r\x00\x0B")
	}
	// array_unique([$locale, 'fa']): the reader's locale plus the always-present fa copy.
	args := store.CountPublishedArticlesParams{
		HasCategory: b2i(hasCategory), Category: sql.NullString{String: category, Valid: true},
		HasTerm: b2i(term != ""), LangA: locale, LangB: "fa", Pattern: "%" + term + "%",
	}
	total, err := h.q.CountPublishedArticles(c, args)
	if err != nil {
		return fmt.Errorf("content: count articles: %w", err)
	}

	items := []*jsonx.OrderedMap{}
	offset := (int64(page) - 1) * int64(perPage)
	if total > 0 && offset < total && offset <= math.MaxInt32 {
		rows, err := h.q.ListPublishedArticles(c, store.ListPublishedArticlesParams{
			HasCategory: args.HasCategory, Category: args.Category, HasTerm: args.HasTerm,
			LangA: args.LangA, LangB: args.LangB, Pattern: args.Pattern,
			Limit: int32(perPage), Offset: int32(offset), //nolint:gosec // G115: bounded above
		})
		if err != nil {
			return fmt.Errorf("content: list articles: %w", err)
		}
		for _, r := range rows {
			items = append(items, h.summary(c, articleRow{ID: r.ID, Slug: r.Slug, Title: r.Title, Excerpt: r.Excerpt,
				CyclePhases: r.CyclePhases, Category: r.Category, ReadTimeMinutes: r.ReadTimeMinutes,
				ImageURL: r.ImageUrl, ImagePath: r.ImagePath, PublishedAt: r.PublishedAt}, locale))
		}
	}

	cats, err := h.q.ListPublishedArticleCategories(c)
	if err != nil {
		return fmt.Errorf("content: article categories: %w", err)
	}
	categories := make([]string, 0, len(cats))
	for _, cat := range cats {
		categories = append(categories, cat.String)
	}

	p := httpx.Page{Total: int(total), PerPage: perPage, CurrentPage: page}
	return httpx.OK(c, jsonx.Obj("items", jsonx.List(items), "categories", jsonx.List(categories), "meta", p.Meta()))
}

// Article is GET /articles/{slug} (ArticleController::show).
func (h *Handlers) Article(c fiber.Ctx) error {
	locale := i18n.ResolveLocale(c, "")
	row, err := h.q.GetPublishedArticleBySlug(c, c.Params("slug"))
	if errors.Is(err, sql.ErrNoRows) {
		return httpx.Fail(fiber.StatusNotFound, pick(locale == "fa", "مقاله یافت نشد.", "Article not found."))
	}
	if err != nil {
		return fmt.Errorf("content: article: %w", err)
	}
	a := articleRow{ID: row.ID, Slug: row.Slug, Title: row.Title, Excerpt: row.Excerpt, Body: row.Body,
		CyclePhases: row.CyclePhases, Category: row.Category, ReadTimeMinutes: row.ReadTimeMinutes,
		ImageURL: row.ImageUrl, ImagePath: row.ImagePath, PublishedAt: row.PublishedAt}

	detail := h.summary(c, a, locale)
	var body any
	if s, ok := pickString(nullJSON(a.Body), locale, i18n.LanguagesOf(c).DefaultCode()); ok && a.Body.Valid {
		if clean, ok := sanitizer.Clean(s); ok {
			body = clean
		}
	}
	detail.Set("body", body)

	phases := phaseList(a.CyclePhases)
	phasesJSON, _ := json.Marshal(phases)
	related, err := h.q.ListRelatedArticles(c, store.ListRelatedArticlesParams{
		ID:          a.ID,
		HasCategory: b2i(a.Category.Valid && a.Category.String != ""),
		Category:    a.Category,
		HasPhases:   b2i(len(phases) > 0),
		Phases:      string(phasesJSON),
	})
	if err != nil {
		return fmt.Errorf("content: related articles: %w", err)
	}
	rel := make([]*jsonx.OrderedMap, len(related))
	for i, r := range related {
		rel[i] = h.summary(c, articleRow{ID: r.ID, Slug: r.Slug, Title: r.Title, Excerpt: r.Excerpt,
			CyclePhases: r.CyclePhases, Category: r.Category, ReadTimeMinutes: r.ReadTimeMinutes,
			ImageURL: r.ImageUrl, ImagePath: r.ImagePath, PublishedAt: r.PublishedAt}, locale)
	}
	return httpx.OK(c, jsonx.Obj("article", detail, "related", jsonx.List(rel)))
}

// summary is ArticleController::summary (the card shape).
func (h *Handlers) summary(c fiber.Ctx, a articleRow, locale string) *jsonx.OrderedMap {
	def := i18n.LanguagesOf(c).DefaultCode()
	var excerpt any
	if s, ok := pickString(nullJSON(a.Excerpt), locale, def); ok && a.Excerpt.Valid {
		if txt, ok := sanitizer.PlainText(s); ok {
			excerpt = txt
		}
	}
	// image_url accessor: an uploaded image_path wins over the stored external URL.
	var imageURL any
	switch {
	case a.ImagePath.Valid && phpTruthy(a.ImagePath.String):
		imageURL = PublicURL(h.appURL, a.ImagePath.String)
	case a.ImageURL.Valid:
		imageURL = a.ImageURL.String
	}
	var readTime any
	if a.ReadTimeMinutes.Valid {
		readTime = a.ReadTimeMinutes.Int16
	}
	var phases any = []any{}
	if a.CyclePhases.Valid {
		if v := phpValue(a.CyclePhases.V); v != nil {
			phases = v
		}
	}
	var published any
	if a.PublishedAt.Valid {
		published = civildate.InTehran(a.PublishedAt.Time).String()
	}
	return jsonx.Obj(
		"id", a.ID,
		"slug", a.Slug,
		"title", phpValue(i18n.Pick(a.Title, locale, def)),
		"excerpt", excerpt,
		"image_url", imageURL,
		"read_time_minutes", readTime,
		"category", nullString(a.Category),
		"cycle_phases", phases,
		"published_at", published,
	)
}

// phaseList is `$article->cycle_phases ?? []` as the foreach in related() walks it:
// the values of a list/object; a scalar yields nothing.
func phaseList(col db.NullRawJSON) []any {
	out := []any{}
	if !col.Valid {
		return out
	}
	v, err := phpval.Decode(col.V)
	if err != nil {
		return out
	}
	_, vals := phpval.Entries(v)
	for _, x := range vals {
		out = append(out, phpval.Packed(x))
	}
	return out
}

func b2i(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
