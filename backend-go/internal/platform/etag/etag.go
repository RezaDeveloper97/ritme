// Package etag is the ETag / 304 helper of the engine GETs (T-M9-31, D-132, docs/engine-v6/README.md §3).
//
// An engine response is a pure function of the user's engine revision, her local day, the locale, the engine params
// and the config snapshots it reads, and the binary that computed it. The tag names all of them:
//
//	W/"u{uid36}.e{rev}-{localToday}-{locale}-{paramsFP}-{cfgFP}-{build}"
//
// The user id is part of the tag, so two accounts on one browser can never revalidate each other's cached body. Every
// input is known right after the auth query (the revision rides in it, T-M9-31), so a revalidation is answered with a
// 304 before the handler does any other work: one query. Responses carry `Cache-Control: private, no-cache` (the
// browser stores the body but asks every time) and `Vary: Authorization, Accept-Language`.
package etag

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// CacheControl is the Cache-Control of every tagged engine response.
const CacheControl = "private, no-cache"

// Engine is the input of one engine ETag.
type Engine struct {
	UserID   uint64
	Rev      uint64         // user_revisions.engine_rev from the auth query (0 without a row)
	Today    civildate.Date // the user-local day (civildate.LocalToday)
	Locale   string         // the resolved response locale
	ParamsFP string         // engineparams.Fingerprint(Today)
	ConfigFP string         // configcache.Fingerprint(tables the handler reads)
	Build    string         // Build()
}

// Tag is the weak entity tag of e.
func (e Engine) Tag() string {
	var b strings.Builder
	b.Grow(64)
	b.WriteString(`W/"u`)
	b.WriteString(strconv.FormatUint(e.UserID, 36))
	b.WriteString(".e")
	b.WriteString(strconv.FormatUint(e.Rev, 10))
	for _, part := range []string{e.Today.String(), e.Locale, e.ParamsFP, e.ConfigFP, e.Build} {
		b.WriteByte('-')
		writeSafe(&b, part)
	}
	b.WriteByte('"')
	return b.String()
}

// writeSafe writes s with every byte that may not appear inside an entity tag (RFC 9110 etagc: %x21 / %x23-7E) as
// '_', so a client-chosen locale can never break the header.
func writeSafe(b *strings.Builder, s string) {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 0x21 || c == '"' || c > 0x7e {
			c = '_'
		}
		b.WriteByte(c)
	}
}

// Match reports whether an If-None-Match header value matches tag with the weak comparison of RFC 9110 §13.1.2:
// "*" or any listed tag whose opaque part equals tag's (W/ prefixes ignored).
func Match(ifNoneMatch, tag string) bool {
	want := opaque(tag)
	if want == "" {
		return false
	}
	for _, cand := range strings.Split(ifNoneMatch, ",") {
		cand = strings.TrimSpace(cand)
		if cand == "*" || (cand != "" && opaque(cand) == want) {
			return true
		}
	}
	return false
}

// opaque is the quoted part of an entity tag without the weak prefix; "" when t is not a tag.
func opaque(t string) string {
	t = strings.TrimPrefix(t, "W/")
	if len(t) < 2 || t[0] != '"' || t[len(t)-1] != '"' {
		return ""
	}
	return t
}

// SetHeaders adds the ETag, Cache-Control and Vary headers of a tagged engine response.
func SetHeaders(c fiber.Ctx, tag string) {
	c.Set(fiber.HeaderETag, tag)
	c.Set(fiber.HeaderCacheControl, CacheControl)
	c.Vary(fiber.HeaderAuthorization, fiber.HeaderAcceptLanguage)
}

// NotModified sets the engine headers and, when the request's If-None-Match matches tag, answers 304 with no body
// and returns true: the handler then returns nil at once. Call it first in an engine GET handler, before any other
// work.
func NotModified(c fiber.Ctx, tag string) bool {
	SetHeaders(c, tag)
	inm := c.Get(fiber.HeaderIfNoneMatch)
	if inm == "" || !Match(inm, tag) {
		return false
	}
	c.Status(fiber.StatusNotModified)
	c.Response().ResetBody()
	return true
}

// Middleware tags a route: tag computes the request's ETag (from data already on the context — the auth locals, the
// request clock, the locale); a matching If-None-Match ends the request with 304, anything else continues to the
// handler with the headers already set. A tag error is returned as is (500), never turned into a 304.
func Middleware(tag func(c fiber.Ctx) (string, error)) fiber.Handler {
	return func(c fiber.Ctx) error {
		t, err := tag(c)
		if err != nil {
			return err
		}
		if NotModified(c, t) {
			return nil
		}
		return c.Next()
	}
}

// Build identifies the running binary for the {build} part of the tag and the engine state cache key: a new deploy
// changes every tag, so a client never keeps a body computed by older code. It is the VCS revision stamped by the Go
// toolchain for a clean tree, otherwise (no VCS stamp, or a modified tree such as a beta build of the working tree)
// the first bytes of the executable's SHA-256, otherwise "dev". Computed once; call it at start-up to keep the
// hashing off the first request.
var Build = sync.OnceValue(computeBuild)

func computeBuild() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		var rev string
		var dirty bool
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				rev = s.Value
			case "vcs.modified":
				dirty = s.Value == "true"
			}
		}
		if rev != "" && !dirty {
			if len(rev) > 12 {
				rev = rev[:12]
			}
			return rev
		}
	}
	if exe, err := os.Executable(); err == nil {
		if f, err := os.Open(exe); err == nil { //nolint:gosec // G304: our own executable
			defer func() { _ = f.Close() }()
			h := sha256.New()
			if _, err := io.Copy(h, f); err == nil {
				return hex.EncodeToString(h.Sum(nil)[:6])
			}
		}
	}
	return "dev"
}
