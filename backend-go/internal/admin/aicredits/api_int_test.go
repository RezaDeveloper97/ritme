package aicredits_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/aicredits"
	"github.com/ritme/backend-go/internal/admin/billing"
	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/ai/credits"
	"github.com/ritme/backend-go/internal/ai/usage"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/plus"
)

func TestMain(m *testing.M) { testdb.Main(m) }

type env struct {
	*admintest.Env
}

// newEnv mounts the module with an Arvan adapter whose gateway is srv (nil = no provider configured).
func newEnv(t *testing.T, srv *httptest.Server) *env {
	t.Helper()
	e := &env{Env: admintest.New(t)}
	var gw aicredits.Gateway
	if srv != nil {
		compat := ai.NewOpenAICompat(ai.OpenAICompatConfig{Provider: ai.ProviderArvan, BaseURL: srv.URL + "/v1",
			APIKey: "test-key", AuthScheme: "apikey"}, srv.Client())
		gw = ai.NewClientWith(ai.Options{Provider: ai.ProviderArvan,
			Adapters: map[string]ai.Adapter{ai.ProviderArvan: {Chatter: compat, Lister: compat}}})
	}
	aicredits.New(e.DB, gw, admintest.Quiet).Routes(e.Route(), e.Kit)
	svc := plus.NewService(e.DB, config.Plus{VATRateBps: 1000, TrialDays: 7, InvoiceTTL: 30 * time.Minute}, plus.FakeGateway{}, admintest.Quiet)
	billing.New(e.DB, svc, nil, admintest.Quiet).Routes(e.Route(), e.Kit)
	return e
}

func id(v uint64) string { return strconv.FormatUint(v, 10) }

func num(v any) uint64 { f, _ := v.(float64); return uint64(f) }

func (e *env) user(mobile string) uint64 {
	e.T.Helper()
	res := e.Exec("INSERT INTO users (mobile, name, created_at, updated_at) VALUES (?, 'Sara', NOW(), NOW())", mobile)
	n, err := res.LastInsertId()
	require.NoError(e.T, err)
	return uint64(n) //nolint:gosec // test ids
}

func (e *env) modelID(code string) uint64 {
	return uint64(e.Int("SELECT id FROM ai_models WHERE code = ?", code)) //nolint:gosec // test ids
}

func (e *env) featureID(key string) uint64 {
	return uint64(e.Int("SELECT id FROM ai_features WHERE `key` = ?", key)) //nolint:gosec // test ids
}

func TestGuards(t *testing.T) {
	e := newEnv(t, nil)
	u := e.user("09121230001")
	model, feat := e.modelID("gemini-flash"), e.featureID("assistant")
	pack := uint64(e.Int("SELECT id FROM ai_credit_packs WHERE code = 'credits_50'")) //nolint:gosec // test ids

	reads := []string{"/ai/models", "/ai/models/" + id(model), "/ai/features", "/ai/credit-packs",
		"/ai/credit-packs/" + id(pack), "/users/" + id(u) + "/ai-credits"}
	editor := e.As(admintest.EditorID)
	for _, p := range reads {
		assert.Equal(t, 401, e.Anonymous().Get(p).Status, p)
		assert.Equal(t, 200, editor.Get(p).Status, "an editor reads "+p)
	}
	writes := []struct{ method, path string }{
		{fiber.MethodPost, "/ai/models"},
		{fiber.MethodPost, "/ai/models/sync"},
		{fiber.MethodPut, "/ai/models/" + id(model)},
		{fiber.MethodDelete, "/ai/models/" + id(model)},
		{fiber.MethodPut, "/ai/features/" + id(feat)},
		{fiber.MethodPost, "/ai/credit-packs"},
		{fiber.MethodPut, "/ai/credit-packs/" + id(pack)},
		{fiber.MethodDelete, "/ai/credit-packs/" + id(pack)},
		{fiber.MethodPost, "/users/" + id(u) + "/ai-credits/adjust"},
	}
	for _, w := range writes {
		assert.Equal(t, 403, editor.JSON(w.method, w.path, map[string]any{"delta": 5, "note": "gift"}).Status, w.path)
	}
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM ai_credit_grants"))

	noToken := e.As(admintest.SuperID)
	noToken.CSRF = ""
	r := noToken.JSON(fiber.MethodPost, "/users/"+id(u)+"/ai-credits/adjust", map[string]any{"delta": 5, "note": "gift"})
	assert.Equal(t, 419, r.Status)
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM ai_credit_grants"))

	super := e.As(admintest.SuperID)
	for _, p := range []string{"/ai/models/999999", "/ai/models/x", "/ai/credit-packs/999999", "/users/999999/ai-credits"} {
		assert.Equal(t, 404, super.Get(p).Status, p)
	}
	assert.Equal(t, 404, super.JSON(fiber.MethodPut, "/ai/features/999999", map[string]any{"credit_cost": 1}).Status)
}

func TestModelsCRUD(t *testing.T) {
	e := newEnv(t, nil)
	c := e.As(admintest.SuperID)

	r := c.Get("/ai/models")
	require.Equal(t, 200, r.Status, r.Body)
	// goose 00125 seeds Gemini Flash, 00127 the speech-to-text model of voice logging.
	require.Len(t, r.Items(), 2)
	seeded := map[string]map[string]any{}
	for _, it := range r.Items() {
		row := it.(map[string]any)
		seeded[row["code"].(string)] = row
	}
	seed := seeded["gemini-flash"]
	require.NotNil(t, seed)
	assert.EqualValues(t, 5, seed["slots_count"], "the four seeded slots use it, and voice_stt as its fallback")
	stt := seeded["gpt-4o-transcribe"]
	require.NotNil(t, stt)
	assert.Equal(t, "arvan", stt["provider"])
	assert.Equal(t, []any{"audio"}, stt["capabilities"])
	assert.EqualValues(t, 1, stt["slots_count"], "the voice_stt slot")
	assert.Equal(t, false, seed["configured"], "no provider key in the test env")
	assert.Equal(t, []any{"chat", "vision", "audio", "json"}, r.Data()["capabilities"])

	r = c.JSON(fiber.MethodPost, "/ai/models", map[string]any{"code": "Bad Code", "provider": "openai",
		"upstream_model": "", "title": "", "capabilities": []any{"telepathy"}, "input_price_milli": -1})
	require.Equal(t, 422, r.Status)
	for _, f := range []string{"code", "provider", "upstream_model", "title", "capabilities.0", "input_price_milli"} {
		assert.Contains(t, r.Errors(), f)
	}
	r = c.JSON(fiber.MethodPost, "/ai/models", map[string]any{"code": "x", "provider": "arvan",
		"upstream_model": "m", "title": "M", "capabilities": []any{}})
	require.Equal(t, 422, r.Status, "at least one capability")
	assert.Contains(t, r.Errors(), "capabilities")

	r = c.JSON(fiber.MethodPost, "/ai/models", map[string]any{"code": "gpt-mini", "provider": "arvan",
		"upstream_model": "gpt-4o-mini", "title": "GPT mini", "capabilities": []any{"json", "chat", "chat"},
		"input_price_milli": 150, "output_price_milli": 600})
	require.Equal(t, 201, r.Status, r.Body)
	m := r.Obj("model")
	assert.Equal(t, []any{"chat", "json"}, m["capabilities"], "deduplicated, canonical order")
	assert.Equal(t, true, m["is_active"])
	assert.EqualValues(t, 0, m["audio_price_milli"])
	mid := num(m["id"])
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'ai_model.create'"))

	r = c.JSON(fiber.MethodPost, "/ai/models", map[string]any{"code": "gpt-mini", "provider": "arvan",
		"upstream_model": "gpt-4o-mini", "title": "again", "capabilities": []any{"chat"}})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "code")
	assert.Contains(t, r.Errors(), "upstream_model", "UNIQUE(provider, upstream_model)")

	r = c.JSON(fiber.MethodPut, "/ai/models/"+id(mid), map[string]any{"provider": "arvan", "upstream_model": "gpt-4o-mini",
		"title": "GPT-4o mini", "capabilities": []any{"chat"}, "output_price_milli": 700})
	require.Equal(t, 200, r.Status, r.Body)
	assert.EqualValues(t, 700, r.Obj("model")["output_price_milli"])
	assert.EqualValues(t, 150, r.Obj("model")["input_price_milli"], "absent price keeps its value")

	// Assign it to the assistant slot (chat), then removing chat is refused and delete is refused.
	r = c.JSON(fiber.MethodPut, "/ai/features/"+id(e.featureID("assistant")), map[string]any{"credit_cost": 2,
		"model_id": mid})
	require.Equal(t, 200, r.Status, r.Body)
	r = c.JSON(fiber.MethodPut, "/ai/models/"+id(mid), map[string]any{"provider": "arvan", "upstream_model": "gpt-4o-mini",
		"title": "GPT-4o mini", "capabilities": []any{"json"}})
	require.Equal(t, 422, r.Status)
	assert.Equal(t, "ai_capability_in_use", r.Code())
	assert.Equal(t, []any{"assistant"}, r.Body["slots"])
	r = c.JSON(fiber.MethodDelete, "/ai/models/"+id(mid), nil)
	require.Equal(t, 422, r.Status)
	assert.Equal(t, "ai_model_in_use", r.Code())
	assert.Equal(t, []any{"assistant"}, r.Body["slots"])

	r = c.JSON(fiber.MethodPut, "/ai/features/"+id(e.featureID("assistant")), map[string]any{"credit_cost": 1,
		"model_id": e.modelID("gemini-flash")})
	require.Equal(t, 200, r.Status, r.Body)
	r = c.JSON(fiber.MethodDelete, "/ai/models/"+id(mid), nil)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM ai_models WHERE id = ?", mid))
}

func TestFeatureAssignment(t *testing.T) {
	e := newEnv(t, nil)
	c := e.As(admintest.SuperID)
	gemini := e.modelID("gemini-flash")
	chatOnly := num(c.JSON(fiber.MethodPost, "/ai/models", map[string]any{"code": "chat-only", "provider": "arvan",
		"upstream_model": "chat-only", "title": "Chat only", "capabilities": []any{"chat"}}).Obj("model")["id"])
	off := num(c.JSON(fiber.MethodPost, "/ai/models", map[string]any{"code": "off", "provider": "arvan",
		"upstream_model": "off", "title": "Off", "capabilities": []any{"chat", "vision", "audio", "json"},
		"is_active": false}).Obj("model")["id"])

	r := c.Get("/ai/features")
	require.Equal(t, 200, r.Status, r.Body)
	require.Len(t, r.Items(), 5)
	byKey := map[string]map[string]any{}
	for _, it := range r.Items() {
		row := it.(map[string]any)
		byKey[row["key"].(string)] = row
	}
	// goose 00127: the internal speech slot of voice logging defaults to GPT-4o Transcribe, Gemini as fallback.
	stt := byKey["voice_stt"]
	require.NotNil(t, stt)
	assert.Equal(t, "gpt-4o-transcribe", stt["model"].(map[string]any)["code"])
	assert.Equal(t, "gemini-flash", stt["fallback_model"].(map[string]any)["code"])
	assert.Equal(t, []any{"audio"}, stt["needs"])
	lab := byKey["lab_analysis"]
	assert.Equal(t, []any{"vision", "json"}, lab["needs"])
	assert.Equal(t, true, lab["declared"])
	assert.Equal(t, true, lab["usable"])
	assert.Equal(t, []any{float64(gemini)}, lab["usable_model_ids"], "chat-only lacks vision, off is inactive")
	assert.Len(t, r.Data()["models"], 4, "the two seeded models and the two made above")

	labID := e.featureID("lab_analysis")
	r = c.JSON(fiber.MethodPut, "/ai/features/"+id(labID), map[string]any{"credit_cost": 5, "model_id": chatOnly})
	require.Equal(t, 422, r.Status)
	assert.Equal(t, "ai_model_lacks_capability", r.Code())
	assert.Equal(t, []any{"vision", "json"}, r.Body["missing"])
	assert.Equal(t, "model_id", r.Body["field"])

	r = c.JSON(fiber.MethodPut, "/ai/features/"+id(labID), map[string]any{"credit_cost": 5, "fallback_model_id": off})
	require.Equal(t, 422, r.Status)
	assert.Equal(t, "ai_model_inactive", r.Code())

	r = c.JSON(fiber.MethodPut, "/ai/features/"+id(labID), map[string]any{"credit_cost": 5, "fallback_model_id": gemini})
	require.Equal(t, 422, r.Status)
	assert.Equal(t, "ai_fallback_same_as_model", r.Code())

	r = c.JSON(fiber.MethodPut, "/ai/features/"+id(labID), map[string]any{"credit_cost": 100001, "model_id": 999999})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "credit_cost")

	r = c.JSON(fiber.MethodPut, "/ai/features/"+id(labID), map[string]any{"credit_cost": 5, "model_id": 999999})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "model_id", "unknown model")

	r = c.JSON(fiber.MethodPut, "/ai/features/"+id(labID), map[string]any{"credit_cost": 7, "is_active": false,
		"title": map[string]any{"fa": "تحلیل آزمایش ۲"}})
	require.Equal(t, 200, r.Status, r.Body)
	f := r.Obj("feature")
	assert.EqualValues(t, 7, f["credit_cost"])
	assert.Equal(t, false, f["usable"])
	assert.EqualValues(t, gemini, f["model_id"], "absent model keeps its value")
	assert.Equal(t, map[string]any{"fa": "تحلیل آزمایش ۲"}, f["title"])

	r = c.JSON(fiber.MethodPut, "/ai/features/"+id(labID), map[string]any{"credit_cost": 7, "model_id": nil, "is_active": true})
	require.Equal(t, 200, r.Status, r.Body)
	assert.Nil(t, r.Obj("feature")["model_id"])
	assert.Equal(t, false, r.Obj("feature")["usable"], "no usable model")

	// The running catalogue (what the gate reads) sees the change after the publish.
	feat, ok, err := credits.NewCatalog(e.DB).Feature(t.Context(), ai.FeatureLabAnalysis)
	require.NoError(t, err)
	require.True(t, ok)
	assert.EqualValues(t, 7, feat.Cost)
	assert.False(t, feat.Usable())

	// A slot no longer declared in code is listed, flagged, and takes any model.
	e.Exec("INSERT INTO ai_features (`key`, title, credit_cost, is_active, sort_order, created_at, updated_at) VALUES ('retired', '{}', 1, 1, 9, NOW(), NOW())")
	r = c.Get("/ai/features")
	last := r.Items()[len(r.Items())-1].(map[string]any)
	assert.Equal(t, "retired", last["key"])
	assert.Equal(t, false, last["declared"])
	assert.Equal(t, false, last["usable"])
}

func TestSync(t *testing.T) {
	var auth string
	listing := `{"data":[{"id":"gpt-4o-mini"},{"id":"Llama 3.1 / 70B"},{"id":"llama-3-1-70b"},{"id":""}]}`
	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(listing))
	}))
	t.Cleanup(srv.Close)
	e := newEnv(t, srv)
	c := e.As(admintest.SuperID)

	r := c.Get("/ai/models")
	assert.Equal(t, []any{"arvan"}, r.Data()["configured_providers"])

	r = c.JSON(fiber.MethodPost, "/ai/models/sync", map[string]any{"provider": "gemini"})
	require.Equal(t, 422, r.Status)
	assert.Equal(t, "ai_provider_not_configured", r.Code())
	r = c.JSON(fiber.MethodPost, "/ai/models/sync", map[string]any{"provider": "nope"})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "provider")

	r = c.JSON(fiber.MethodPost, "/ai/models/sync", map[string]any{"provider": "arvan"})
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, "apikey test-key", auth)
	added := r.Data()["added"].([]any)
	require.Len(t, added, 3)
	codes := []string{}
	for _, a := range added {
		codes = append(codes, a.(map[string]any)["code"].(string))
	}
	assert.Equal(t, []string{"llama-3-1-70b", "gpt-4o-mini", "llama-3-1-70b-2"}, codes, "slugged, made unique (gateway order is sorted)")
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM ai_models WHERE provider = 'arvan' AND is_active = 1 AND code <> 'gpt-4o-transcribe'"), "inactive")
	assert.Equal(t, `["chat"]`, e.String("SELECT capabilities FROM ai_models WHERE upstream_model = 'gpt-4o-mini'"))
	assert.Equal(t, 0, e.Int("SELECT input_price_milli + output_price_milli FROM ai_models WHERE upstream_model = 'gpt-4o-mini'"))

	r = c.JSON(fiber.MethodPost, "/ai/models/sync", map[string]any{"provider": "arvan"})
	require.Equal(t, 200, r.Status, r.Body)
	assert.Empty(t, r.Data()["added"])
	assert.EqualValues(t, 3, r.Data()["existing"])

	status = http.StatusNotFound
	r = c.JSON(fiber.MethodPost, "/ai/models/sync", map[string]any{"provider": "arvan"})
	require.Equal(t, 422, r.Status)
	assert.Equal(t, "ai_provider_unavailable", r.Code())
	assert.NotContains(t, string(r.Raw), "test-key")
}

func TestCreditPacks(t *testing.T) {
	e := newEnv(t, nil)
	c := e.As(admintest.SuperID)

	r := c.Get("/ai/credit-packs")
	require.Equal(t, 200, r.Status, r.Body)
	require.Len(t, r.Items(), 3)

	r = c.JSON(fiber.MethodPost, "/ai/credit-packs", map[string]any{"code": "credits_50", "title": map[string]any{},
		"credits": 0, "price_rials": 10})
	require.Equal(t, 422, r.Status)
	for _, f := range []string{"code", "credits", "price_rials"} {
		assert.Contains(t, r.Errors(), f)
	}

	r = c.JSON(fiber.MethodPost, "/ai/credit-packs", map[string]any{"code": "credits_1000",
		"title": map[string]any{"fa": "۱۰۰۰ اعتبار", "en": "1000 credits"}, "badge": map[string]any{"fa": "بصرفه"},
		"credits": 1000, "price_rials": 6900000})
	require.Equal(t, 201, r.Status, r.Body)
	p := r.Obj("pack")
	assert.EqualValues(t, 4, p["sort_order"])
	assert.EqualValues(t, 6900, p["price_per_credit_rials"])
	pid := num(p["id"])
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM plus_admin_actions WHERE action = 'credit_pack.create'"))

	r = c.JSON(fiber.MethodPut, "/ai/credit-packs/"+id(pid), map[string]any{"title": map[string]any{"fa": "هزار"},
		"credits": 1200, "price_rials": 6900000})
	require.Equal(t, 200, r.Status, r.Body)
	assert.EqualValues(t, 1200, r.Obj("pack")["credits"])
	assert.Equal(t, map[string]any{"fa": "بصرفه"}, r.Obj("pack")["badge"], "absent badge keeps its value")

	r = c.JSON(fiber.MethodDelete, "/ai/credit-packs/"+id(pid), nil)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, true, r.Data()["deleted"])

	// A pack an invoice references is deactivated instead.
	u := e.user("09121230002")
	pack := uint64(e.Int("SELECT id FROM ai_credit_packs WHERE code = 'credits_150'")) //nolint:gosec // test ids
	e.Exec(`INSERT INTO plus_invoices (reference, user_id, kind, credit_pack_id, credits, duration_months, status, currency,
		subtotal_rials, discount_rials, vat_rate_bps, vat_rials, total_rials, expires_at, created_at, updated_at)
		VALUES ('RPPACK01', ?, 'credit_pack', ?, 150, 0, 'pending', 'IRR', 1290000, 0, 0, 0, 1290000, NOW(), NOW(), NOW())`, u, pack)
	r = c.JSON(fiber.MethodDelete, "/ai/credit-packs/"+id(pack), nil)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, true, r.Data()["deactivated"])
	assert.Equal(t, 0, e.Int("SELECT is_active + is_highlighted FROM ai_credit_packs WHERE id = ?", pack))
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM plus_admin_actions WHERE action = 'credit_pack.deactivate'"))
}

func TestAdjustUserCredits(t *testing.T) {
	e := newEnv(t, nil)
	c := e.As(admintest.SuperID)
	u := e.user("09121230003")
	path := "/users/" + id(u) + "/ai-credits"

	r := c.Get(path)
	require.Equal(t, 200, r.Status, r.Body)
	bal := r.Obj("balance")
	assert.Equal(t, false, bal["active"])
	assert.EqualValues(t, 0, bal["extra"])

	for _, body := range []map[string]any{{"delta": 0, "note": "x"}, {"delta": 5}, {"delta": 2000000, "note": "x"},
		{"delta": 5, "note": "   "}} {
		r = c.JSON(fiber.MethodPost, path+"/adjust", body)
		assert.Equal(t, 422, r.Status, body)
	}

	r = c.JSON(fiber.MethodPost, path+"/adjust", map[string]any{"delta": 50, "note": "Support gift #123"})
	require.Equal(t, 200, r.Status, r.Body)
	assert.EqualValues(t, 50, r.Data()["granted"])
	assert.NotNil(t, r.Data()["grant_id"])
	cr := r.Data()["credits"].(map[string]any)
	assert.EqualValues(t, 50, cr["balance"].(map[string]any)["extra"])
	assert.Equal(t, true, cr["balance"].(map[string]any)["extra_locked"], "no subscription: kept, locked")
	grants := cr["grants"].([]any)
	require.Len(t, grants, 1)
	assert.Equal(t, "admin", grants[0].(map[string]any)["kind"])
	assert.Equal(t, "Support gift #123", grants[0].(map[string]any)["note"])

	// A period bucket is never touched by a deduction.
	at := time.Now().In(civildate.Tehran)
	e.Exec(`INSERT INTO plus_trials (user_id, started_at, ends_at, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())`,
		u, at.Add(-time.Hour), at.Add(48*time.Hour))
	r = c.Get(path)
	require.Equal(t, 200, r.Status, r.Body)
	assert.EqualValues(t, 20, r.Obj("balance")["plan_total"], "the trial bucket (trial_ai_credits seed)")

	r = c.JSON(fiber.MethodPost, path+"/adjust", map[string]any{"delta": -80, "note": "Abuse"})
	require.Equal(t, 200, r.Status, r.Body)
	assert.EqualValues(t, 50, r.Data()["taken"], "only the extra credits")
	cr = r.Data()["credits"].(map[string]any)
	assert.EqualValues(t, 0, cr["balance"].(map[string]any)["extra"])
	assert.EqualValues(t, 20, cr["balance"].(map[string]any)["plan_remaining"])
	adj := cr["adjustments"].([]any)
	require.Len(t, adj, 2)
	assert.Contains(t, e.String("SELECT details FROM plus_admin_actions WHERE action = 'credits.adjust' ORDER BY id DESC LIMIT 1"),
		`"taken":50`)
	assert.Equal(t, 2, e.Int("SELECT COUNT(*) FROM plus_admin_actions WHERE action = 'credits.adjust' AND user_id = ? AND days IS NULL", u))
	assert.Equal(t, 2, e.Int("SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'plus.credits.adjust' AND target_id = ?", u))
}

func TestPlanCreditsAndTrialSetting(t *testing.T) {
	e := newEnv(t, nil)
	c := e.As(admintest.SuperID)

	r := c.Get("/plus/plans")
	require.Equal(t, 200, r.Status, r.Body)
	assert.EqualValues(t, 100, r.Items()[0].(map[string]any)["ai_credits"], "plus_1m seed")

	r = c.JSON(fiber.MethodPost, "/plus/plans", map[string]any{"code": "plus_12m", "title": map[string]any{"fa": "۱۲"},
		"duration_months": 12, "price_rials": 6900000, "ai_credits": 2000000})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "ai_credits")

	r = c.JSON(fiber.MethodPost, "/plus/plans", map[string]any{"code": "plus_12m", "title": map[string]any{"fa": "۱۲"},
		"duration_months": 12, "price_rials": 6900000, "ai_credits": 1200})
	require.Equal(t, 201, r.Status, r.Body)
	pid := num(r.Obj("plan")["id"])
	assert.EqualValues(t, 1200, r.Obj("plan")["ai_credits"])

	r = c.JSON(fiber.MethodPut, "/plus/plans/"+id(pid), map[string]any{"title": map[string]any{"fa": "۱۲"},
		"duration_months": 12, "price_rials": 6900000})
	require.Equal(t, 200, r.Status, r.Body)
	assert.EqualValues(t, 1200, r.Obj("plan")["ai_credits"], "absent keeps its value")
	r = c.JSON(fiber.MethodPut, "/plus/plans/"+id(pid), map[string]any{"title": map[string]any{"fa": "۱۲"},
		"duration_months": 12, "price_rials": 6900000, "ai_credits": 0})
	require.Equal(t, 200, r.Status, r.Body)
	assert.EqualValues(t, 0, r.Obj("plan")["ai_credits"])
	assert.Contains(t, e.String("SELECT details FROM plus_admin_actions WHERE action = 'plan.update' ORDER BY id DESC LIMIT 1"),
		`"ai_credits":{"from":1200,"to":0}`)

	r = c.Get("/plus/settings")
	assert.EqualValues(t, 20, r.Obj("settings")["trial_ai_credits"])
	r = c.JSON(fiber.MethodPut, "/plus/settings", map[string]any{"trial_offer_percent": 0, "trial_ai_credits": 100001})
	require.Equal(t, 422, r.Status)
	r = c.JSON(fiber.MethodPut, "/plus/settings", map[string]any{"trial_offer_percent": 0, "trial_ai_credits": 35})
	require.Equal(t, 200, r.Status, r.Body)
	assert.EqualValues(t, 35, r.Obj("settings")["trial_ai_credits"])
	n, err := credits.NewService(e.DB).TrialCredits(t.Context())
	require.NoError(t, err)
	assert.EqualValues(t, 35, n)
}

// GET /ai/usage (internal/ai/usage) also reports the credits users spent, next to the provider cost.
func TestUsageReportCredits(t *testing.T) {
	e := newEnv(t, nil)
	usage.NewAdminHandlers(e.DB, nil, clock.Fixed(time.Date(2026, 11, 12, 12, 0, 0, 0, civildate.Tehran))).Routes(e.Route(), e.Kit)
	u := e.user("09121230004")
	e.Exec(`INSERT INTO ai_credit_spends (user_id, feature, credits, plan_credits, extra_credits, allocations, status, created_at, updated_at) VALUES
		(?, 'assistant', 1, 1, 0, '[]', 'spent', '2026-11-10 10:00:00', '2026-11-10 10:00:00'),
		(?, 'lab_analysis', 5, 2, 3, '[]', 'spent', '2026-11-11 10:00:00', '2026-11-11 10:00:00'),
		(?, 'lab_analysis', 5, 5, 0, '[]', 'refunded', '2026-11-11 11:00:00', '2026-11-11 11:00:00')`, u, u, u)
	e.Exec("INSERT INTO plus_settings (`key`, `value`, created_at, updated_at) VALUES ('ai_cost_usd_irr_rate', '650000', NOW(), NOW())")

	r := e.As(admintest.EditorID).Get("/ai/usage")
	require.Equal(t, 200, r.Status, r.Body)
	assert.Contains(t, r.Data(), "totals", "the existing fields stay")
	cr := r.Obj("credits")
	assert.EqualValues(t, 6, cr["total"], "refunded uses excluded")
	assert.EqualValues(t, 2, cr["uses"])
	assert.EqualValues(t, 3, cr["extra_credits"])
	assert.Len(t, cr["by_day"], 2)
	first := cr["by_feature"].([]any)[0].(map[string]any)
	assert.Equal(t, "lab_analysis", first["feature"])
	assert.EqualValues(t, 650000, r.Data()["usd_irr_rate"])
}

// The new routes map to the capabilities admin-api.md §29 names (not an integration test: no DB needed).
func TestRouteCapabilities(t *testing.T) {
	p := httpadmin.Prefix
	assert.Equal(t, httpadmin.CapConfig, httpadmin.RouteCapability(p+"/ai/models"))
	assert.Equal(t, httpadmin.CapConfig, httpadmin.RouteCapability(p+"/ai/models/sync"))
	assert.Equal(t, httpadmin.CapConfig, httpadmin.RouteCapability(p+"/ai/features/:id"))
	assert.Equal(t, httpadmin.CapConfig, httpadmin.RouteCapability(p+"/ai/usage"))
	assert.Equal(t, httpadmin.CapBilling, httpadmin.RouteCapability(p+"/ai/credit-packs"))
	assert.Equal(t, httpadmin.CapBilling, httpadmin.RouteCapability(p+"/ai/credit-packs/:id"))
	assert.Equal(t, httpadmin.CapBilling, httpadmin.RouteCapability(p+"/users/:id/ai-credits"))
	assert.Equal(t, httpadmin.CapBilling, httpadmin.RouteCapability(p+"/users/:id/ai-credits/adjust"))
	assert.Equal(t, httpadmin.CapUsers, httpadmin.RouteCapability(p+"/users/:id/overview"))
}
