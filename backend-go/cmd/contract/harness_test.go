package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	"github.com/ritme/backend-go/cmd/contract/internal/ojson"
)

func parseCaseFile(t *testing.T, src string) *CaseFile {
	t.Helper()
	var f CaseFile
	dec := yaml.NewDecoder(strings.NewReader(src))
	dec.KnownFields(true)
	require.NoError(t, dec.Decode(&f))
	return &f
}

func TestExpand_PersonaLocaleMatrixAndIDs(t *testing.T) {
	f := parseCaseFile(t, `
group: demo
defaults:
  locales: [none, fa]
cases:
  - name: list
    persona: [regular, engaged]
    method: get
    path: /api/v1/reminders
    query: {type: doctor}
    status: 200
  - name: single
    persona: regular
    locales: en
    method: GET
    path: /api/v1/x
    status: 200
`)
	cases, err := f.Expand()
	require.NoError(t, err)
	var ids []string
	for _, c := range cases {
		ids = append(ids, c.ID)
	}
	assert.Equal(t, []string{"list.regular.none", "list.regular.fa", "list.engaged.none", "list.engaged.fa", "single"}, ids)
	assert.Equal(t, "GET", cases[0].Steps[0].Method)
	assert.Empty(t, cases[0].Steps[0].Locale)
	assert.Equal(t, "fa", cases[1].Steps[0].Locale)
	assert.Equal(t, "en", cases[4].Steps[0].Locale)
	assert.Equal(t, DefaultNow, cases[0].Steps[0].Now)
	assert.Equal(t, "/api/v1/reminders?type=doctor", cases[0].Steps[0].URL(nil))
	assert.False(t, cases[0].Writes())
}

func TestExpand_StepsInheritCaseSettings(t *testing.T) {
	f := parseCaseFile(t, `
group: demo
cases:
  - name: flow
    persona: engaged
    now: "2026-09-24T08:00:00+03:30"
    ignore: [data.created_at, "header:x-ratelimit-remaining"]
    patterns: {data.token: "^eyJ"}
    steps:
      - {method: POST, path: /api/v1/reminders, body: {title: T, is_active: true, n: 1.50, d: 2026-09-23, x: ~}, status: 201, capture: {rid: data.id}}
      - {method: PUT, path: "/api/v1/reminders/{{rid}}", as: regular, now: none, status: 404}
`)
	cases, err := f.Expand()
	require.NoError(t, err)
	require.Len(t, cases, 1)
	c := cases[0]
	assert.True(t, c.Writes())
	s1, s2 := c.Steps[0], c.Steps[1]
	assert.Equal(t, "2026-09-24T08:00:00+03:30", s1.Now)
	assert.Equal(t, "none", s2.Now)
	assert.Equal(t, "engaged", s1.Persona)
	assert.Equal(t, "regular", s2.Persona)
	assert.Equal(t, []string{"x-ratelimit-remaining"}, s2.IgnoreHeaders)
	// YAML scalars keep their type; the date stays a string; key order is kept.
	assert.Equal(t, `{"title":"T","is_active":true,"n":1.50,"d":"2026-09-23","x":null}`, string(ojson.Encode(s1.Body.V, "")))
	assert.Equal(t, "/api/v1/reminders/42", s2.URL(map[string]string{"rid": "42"}))
	// Rules apply to every step.
	assert.Len(t, s2.Rules.Ignore, 1)
	assert.Len(t, s2.Rules.Patterns, 1)
}

func TestExpand_Sweep(t *testing.T) {
	f := parseCaseFile(t, `
group: demo
cases:
  - name: date
    persona: regular
    sweep: {var: d, from: "2026-09-29", days: 3}
    method: GET
    path: /api/v1/cycle/date/{{d}}
    status: 200
  - name: month
    persona: regular
    sweep: {var: ym, values: ["2026/8", "2026/9"]}
    method: GET
    path: /api/v1/cycle/month/{{ym}}
    query: {view: calendar}
    status: 200
`)
	cases, err := f.Expand()
	require.NoError(t, err)
	require.Len(t, cases, 2)
	var urls []string
	for _, c := range cases {
		for _, s := range c.Steps {
			urls = append(urls, s.URL(nil))
		}
	}
	assert.Equal(t, []string{
		"/api/v1/cycle/date/2026-09-29", "/api/v1/cycle/date/2026-09-30", "/api/v1/cycle/date/2026-10-01",
		"/api/v1/cycle/month/2026/8?view=calendar", "/api/v1/cycle/month/2026/9?view=calendar",
	}, urls)
}

func TestExpand_Errors(t *testing.T) {
	for name, src := range map[string]string{
		"no status":      "group: d\ncases:\n  - {name: a, method: GET, path: /x}",
		"bad name":       "group: d\ncases:\n  - {name: A B, method: GET, path: /x, status: 200}",
		"both forms":     "group: d\ncases:\n  - {name: a, method: GET, path: /x, status: 200, steps: [{method: GET, path: /y, status: 200}]}",
		"duplicate id":   "group: d\ncases:\n  - {name: a, method: GET, path: /x, status: 200}\n  - {name: a, method: GET, path: /y, status: 200}",
		"relative path":  "group: d\ncases:\n  - {name: a, method: GET, path: x, status: 200}",
		"bad pattern":    "group: d\ncases:\n  - {name: a, method: GET, path: /x, status: 200, patterns: {a: '('}}",
		"body+raw":       "group: d\ncases:\n  - {name: a, method: POST, path: /x, status: 200, body: {a: 1}, raw_body: 'x'}",
		"sweep no range": "group: d\ncases:\n  - {name: a, method: GET, path: /x, status: 200, sweep: {var: d}}",
	} {
		_, err := parseCaseFile(t, src).Expand()
		assert.Error(t, err, name)
	}
}

func TestSelectGroups(t *testing.T) {
	all := map[string]*CaseFile{"health": {}, "cycle": {}, "cycle-sweep": {}, "public": {}}
	got, err := SelectGroups(all, "all")
	require.NoError(t, err)
	assert.Equal(t, []string{"cycle", "cycle-sweep", "health", "public"}, got)
	got, err = SelectGroups(all, "health, cycle*")
	require.NoError(t, err)
	assert.Equal(t, []string{"health", "cycle", "cycle-sweep"}, got)
	_, err = SelectGroups(all, "nope")
	require.Error(t, err)
}

func headerMap(kv ...string) http.Header {
	h := http.Header{}
	for i := 0; i+1 < len(kv); i += 2 {
		h.Set(kv[i], kv[i+1])
	}
	return h
}

// A golden built from a Laravel result must compare clean against an equivalent Go
// result (escaping, key order, volatile fields) and flag real differences.
func TestGolden_RoundTripAndCompare(t *testing.T) {
	f := parseCaseFile(t, `
group: demo
cases:
  - name: login
    method: POST
    path: /api/v1/auth/verify-otp
    body: {mobile: "09900000004", code: "1234"}
    status: 200
    patterns: {data.access_token: "^eyJ"}
  - name: err
    method: GET
    path: /api/v1/nope
    status: 404
    strict_bytes: true
`)
	cases, err := f.Expand()
	require.NoError(t, err)

	laravel := Result{Method: "POST", URL: "/api/v1/auth/verify-otp", Body: []byte(`{"mobile":"09900000004","code":"1234"}`),
		Status: 200, Headers: headerMap("Content-Type", "application/json", "X-RateLimit-Limit", "10", "X-RateLimit-Remaining", "9"),
		Raw: []byte(`{"success":true,"data":{"user":{"url":"http:\/\/127.0.0.1:8090\/a","name":"س"},"access_token":"eyJold","n":"65.50"}}`)}
	doc, problems := BuildGolden(cases[0], []Result{laravel})
	assert.Empty(t, problems)
	assert.Contains(t, string(doc), `"access_token": "<pattern:^eyJ>"`)

	dir := t.TempDir()
	path := filepath.Join(dir, "g.json")
	require.NoError(t, os.WriteFile(path, doc, 0o600))
	golden, err := LoadGolden(path)
	require.NoError(t, err)
	require.Len(t, golden, 1)

	goRes := laravel
	goRes.Raw = []byte(`{"data":{"n":"65.50","access_token":"eyJnew","user":{"name":"س","url":"http://127.0.0.1:8090/a"}},"success":true}`)
	assert.Empty(t, CompareStep(1, &cases[0].Steps[0], &golden[0], &goRes))

	goRes.Raw = []byte(`{"data":{"n":65.5,"access_token":"nope","user":{"name":"س","url":"http://127.0.0.1:8090/a"}},"success":true}`)
	goRes.Headers = headerMap("Content-Type", "application/json; charset=utf-8", "X-RateLimit-Limit", "10")
	var got []string
	for _, m := range CompareStep(1, &cases[0].Steps[0], &golden[0], &goRes) {
		got = append(got, m.Where)
	}
	assert.ElementsMatch(t, []string{"header:content-type", "header:x-ratelimit-remaining", "$.data.n", "$.data.access_token"}, got)

	// strict_bytes: same JSON, different whitespace → difference.
	pretty := Result{Method: "GET", URL: "/api/v1/nope", Status: 404, Headers: headerMap("Content-Type", "application/json"),
		Raw: []byte("{\n    \"message\": \"The route api/v1/nope could not be found.\"\n}")}
	doc, problems = BuildGolden(cases[1], []Result{pretty})
	assert.Empty(t, problems)
	require.NoError(t, os.WriteFile(path, doc, 0o600))
	golden, err = LoadGolden(path)
	require.NoError(t, err)
	assert.Empty(t, CompareStep(1, &cases[1].Steps[0], &golden[0], &pretty))
	compact := pretty
	compact.Raw = []byte(`{"message":"The route api/v1/nope could not be found."}`)
	m := CompareStep(1, &cases[1].Steps[0], &golden[0], &compact)
	require.Len(t, m, 1)
	assert.Equal(t, "body", m[0].Where)
}

func TestBuildGolden_FlagsUnexpectedStatusAndPattern(t *testing.T) {
	f := parseCaseFile(t, "group: d\ncases:\n  - {name: a, method: GET, path: /x, status: 200, patterns: {t: '^eyJ'}}")
	cases, err := f.Expand()
	require.NoError(t, err)
	_, problems := BuildGolden(cases[0], []Result{{Method: "GET", URL: "/x", Status: 500, Headers: http.Header{}, Raw: []byte(`{"t":"x"}`)}})
	assert.Len(t, problems, 2)
}

func TestCanonicalise(t *testing.T) {
	in := []byte(`{"a":"http:\/\/127.0.0.1:18090\/storage\/x.png","b":"http://127.0.0.1:18090/api/v1/health-logs?page=2"}`)
	assert.Equal(t, `{"a":"http:\/\/127.0.0.1:8090\/storage\/x.png","b":"http://127.0.0.1:8090/api/v1/health-logs?page=2"}`,
		string(canonicalise(in, "http://127.0.0.1:18090")))
}

func TestAllowlist(t *testing.T) {
	root := t.TempDir()
	devs := filepath.Join(root, "deviations.md")
	require.NoError(t, os.WriteFile(devs, []byte("| D-01 | Reminders | x | y | proposed |\n"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "allowlist"), 0o750))
	write := func(s string) {
		require.NoError(t, os.WriteFile(filepath.Join(root, "allowlist", "reminders.yaml"), []byte(s), 0o600))
	}

	write(`group: reminders
entries:
  - {case: "enums.*.ar", where: status, deviation: D-01, reason: 500 in Laravel}
  - {case: "enums.*", step: 1, where: "data.recurrences[*].label", deviation: D-01}
`)
	al, err := LoadAllowlist(root, "reminders", devs)
	require.NoError(t, err)
	assert.NotNil(t, al.Allowed("enums.engaged.ar", Mismatch{Step: 1, Kind: "status", Where: "status"}))
	assert.Nil(t, al.Allowed("enums.engaged.fa", Mismatch{Step: 1, Kind: "status", Where: "status"}))
	assert.NotNil(t, al.Allowed("enums.engaged.fa", Mismatch{Step: 1, Kind: "body", Path: ojson.Path{"data", "recurrences", "2", "label"}}))
	assert.Nil(t, al.Allowed("enums.engaged.fa", Mismatch{Step: 2, Kind: "body", Path: ojson.Path{"data", "recurrences", "2", "label"}}))

	write("entries:\n  - {case: x, where: status, deviation: D-99}\n")
	_, err = LoadAllowlist(root, "reminders", devs)
	require.ErrorContains(t, err, "D-99 is not in")

	write("entries:\n  - {case: x, where: status, deviation: nope}\n")
	_, err = LoadAllowlist(root, "reminders", devs)
	require.ErrorContains(t, err, "not a D-nn id")

	al, err = LoadAllowlist(root, "missing-group", devs)
	require.NoError(t, err)
	assert.Empty(t, al.Entries)
}

func TestMinter_PassportShapedToken(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "oauth-private.key")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600))

	m, err := NewMinter(path, ContractClientID)
	require.NoError(t, err)
	now := time.Date(2026, 9, 23, 6, 30, 0, 123456000, time.UTC)
	tok, row, err := m.Mint(1004, now, now.AddDate(1, 0, 0), now, false)
	require.NoError(t, err)

	parts := strings.Split(tok, ".")
	require.Len(t, parts, 3)
	hdr, _ := base64.RawURLEncoding.DecodeString(parts[0])
	assert.JSONEq(t, `{"typ":"JWT","alg":"RS256"}`, string(hdr))
	payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var claims map[string]any
	require.NoError(t, json.Unmarshal(payload, &claims))
	assert.Equal(t, ContractClientID, claims["aud"])
	assert.Equal(t, "1004", claims["sub"])
	assert.Equal(t, row.ID, claims["jti"])
	assert.Len(t, row.ID, 80)
	assert.Contains(t, string(payload), `"iat":1790145000.123456`)

	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	require.NoError(t, rsa.VerifyPKCS1v15(&key.PublicKey, 5, sum[:], sig)) // 5 = crypto.SHA256

	assert.Equal(t, "2026-09-23 10:00:00", row.CreatedAt) // Tehran wall-clock
	assert.Contains(t, row.InsertSQL(), "'"+ContractClientID+"'")

	synth, err := m.MintSynthetic(now)
	require.NoError(t, err)
	byName := map[string]Session{}
	for _, s := range synth {
		byName[s.Persona] = s
	}
	assert.True(t, byName["token_revoked"].Row.Revoked)
	assert.Nil(t, byName["token_unknown"].Row)
	assert.Equal(t, "not-a-jwt", byName["token_garbage"].Token)
	assert.ElementsMatch(t, syntheticPersonas, sortedKeys(byName))
}

func TestCleanDumpAndResetScript(t *testing.T) {
	raw := []byte("/*M!999999\\- enable the sandbox mode */ \nCREATE TABLE x (id int);\n")
	assert.Equal(t, "CREATE TABLE x (id int);\n", string(cleanDump(raw)))
	script := string(resetScript([]byte("DUMP;"), map[string]Session{
		"b": {Row: &TokenRow{ID: "2", UserID: 2}}, "a": {Row: &TokenRow{ID: "1", UserID: 1}}, "c": {},
	}))
	assert.True(t, strings.HasPrefix(script, "DUMP;\n"))
	assert.Less(t, strings.Index(script, "VALUES ('1'"), strings.Index(script, "VALUES ('2'"))
}
