package validation

import (
	"mime"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Input is $request->all() as a Laravel controller sees it: the JSON body (or form body;
// the query string for GET/HEAD without a JSON body) plus query parameters the body does
// not define, after the global TrimStrings and ConvertEmptyStringsToNull middleware.
// An unparseable JSON body counts as empty, as in Laravel.
func Input(c fiber.Ctx) phpval.Map {
	query := Query(c)
	var source phpval.Map
	switch {
	case isJSON(c):
		source = jsonBody(c.Body())
	case c.Method() == fiber.MethodGet || c.Method() == fiber.MethodHead:
		source = query
	default:
		source = formBody(c)
	}
	all := phpval.NewMap()
	for _, k := range source.Keys() {
		x, _ := source.Get(k)
		all.Set(k, x)
	}
	for _, k := range query.Keys() {
		if _, ok := all.Get(k); !ok {
			x, _ := query.Get(k)
			all.Set(k, x)
		}
	}
	return all
}

// Query is $request->query() (parse_str of the raw query string) after TrimStrings and
// ConvertEmptyStringsToNull.
func Query(c fiber.Ctx) phpval.Map {
	return ParseQuery(string(c.Request().URI().QueryString()))
}

// ParseQuery is parse_str($raw) followed by the two input-normalising middleware.
func ParseQuery(raw string) phpval.Map {
	return Normalize(parseStr(raw)).(phpval.Map)
}

// DecodeBody is the JSON request body as Laravel's $request->json()->all() after the
// input middleware: an object becomes a Map, a list is keyed "0", "1", …, and
// invalid JSON or a scalar body yields an empty Map (scalars: [0 => value]).
func DecodeBody(body []byte) phpval.Map { return jsonBody(body) }

func isJSON(c fiber.Ctx) bool {
	ct := c.Get(fiber.HeaderContentType)
	return strings.Contains(ct, "/json") || strings.Contains(ct, "+json")
}

func jsonBody(b []byte) phpval.Map {
	if len(strings.TrimSpace(string(b))) == 0 {
		return phpval.NewMap()
	}
	v, err := phpval.Decode(b)
	if err != nil {
		return phpval.NewMap()
	}
	return Normalize(toMap(v)).(phpval.Map)
}

// toMap is the (array) cast of a decoded body.
func toMap(v any) phpval.Map {
	switch x := v.(type) {
	case phpval.Map:
		return x
	case []any:
		m := phpval.NewMap()
		for i, e := range x {
			m.Set(strconv.Itoa(i), e)
		}
		return m
	case nil:
		return phpval.NewMap()
	}
	m := phpval.NewMap()
	m.Set("0", v)
	return m
}

func formBody(c fiber.Ctx) phpval.Map {
	ct, _, _ := mime.ParseMediaType(c.Get(fiber.HeaderContentType))
	switch ct {
	case fiber.MIMEApplicationForm:
		return Normalize(parseStr(string(c.Body()))).(phpval.Map)
	case fiber.MIMEMultipartForm:
		form, err := c.MultipartForm()
		if err != nil {
			return phpval.NewMap()
		}
		var parts []string
		for k, vals := range form.Value {
			for _, v := range vals {
				parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
			}
		}
		return Normalize(parseStr(strings.Join(parts, "&"))).(phpval.Map)
	}
	return phpval.NewMap()
}

// parseStr is PHP's parse_str: "a=1&b[]=2&b[]=3&c[x]=4" with "." and " " in the
// top-level name turned into "_".
func parseStr(raw string) phpval.Map {
	out := phpval.NewMap()
	for _, pair := range strings.Split(raw, "&") {
		if pair == "" {
			continue
		}
		k, v, _ := strings.Cut(pair, "=")
		key, err := url.QueryUnescape(k)
		if err != nil {
			continue
		}
		val, err := url.QueryUnescape(v)
		if err != nil {
			continue
		}
		base, rest := key, ""
		if i := strings.IndexByte(key, '['); i > 0 && strings.Contains(key[i:], "]") {
			base, rest = key[:i], key[i:]
		}
		base = strings.NewReplacer(".", "_", " ", "_").Replace(base)
		if base == "" {
			continue
		}
		var segs []string
		for rest != "" && rest[0] == '[' {
			end := strings.IndexByte(rest, ']')
			if end < 0 {
				break
			}
			segs = append(segs, rest[1:end])
			rest = rest[end+1:]
		}
		setQueryValue(out, append([]string{base}, segs...), val)
	}
	packed := phpval.NewMap() // b[]=1&b[]=2 is the list ["1","2"]
	for _, k := range out.Keys() {
		v, _ := out.Get(k)
		packed.Set(k, phpval.Packed(v))
	}
	return packed
}

func setQueryValue(m phpval.Map, segs []string, val string) {
	key := segs[0]
	if key == "" { // "[]": next integer key
		key = strconv.Itoa(nextIndex(m))
	}
	if len(segs) == 1 {
		m.Set(key, val)
		return
	}
	child, ok := m.Get(key)
	cm, isMap := child.(phpval.Map)
	if !ok || !isMap {
		cm = phpval.NewMap()
		m.Set(key, cm)
	}
	setQueryValue(cm, segs[1:], val)
}

func nextIndex(m phpval.Map) int {
	next := 0
	for _, k := range m.Keys() {
		if n, err := strconv.Atoi(k); err == nil && n >= next {
			next = n + 1
		}
	}
	return next
}

// neverTrim are TrimStrings::$except.
var neverTrim = map[string]bool{"current_password": true, "password": true, "password_confirmation": true}

// Normalize applies TrimStrings (Str::trim: whitespace and invisible characters) and
// ConvertEmptyStringsToNull ("" → null) recursively; keys and structure are kept.
func Normalize(v any) any { return normalize("", v) }

func normalize(key string, v any) any {
	switch x := v.(type) {
	case phpval.Map:
		out := phpval.NewMap()
		for _, k := range x.Keys() {
			val, _ := x.Get(k)
			out.Set(k, normalize(joinKey(key, k), val))
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = normalize(joinKey(key, strconv.Itoa(i)), e)
		}
		return out
	case string:
		if !neverTrim[key] {
			x = TrimString(x)
		}
		if x == "" {
			return nil
		}
		return x
	}
	return v
}

func joinKey(prefix, k string) string {
	if prefix == "" {
		return k
	}
	return prefix + "." + k
}

// TrimString is Str::trim($s): strips whitespace, NUL and the "invisible" characters
// Laravel lists (NBSP, zero-width spaces, BOM, …) from both ends.
func TrimString(s string) string {
	return strings.TrimFunc(s, isTrimmable)
}

func isTrimmable(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\v', '\f', '\r', 0,
		0x00A0, 0x00AD, 0x034F, 0x061C, 0x115F, 0x1160, 0x17B4, 0x17B5, 0x180E,
		0x2000, 0x2001, 0x2002, 0x2003, 0x2004, 0x2005, 0x2006, 0x2007, 0x2008, 0x2009, 0x200A,
		0x200B, 0x200C, 0x200D, 0x200E, 0x200F, 0x202F, 0x205F, 0x2060, 0x2061, 0x2062, 0x2063,
		0x2064, 0x2065, 0x206A, 0x206B, 0x206C, 0x206D, 0x206E, 0x206F, 0x3000, 0x2800, 0x3164,
		0xFEFF, 0xFFA0, 0x1D159, 0x1D173, 0x1D174, 0x1D175, 0x1D176, 0x1D177, 0x1D178, 0x1D179,
		0x1D17A, 0xE0020:
		return true
	}
	return false
}
