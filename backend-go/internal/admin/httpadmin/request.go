package httpadmin

import (
	"database/sql"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Pagination defaults: ?page (≥1) and ?per_page (1…MaxPerPage, default DefaultPerPage,
// the Blade panel's paginate(20)).
const (
	DefaultPerPage = 20
	MaxPerPage     = 100
)

// Pagination is the requested page.
type Pagination struct {
	Page    int
	PerPage int
}

// Offset is the SQL offset.
func (p Pagination) Offset() int { return (p.Page - 1) * p.PerPage }

// PageOf reads ?page and ?per_page (invalid values fall back to the defaults).
func PageOf(c fiber.Ctx) Pagination {
	p := Pagination{Page: 1, PerPage: DefaultPerPage}
	if n, err := strconv.Atoi(strings.TrimSpace(c.Query("page"))); err == nil && n >= 1 && n <= 1_000_000 {
		p.Page = n
	}
	if n, err := strconv.Atoi(strings.TrimSpace(c.Query("per_page"))); err == nil && n >= 1 {
		p.PerPage = min(n, MaxPerPage)
	}
	return p
}

// Now is the request clock (X-Test-Now aware, like Carbon::now()).
func Now(c fiber.Ctx) time.Time { return clock.FromContext(c, clock.Real{}).Now() }

// ID parses a numeric route parameter; ok=false means "answer 404".
func ID(c fiber.Ctx, name string) (uint64, bool) {
	s := c.Params(name)
	if s == "" || len(s) > 20 || (len(s) > 1 && s[0] == '0') {
		return 0, false
	}
	n, err := strconv.ParseUint(s, 10, 64)
	return n, err == nil && n > 0
}

// Validate runs Laravel rules over the request input in the request locale (the admin
// chains pin the default language) and returns the validated data or the admin 422.
func Validate(c fiber.Ctx, rules validation.Rules, opts ...validation.Option) (phpval.Map, error) {
	data := validation.Input(c)
	opts = append([]validation.Option{validation.Now(Now(c))}, opts...)
	v := validation.Make(lang.Default(), i18n.Locale(c), data, rules, opts...)
	if v.Fails() {
		return nil, Invalid(v.Errors())
	}
	return v.Validated(), nil
}

// Trans translates a lang key in the request locale (validation.confirmed, …).
func Trans(c fiber.Ctx, key string, params map[string]string) string {
	return lang.Default().Trans(key, params, i18n.Locale(c))
}

// AttributeName is the :attribute for field (validation.attributes.<field>, else the
// field name with underscores as spaces, as Laravel prints it).
func AttributeName(c fiber.Ctx, field string) string {
	if s, ok := lang.Default().Get("validation.attributes."+field, i18n.Locale(c)); ok {
		if str, isStr := s.(string); isStr {
			return str
		}
	}
	return strings.ReplaceAll(field, "_", " ")
}

// String returns data[key] as a string ("" when missing or null).
func String(data phpval.Map, key string) string {
	v, ok := data.Get(key)
	if !ok || v == nil {
		return ""
	}
	return phpval.ToString(v)
}

// Bool returns data[key] as $request->boolean() does (missing = false).
func Bool(data phpval.Map, key string) bool {
	v, ok := data.Get(key)
	if !ok || v == nil {
		return false
	}
	switch x := v.(type) {
	case bool:
		return x
	case string:
		switch strings.ToLower(x) {
		case "1", "true", "on", "yes":
			return true
		}
		return false
	}
	return phpval.Truthy(v)
}

// Audit writes one audit line for an admin mutation: who (admin id), what (action,
// e.g. "user.block"), on what (target type + id) and from where. Never pass secrets.
func Audit(c fiber.Ctx, logger *slog.Logger, action, targetType string, targetID uint64, extra ...slog.Attr) {
	attrs := []slog.Attr{
		slog.String("audit", action),
		slog.String("target_type", targetType),
		slog.Uint64("target_id", targetID),
		slog.String("ip", c.IP()),
	}
	if a := CurrentAdmin(c); a != nil {
		attrs = append(attrs, slog.Uint64("admin_id", a.ID))
	}
	logger.LogAttrs(c.Context(), slog.LevelInfo, "admin audit", append(attrs, extra...)...)
}

// DBTime is a Carbon value as Eloquent writes it: whole seconds, Asia/Tehran wall-clock.
func DBTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t.In(civildate.Tehran).Truncate(time.Second), Valid: true}
}
