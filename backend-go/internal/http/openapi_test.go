package http

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	_ "github.com/go-sql-driver/mysql" // lazy handle for the route inventory; never connects
	"github.com/gofiber/fiber/v3"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	"github.com/ritme/backend-go/api"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// The OpenAPI document (api/openapi.yaml) is hand-maintained. These tests keep it honest:
//
//   - every public route registered by a routes_<domain>.go file has an operation in the
//     spec and every operation in the spec is a registered route;
//   - every contract golden (contract/golden/**, recorded from Laravel) validates against
//     the response schema of its operation and status code.
//
// Scope: the public surface (/api/v1/*, /up, /storage/*). The admin API (/api/admin/v1) is
// documented in docs/go-migration/admin-api.md and the docs UI (/docs) documents itself,
// so both are ignored here in both directions.

const (
	specURL    = "file:///openapi.json"
	goldenRoot = "../../contract/golden"
	keysDir    = "../../contract/fixtures/keys"
)

// outOfScope reports whether a path is not part of the documented public surface.
func outOfScope(path string) bool {
	return strings.HasPrefix(path, "/api/admin/") || path == docsPrefix || strings.HasPrefix(path, docsPrefix+"/")
}

var httpMethods = []string{"get", "put", "post", "delete", "patch", "options", "head", "trace"}

// ---------------------------------------------------------------------------- spec

type openAPI struct {
	doc      map[string]any
	compiler *jsonschema.Compiler
	ops      []specOp

	mu       sync.Mutex
	compiled map[string]*jsonschema.Schema
}

type specOp struct {
	method   string // upper case
	path     string // /api/v1/articles/{slug}
	segments []string
	statics  int
	op       map[string]any
}

var (
	specOnce sync.Once
	specVal  *openAPI
	specErr  error
)

func loadSpec(t *testing.T) *openAPI {
	t.Helper()
	specOnce.Do(func() { specVal, specErr = parseSpec(api.OpenAPI) })
	require.NoError(t, specErr)
	return specVal
}

func parseSpec(raw []byte) (*openAPI, error) {
	var y any
	if err := yaml.Unmarshal(raw, &y); err != nil {
		return nil, fmt.Errorf("openapi.yaml: %w", err)
	}
	js, err := json.Marshal(y)
	if err != nil {
		return nil, fmt.Errorf("openapi.yaml is not JSON-compatible: %w", err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(js))
	if err != nil {
		return nil, err
	}
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	if err := c.AddResource(specURL, doc); err != nil {
		return nil, err
	}
	s := &openAPI{doc: doc.(map[string]any), compiler: c, compiled: map[string]*jsonschema.Schema{}}
	paths, _ := s.doc["paths"].(map[string]any)
	for p, item := range paths {
		m, _ := item.(map[string]any)
		for _, method := range httpMethods {
			op, ok := m[method].(map[string]any)
			if !ok {
				continue
			}
			segs := strings.Split(strings.Trim(p, "/"), "/")
			statics := 0
			for _, sg := range segs {
				if !strings.HasPrefix(sg, "{") {
					statics++
				}
			}
			s.ops = append(s.ops, specOp{method: strings.ToUpper(method), path: p, segments: segs, statics: statics, op: op})
		}
	}
	return s, nil
}

// schema compiles the JSON schema at a JSON pointer inside the spec (cached).
func (s *openAPI) schema(pointer string) (*jsonschema.Schema, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sch, ok := s.compiled[pointer]; ok {
		return sch, nil
	}
	sch, err := s.compiler.Compile(specURL + "#" + pointer)
	if err != nil {
		return nil, err
	}
	s.compiled[pointer] = sch
	return sch, nil
}

func escapePointer(tok string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(tok)
}

// lookup follows a "#/a/b" pointer inside the document.
func (s *openAPI) lookup(ref string) (map[string]any, bool) {
	var cur any = s.doc
	for _, tok := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		tok = strings.NewReplacer("~1", "/", "~0", "~").Replace(tok)
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur = m[tok]
	}
	m, ok := cur.(map[string]any)
	return m, ok
}

// match finds the operation for a concrete request path the way Fiber routes it: static
// segments beat parameters, `{path}` as the last segment of /storage swallows the rest.
func (s *openAPI) match(method, urlPath string) (op *specOp, pathKnown bool) {
	if method == fiber.MethodHead {
		method = fiber.MethodGet
	}
	segs := strings.Split(strings.Trim(urlPath, "/"), "/")
	for i := range s.ops {
		cand := &s.ops[i]
		if !segmentsMatch(cand.segments, segs) {
			continue
		}
		pathKnown = true
		if cand.method != method {
			continue
		}
		if op == nil || cand.statics > op.statics {
			op = cand
		}
	}
	return op, pathKnown
}

func segmentsMatch(tmpl, segs []string) bool {
	if n := len(tmpl); n > 0 && tmpl[n-1] == "{path}" && tmpl[0] == "storage" {
		return len(segs) >= n && segmentsMatch(tmpl[:n-1], segs[:n-1])
	}
	if len(tmpl) != len(segs) {
		return false
	}
	for i, sg := range tmpl {
		if !strings.HasPrefix(sg, "{") && sg != segs[i] {
			return false
		}
	}
	return true
}

// responsePointer returns the pointer of the JSON schema documented for status, following
// a components/responses $ref. ok is false when the status is not documented at all;
// pointer is "" when the response has no application/json body.
func (s *openAPI) responsePointer(base string, op map[string]any, status int) (pointer string, ok bool) {
	responses, _ := op["responses"].(map[string]any)
	resp, found := responses[strconv.Itoa(status)].(map[string]any)
	if !found {
		return "", false
	}
	at := base + "/responses/" + strconv.Itoa(status)
	if ref, isRef := resp["$ref"].(string); isRef {
		if resp, found = s.lookup(ref); !found {
			return "", false
		}
		at = strings.TrimPrefix(ref, "#")
	}
	content, _ := resp["content"].(map[string]any)
	if _, has := content["application/json"]; !has {
		return "", true
	}
	return at + "/content/application~1json/schema", true
}

func opPointer(o *specOp) string {
	return "/paths/" + escapePointer(o.path) + "/" + strings.ToLower(o.method)
}

// ---------------------------------------------------------------------------- routes

// registeredRoutes mounts the default registry (every routes_<domain>.go) on a bare Fiber
// app with inert dependencies and returns "METHOD /path/{param}" for the public surface.
func registeredRoutes(t *testing.T) []string {
	t.Helper()
	keys, err := filepath.Abs(keysDir)
	require.NoError(t, err)
	db, err := sql.Open("mysql", "inventory:inventory@tcp(127.0.0.1:1)/inventory")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	rc := cache.New(config.Redis{Host: "127.0.0.1", Port: 1, Prefix: "ritme-go-test:"})
	t.Cleanup(func() { _ = rc.Redis().Close() })

	app := fiber.New()
	Mount(app, &Deps{
		Config: &config.Config{
			App:         config.App{Env: "testing", URL: "http://localhost"},
			SMS:         config.SMS{Provider: "log"},
			Passport:    config.Passport{TokenLifetimeDays: 365, RefreshWindowDays: 30},
			StoragePath: keys,
		},
		DB:     db,
		Cache:  rc,
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	})

	seen := map[string]bool{}
	var out []string
	for _, r := range app.GetRoutes(true) {
		// Fiber answers HEAD for every GET by itself; OPTIONS are CORS preflights.
		if r.Method == fiber.MethodHead || r.Method == fiber.MethodOptions || outOfScope(r.Path) {
			continue
		}
		key := r.Method + " " + fiberToOpenAPI(r.Path)
		if !seen[key] {
			seen[key] = true
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

// fiberToOpenAPI turns /a/:id/* into /a/{id}/{path}.
func fiberToOpenAPI(p string) string {
	segs := strings.Split(p, "/")
	for i, sg := range segs {
		switch {
		case strings.HasPrefix(sg, ":"):
			segs[i] = "{" + strings.TrimSuffix(strings.TrimPrefix(sg, ":"), "?") + "}"
		case sg == "*" || sg == "+":
			segs[i] = "{path}"
		}
	}
	return strings.Join(segs, "/")
}

func TestOpenAPI_EveryRouteDocumentedAndViceVersa(t *testing.T) {
	spec := loadSpec(t)
	routes := registeredRoutes(t)
	require.NotEmpty(t, routes)

	var documented []string
	for _, o := range spec.ops {
		if !outOfScope(o.path) {
			documented = append(documented, o.method+" "+o.path)
		}
	}
	sort.Strings(documented)

	missing := difference(routes, documented)
	stale := difference(documented, routes)
	assert.Empty(t, missing, "routes registered in internal/http/routes_*.go without an operation in api/openapi.yaml")
	assert.Empty(t, stale, "operations in api/openapi.yaml without a registered route")
}

func difference(a, b []string) []string {
	in := map[string]bool{}
	for _, x := range b {
		in[x] = true
	}
	var out []string
	for _, x := range a {
		if !in[x] {
			out = append(out, x)
		}
	}
	return out
}

// ---------------------------------------------------------------------------- well-formedness

func TestOpenAPI_SpecIsWellFormed(t *testing.T) {
	spec := loadSpec(t)
	assert.Equal(t, "3.1.0", spec.doc["openapi"])

	ids := map[string]string{}
	for i := range spec.ops {
		o := &spec.ops[i]
		name := o.method + " " + o.path
		id, _ := o.op["operationId"].(string)
		if assert.NotEmpty(t, id, "%s: operationId", name) {
			assert.NotContains(t, ids, id, "%s: operationId %q also used by %s", name, id, ids[id])
			ids[id] = name
		}
		assert.NotEmpty(t, o.op["summary"], "%s: summary", name)
		assert.NotEmpty(t, o.op["tags"], "%s: tags", name)

		// Every {param} in the path is declared as a path parameter.
		declared := map[string]bool{}
		params, _ := o.op["parameters"].([]any)
		for _, p := range params {
			pm, _ := p.(map[string]any)
			if ref, ok := pm["$ref"].(string); ok {
				pm, _ = spec.lookup(ref)
			}
			if pm["in"] == "path" {
				declared[pm["name"].(string)] = true
			}
		}
		for _, sg := range o.segments {
			if strings.HasPrefix(sg, "{") {
				assert.True(t, declared[strings.Trim(sg, "{}")], "%s: path parameter %s not declared", name, sg)
			}
		}

		// Every response and request body schema compiles (all $refs resolve).
		base := opPointer(o)
		responses, _ := o.op["responses"].(map[string]any)
		require.NotEmpty(t, responses, "%s: responses", name)
		for code := range responses {
			status, err := strconv.Atoi(code)
			require.NoError(t, err, "%s: response key %q", name, code)
			ptr, ok := spec.responsePointer(base, o.op, status)
			require.True(t, ok, "%s %d: dangling $ref", name, status)
			if ptr != "" {
				_, err := spec.schema(ptr)
				assert.NoError(t, err, "%s %d: schema", name, status)
			}
		}
		if rb, ok := o.op["requestBody"].(map[string]any); ok {
			content, _ := rb["content"].(map[string]any)
			for ct := range content {
				_, err := spec.schema(base + "/requestBody/content/" + escapePointer(ct) + "/schema")
				assert.NoError(t, err, "%s: request body %s", name, ct)
			}
		}
	}
}

// ---------------------------------------------------------------------------- goldens

type goldenFile struct {
	Case  string       `json:"case"`
	Steps []goldenStep `json:"steps"`
}

type goldenStep struct {
	Request struct {
		Method string `json:"method"`
		URL    string `json:"url"`
	} `json:"request"`
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
}

func TestOpenAPI_GoldensValidateAgainstSpec(t *testing.T) {
	spec := loadSpec(t)
	files := goldenFiles(t)
	require.NotEmpty(t, files, "no goldens under %s", goldenRoot)

	notFound := "/components/responses/RouteNotFound"
	notAllowed := "/components/responses/MethodNotAllowed"
	validated := 0
	for _, file := range files {
		raw, err := os.ReadFile(file)
		require.NoError(t, err)
		var g goldenFile
		require.NoError(t, json.Unmarshal(raw, &g), file)
		for i, st := range g.Steps {
			where := fmt.Sprintf("%s step %d: %s %s → %d", g.Case, i, st.Request.Method, st.Request.URL, st.Status)
			urlPath, _, _ := strings.Cut(st.Request.URL, "?")

			var ptr string
			op, pathKnown := spec.match(st.Request.Method, urlPath)
			switch {
			case op != nil:
				var ok bool
				ptr, ok = spec.responsePointer(opPointer(op), op.op, st.Status)
				if !assert.True(t, ok, "%s: status not documented for %s %s", where, op.method, op.path) {
					continue
				}
			case pathKnown:
				if !assert.Equal(t, fiber.StatusMethodNotAllowed, st.Status, "%s: no operation for this request", where) {
					continue
				}
				ptr = notAllowed + "/content/application~1json/schema"
			default:
				if !assert.Equal(t, fiber.StatusNotFound, st.Status, "%s: no operation for this request", where) {
					continue
				}
				ptr = notFound + "/content/application~1json/schema"
			}

			body := bytes.TrimSpace(st.Body)
			if len(body) == 0 || bytes.Equal(body, []byte("null")) {
				continue // HEAD, /up (text), ignored bodies
			}
			if !assert.NotEmpty(t, ptr, "%s: JSON body but the response documents no application/json content", where) {
				continue
			}
			sch, err := spec.schema(ptr)
			require.NoError(t, err, where)
			inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(body))
			require.NoError(t, err, where)
			if err := sch.Validate(inst); err != nil {
				t.Errorf("%s (schema %s):\n%v", where, ptr, err)
				continue
			}
			validated++
		}
	}
	t.Logf("validated %d golden bodies from %d files", validated, len(files))
}

func goldenFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(goldenRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(p, ".json") {
			files = append(files, p)
		}
		return nil
	})
	require.NoError(t, err)
	sort.Strings(files)
	return files
}

// ---------------------------------------------------------------------------- /docs

func docsApp(env, user, pass string) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(slog.New(slog.NewJSONHandler(io.Discard, nil)))})
	mountDocs(app, &config.Config{
		App:     config.App{Env: env},
		Swagger: config.Swagger{User: user, Password: pass},
	})
	return app
}

func docsGet(t *testing.T, app *fiber.App, path, user, pass string) (*nethttp.Response, string) {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodGet, path, nil)
	if user != "" || pass != "" {
		req.Header.Set(fiber.HeaderAuthorization, "Basic "+base64.StdEncoding.EncodeToString([]byte(user+":"+pass)))
	}
	resp, err := app.Test(req)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	_ = resp.Body.Close()
	return resp, string(body)
}

func TestOpenAPI_DocsBasicAuth(t *testing.T) {
	app := docsApp("production", "ritme", "s3cret")
	for _, p := range []string{docsPrefix, docsPrefix + "/openapi.yaml", docsPrefix + "/assets/swagger-ui.css"} {
		for _, creds := range [][2]string{{"", ""}, {"ritme", "wrong"}, {"other", "s3cret"}} {
			resp, body := docsGet(t, app, p, creds[0], creds[1])
			assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode, "%s %v", p, creds)
			assert.Equal(t, `Basic realm="Ritme API Docs"`, resp.Header.Get(fiber.HeaderWWWAuthenticate))
			assert.Equal(t, "Authentication required.", body)
		}
	}

	resp, body := docsGet(t, app, docsPrefix, "ritme", "s3cret")
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get(fiber.HeaderContentType), "text/html")
	assert.Contains(t, body, `id="swagger-ui"`)
	assert.Contains(t, body, "/docs/assets/swagger-ui-bundle.js")
	assert.Contains(t, body, "openapi.yaml")
	assert.NotContains(t, body, "https://", "the docs page must not load anything from a CDN")

	resp, body = docsGet(t, app, docsPrefix+"/openapi.yaml", "ritme", "s3cret")
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, string(api.OpenAPI), body)

	for _, asset := range []string{"swagger-ui-bundle.js", "swagger-ui-standalone-preset.js", "swagger-ui.css", "favicon-32x32.png"} {
		resp, body = docsGet(t, app, docsPrefix+"/assets/"+asset, "ritme", "s3cret")
		require.Equal(t, fiber.StatusOK, resp.StatusCode, asset)
		assert.NotEmpty(t, body, asset)
	}
	resp, body = docsGet(t, app, docsPrefix+"/assets/swagger-ui-bundle.js", "ritme", "s3cret")
	assert.Contains(t, body, "SwaggerUIBundle", "bundle is inflated for clients without gzip")
	assert.Empty(t, resp.Header.Get(fiber.HeaderContentEncoding))
	resp, _ = docsGet(t, app, docsPrefix+"/assets/nope.js", "ritme", "s3cret")
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestOpenAPI_DocsFailClosedWithoutPassword(t *testing.T) {
	for _, env := range []string{"production", "staging", ""} {
		resp, _ := docsGet(t, docsApp(env, "ritme", ""), docsPrefix, "ritme", "")
		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode, "APP_ENV=%q", env)
	}
	for _, env := range []string{"local", "testing"} {
		resp, _ := docsGet(t, docsApp(env, "ritme", ""), docsPrefix, "", "")
		assert.Equal(t, fiber.StatusOK, resp.StatusCode, "APP_ENV=%q", env)
	}
}
