package httpx

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Paginator labels: lang/en/pagination.php. No other locale ships pagination.php, so
// the fallback locale (en) answers for all of them.
const (
	labelPrevious = "&laquo; Previous"
	labelNext     = "Next &raquo;"
	onEachSide    = 3 // AbstractPaginator::$onEachSide
)

// Page is one page of a LengthAwarePaginator (Builder::paginate($perPage)).
type Page struct {
	Items       any    // the page's rows (a slice); nil encodes as []
	Total       int    // total rows over all pages
	PerPage     int    // page size
	CurrentPage int    // requested page (≥ 1; may exceed LastPage)
	Path        string // absolute URL without query: $request->url()
	Count       int    // number of Items (from/to are null when 0)
}

// NewPage builds a Page from the current page's rows.
func NewPage[T any](items []T, total, perPage, currentPage int, path string) Page {
	return Page{Items: jsonx.List(items), Total: total, PerPage: perPage, CurrentPage: currentPage,
		Path: path, Count: len(items)}
}

// LastPage is max((int) ceil(total / perPage), 1).
func (p Page) LastPage() int {
	if p.PerPage <= 0 {
		return 1
	}
	last := (p.Total + p.PerPage - 1) / p.PerPage
	if last < 1 {
		return 1
	}
	return last
}

// Offset is the SQL offset of the current page.
func (p Page) Offset() int { return (p.CurrentPage - 1) * p.PerPage }

func (p Page) url(page int) string {
	if page <= 0 {
		page = 1
	}
	sep := "?"
	if strings.Contains(p.Path, "?") {
		sep = "&"
	}
	return p.Path + sep + "page=" + strconv.Itoa(page)
}

func (p Page) urlOrNil(page int, ok bool) any {
	if !ok {
		return nil
	}
	return p.url(page)
}

// Raw is LengthAwarePaginator::toArray() — the whole paginator as `data`
// (GET /health-logs): current_page, data, first_page_url, from, last_page,
// last_page_url, links, next_page_url, path, per_page, prev_page_url, to, total.
func (p Page) Raw() *jsonx.OrderedMap {
	last := p.LastPage()
	var from, to any
	if p.Count > 0 {
		first := p.Offset() + 1
		from, to = first, first+p.Count-1
	}
	hasPrev := p.CurrentPage > 1
	hasMore := p.CurrentPage < last
	items := p.Items
	if items == nil {
		items = []any{}
	}
	return jsonx.Obj(
		"current_page", p.CurrentPage,
		"data", items,
		"first_page_url", p.url(1),
		"from", from,
		"last_page", last,
		"last_page_url", p.url(last),
		"links", p.links(),
		"next_page_url", p.urlOrNil(p.CurrentPage+1, hasMore),
		"path", p.Path,
		"per_page", p.PerPage,
		"prev_page_url", p.urlOrNil(p.CurrentPage-1, hasPrev),
		"to", to,
		"total", p.Total,
	)
}

// Meta is the 4-key summary the controllers build by hand — data.pagination
// (notifications) and data.meta (articles): current_page, last_page, per_page, total.
func (p Page) Meta() *jsonx.OrderedMap {
	return jsonx.Obj("current_page", p.CurrentPage, "last_page", p.LastPage(),
		"per_page", p.PerPage, "total", p.Total)
}

// links is LengthAwarePaginator::linkCollection() over UrlWindow::make().
func (p Page) links() []*jsonx.OrderedMap {
	last := p.LastPage()
	hasPrev := p.CurrentPage > 1
	hasMore := p.CurrentPage < last

	var prevPage, nextPage any
	if hasPrev {
		prevPage = p.CurrentPage - 1
	}
	if hasMore {
		nextPage = p.CurrentPage + 1
	}
	out := []*jsonx.OrderedMap{jsonx.Obj(
		"url", p.urlOrNil(p.CurrentPage-1, hasPrev), "label", labelPrevious, "page", prevPage, "active", false)}
	for _, el := range p.elements() {
		if el == nil { // the "..." separator
			out = append(out, jsonx.Obj("url", nil, "label", "...", "active", false))
			continue
		}
		for _, n := range el {
			out = append(out, jsonx.Obj(
				"url", p.url(n), "label", strconv.Itoa(n), "page", n, "active", n == p.CurrentPage))
		}
	}
	return append(out, jsonx.Obj(
		"url", p.urlOrNil(p.CurrentPage+1, hasMore), "label", labelNext, "page", nextPage, "active", false))
}

// elements is LengthAwarePaginator::elements(): page ranges with nil for "...".
func (p Page) elements() [][]int {
	last := p.LastPage()
	rng := func(from, to int) []int {
		var r []int
		for i := from; i <= to; i++ {
			r = append(r, i)
		}
		return r
	}
	if last < onEachSide*2+8 { // small slider
		return [][]int{rng(1, last)}
	}
	window := onEachSide + 4
	start, finish := rng(1, 2), rng(last-1, last)
	switch {
	case p.CurrentPage <= window:
		return [][]int{rng(1, window+onEachSide), nil, finish}
	case p.CurrentPage > last-window:
		return [][]int{start, nil, rng(last-(window+(onEachSide-1)), last)}
	default:
		return [][]int{start, nil, rng(p.CurrentPage-onEachSide, p.CurrentPage+onEachSide), nil, finish}
	}
}

// PageParam is the paginator's current-page resolver: ?page= when it is an integer
// ≥ 1 (filter_var FILTER_VALIDATE_INT), else 1.
func PageParam(c fiber.Ctx) int {
	s := strings.TrimSpace(c.Query("page"))
	if s == "" {
		return 1
	}
	s = strings.TrimPrefix(s, "+")
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || (len(s) > 1 && s[0] == '0') {
		return 1
	}
	return n
}

// RequestURL is $request->url() behind the trusted nginx proxy: scheme and host from
// X-Forwarded-Proto / X-Forwarded-Host / X-Forwarded-Port when present, the raw
// path, no query string, no trailing slash.
func RequestURL(c fiber.Ctx) string {
	scheme := firstHeader(c, fiber.HeaderXForwardedProto)
	if scheme == "" {
		scheme = c.Scheme()
	}
	scheme = strings.ToLower(scheme)
	host := firstHeader(c, fiber.HeaderXForwardedHost)
	if host == "" {
		host = string(c.Request().Host())
	}
	if port := firstHeader(c, "X-Forwarded-Port"); port != "" {
		h := host
		if i := strings.LastIndexByte(h, ':'); i >= 0 && !strings.HasSuffix(h, "]") {
			h = h[:i]
		}
		host = h
		if (scheme == "https" && port != "443") || (scheme == "http" && port != "80") {
			host += ":" + port
		}
	}
	path := strings.TrimRight(rawPath(c), "/")
	return scheme + "://" + host + path
}

func firstHeader(c fiber.Ctx, key string) string {
	v := c.Get(key)
	if i := strings.IndexByte(v, ','); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}
