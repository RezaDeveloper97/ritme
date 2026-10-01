package search

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Handlers are the /api/v1/search actions. Mount them behind the locale middleware and auth RequireUser.
type Handlers struct {
	svc     *Service
	clock   clock.Clock
	bundles *i18n.TranslationStore
}

// NewHandlers wires the handlers; base is the fallback clock (tests pin it per request).
func NewHandlers(svc *Service, base clock.Clock, bundles *i18n.TranslationStore) *Handlers {
	return &Handlers{svc: svc, clock: base, bundles: bundles}
}

// Search is GET /search?q=&scope=all|mine|education|programs|services&limit=: grouped hits, each with the
// in-app route to open. Framework-style 422 on invalid input. The query is never logged.
func (h *Handlers) Search(c fiber.Ctx) error {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	locale := i18n.Locale(c)
	now := clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)

	input := phpval.NewMap()
	query := validation.Query(c)
	for _, k := range []string{"q", "scope", "limit"} {
		if v, ok := query.Get(k); ok {
			input.Set(k, v)
		}
	}
	v := validation.Make(lang.Default(), locale, input, validation.Rules{
		validation.F("q", "required", "string", "min:"+strconv.Itoa(MinQueryLength), "max:"+strconv.Itoa(MaxQueryLength)),
		validation.F("scope", "nullable", "string", validation.In(Scopes...)),
		validation.F("limit", "nullable", "integer", "min:1", "max:"+strconv.Itoa(MaxLimit)),
	}, validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return v.Errors()
	}

	q := Query{
		UserID:  userID,
		Text:    phpval.ToString(mustGet(input, "q")),
		Scope:   ScopeAll,
		Locale:  locale,
		Default: i18n.LanguagesOf(c).DefaultCode(),
		Today:   civildate.InTehran(now),
	}
	if s, ok := input.Get("scope"); ok && s != nil {
		q.Scope = phpval.ToString(s)
	}
	if l, ok := input.Get("limit"); ok && l != nil {
		q.Limit, _ = strconv.Atoi(phpval.ToString(l))
	}
	if q.Scope == ScopeAll || q.Scope == ScopeMine {
		q.Taxonomy = h.bundles.NamespaceMessages(locale, healthlog.TaxonomyNamespace, q.Default)
	}
	res, err := h.svc.Search(c.Context(), q)
	if err != nil {
		return err
	}
	return httpx.OK(c, res.JSON())
}

func mustGet(m phpval.Map, k string) any {
	v, _ := m.Get(k)
	return v
}
