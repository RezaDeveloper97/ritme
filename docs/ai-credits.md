# AI credits («اعتبار») — design and contract

Decided 2026-11-12. Schema: `backend-go/db/migrations/00125_ai_credits.sql`. Backend: `backend-go/internal/ai/credits`
(ledger + catalogue), `internal/ai/access` (the gate), `internal/ai/route.go` + `openai_compat.go` (model routing),
`internal/plus` (credit packs through the normal checkout). Users only ever see the word «اعتبار» / "credit" —
never "token".

## Rules

1. **AI needs a running subscription** (a Plus period or the trial). Without one no AI feature works — not even
   with extra credits.
2. **A subscription period carries its own credits** (`plus_plans.ai_credits`, the whole period at once; the trial's
   come from the setting `trial_ai_credits`). They are usable only while that period runs and are gone with it.
   The next period starts from its plan's amount again; nothing carries over.
3. **Extra credits never expire and are never zeroed.** They come from credit packs (`ai_credit_packs`, bought any
   time) and admin gifts. When the subscription ends they are *locked*, not lost; the next subscription unlocks them.
4. **Spending order:** the period's credits first, then the extra ones.
5. **One use of a feature costs a fixed number of credits** (`ai_features.credit_cost`), whatever the token count.
   Tokens and the provider cost are still logged per call (`ai_usage_logs`) for the admin's margin report.
6. Credits are **reserved before** the provider runs and **refunded** when the call fails.

A period without a plan (admin / challenge gift) has no credits of its own; the admin can gift extra credits.

## Models and features

- `ai_models`: the catalogue. `provider` is `gemini` or `arvan` (ArvanCloud AI gateway, OpenAI-compatible:
  `POST {ARVAN_AI_BASE_URL}/chat/completions`, `Authorization: apikey <ARVAN_AI_API_KEY>`). `upstream_model` is the
  name sent upstream. `capabilities` ⊆ `chat | vision | audio | json`. Prices are milli-USD per 1M tokens.
  **API keys live only in the server env**, never in the database or the admin panel.
- `ai_features`: one row per feature slot. The key is declared in code with
  `ai.RegisterFeature(key, capabilities…)`; the row is created at start-up (`credits.Catalog.Sync`). The admin sets
  `model_id`, `fallback_model_id`, `credit_cost`, `is_active`. A model may be assigned only if it has every
  capability the slot needs.
- Existing slots: `assistant` (chat), `voice_log` (json), `lab_analysis` (vision + json),
  `doc_extract` (vision + json).
- Internal slots (`ai.RegisterInternalFeature`): a model another feature calls into. `voice_stt` (audio) is the
  speech-to-text of voice logging, so a speech-only model and a text model can share one use. The admin assigns
  its model like any slot; it is never listed in `GET /ai/credits`, has no route and is never charged — the use
  is charged once, on `voice_log`. The Arvan gateway serves it through `POST {base}/audio/transcriptions`.
  Its seeded default (goose 00127) is GPT-4o Transcribe on the Arvan gateway with Gemini Flash as fallback; like
  every model it stays editable in the panel (check `upstream_model` against the gateway's own name, and the prices).
- `AI_PROVIDER=fake` (dev, tests, stage) replaces every model with deterministic fixtures.

## The gate (backend) — the only way to call a model

```go
// feature package
var FeatureFoo = ai.RegisterFeature("foo", ai.CapChat)

// routes_<domain>.go
platform := newAIPlatform(d, nil)
r.Post("/api/v1/foo", locale, append(append([]any{guard}, platform.guard.Chain(FeatureFoo)...), h.Foo)...)

// handler
stream, err := client.Chat(c.Context(), FeatureFoo, req)   // or Transcribe / ParseLog / Extract
if err != nil { return access.Error(err, locale) }
```

`Chain` = throttles → feature on + subscription + enough credits (no spend yet) → consent, cost caps, concurrency →
reserve the credits → handler → automatic refund when the handler returns an error or a status ≥ 400.
`Gate` is the same without a cost (subscription still required). A feature needs a `Policy` in
`access.policies` (consent code + throttles). Work that runs after the request stores
`access.ReservationFrom(ctx).At()` and refunds a failed job with `credits.Service.RefundAt`.

### Gate errors (all JSON `{success:false, message, error_code, …}`; never 401)

| HTTP | `error_code` | Extra fields | Meaning |
|---|---|---|---|
| 402 | `ai_subscription_required` | `feature`, `extra_credits` | no running subscription / trial |
| 402 | `ai_credits_exhausted` | `feature`, `required`, `balance`, `plan_remaining`, `extra_credits`, `period_ends_at` | credits do not cover the cost |
| 403 | `ai_feature_disabled` | `feature` | slot switched off or without a usable model |
| 403 | `consent_required` | `consent`, `version`, `reason` | consent not accepted |
| 429 | `ai_busy`, `ai_user_budget_exhausted`, `too_many_requests` | `retry_after`? | try later |
| 503 | `ai_unavailable`, `ai_failed`, `ai_budget_exhausted` | | provider down / global cap |
| 422 | `ai_invalid_request` | | request breaks a platform limit |

The retired codes `plus_required` / `plus_quota_exceeded` are no longer returned by AI routes (they still exist
for non-AI Plus features such as `plus.companion`).

## User API (`/api/v1`, auth, localized)

### `GET /ai/credits`
```json
{"success":true,"data":{
  "can_use": true,
  "period": "plus",                     // "plus" | "trial" | null
  "period_ends_at": "2026-12-12T10:00:00+03:30",   // null when can_use is false
  "plan":  {"total": 100, "used": 12, "remaining": 88},
  "extra": {"remaining": 50, "locked": false},      // locked = kept, but no subscription to spend them under
  "spendable": 138,                     // 0 when can_use is false
  "features": [
    {"key":"assistant","title":"دستیار سلامت","credit_cost":1,"available":true,"affordable":true}
  ]
}}
```

### `GET /ai/credits/usage?before=<id>`
```json
{"success":true,"data":{
  "days": 30, "since": "…", "total": 17,
  "by_feature": [{"feature":"assistant","title":"دستیار سلامت","uses":12,"credits":12}],
  "by_day":     [{"date":"2026-11-10","credits":5}],
  "history":    {"items":[{"id":91,"feature":"lab_analysis","title":"تحلیل آزمایش","credits":5,"at":"…"}],"next":61}
}}
```
`next` is the `before` of the following page, `null` at the end. Refunded uses never appear.

### Credit packs (plus domain)
- `GET /plus/credit-packs` (public): `{currency:"IRR", vat_rate_bps, packs:[{id, code, title, badge, credits, price, is_highlighted}]}` — prices in rials.
- `POST /plus/credit-packs/checkout {pack_id, preview?}` → preview: `{quote:{pack, currency, subtotal, vat_rate_bps, vat, total}}`;
  otherwise `{invoice, payment:{gateway, authority, redirect_url}}` — the same shape as `POST /plus/checkout`.
  Blocked for teen accounts (`403 teen_commercial_blocked`). Unknown pack → 422 on `pack_id`.
- The payment returns through the existing `/plus/return` page and is settled by the existing
  `POST /plus/verify {reference, authority, status?}`; a pack invoice then grants its credits (idempotent).
- Every invoice object now also has `kind` (`plan` | `credit_pack`), `credits` (pack invoices) and `credit_pack`.
- Plans (`GET /plus/plans`, plan refs) now have `ai_credits`.
- `GET /plus/status` → `entitlements` no longer lists `plus.voice_log`, `plus.lab_ai`,
  `plus.assistant_unlimited`, `plus.doc_ai`.
- `GET /assistant` → `quota` is now `{unlimited:false, allowed, limit, used, remaining}` from credits.

## Admin API (`/api/admin/v1`) — see `docs/go-migration/admin-api.md`

§29 (+ §15 / §17 there): models CRUD (+ sync from the Arvan gateway), feature slots (model, fallback, cost, on/off), credit packs CRUD,
`ai_credits` on plans, `trial_ai_credits` in Plus settings, a user's balance + manual gift / deduction, and the
usage report.

## Frontend gate

Every AI surface goes through `features/ai-gate` (`useAiGate(featureKey)` / `<AiGate>`); a 402/403 of the table
above is never handled by hand. See `frontend/CLAUDE.md` → «AI gate».
