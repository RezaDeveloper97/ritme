# Ritme — repository guide

Each app has its own rules; read the one for the code you touch:

| Path | Guide |
|---|---|
| `backend-go/` | `backend-go/CLAUDE.md` — THE backend (Go, Fiber v3) |
| `frontend/` | `frontend/CLAUDE.md` — user web app (Next.js, FSD) |
| `admin-web/` | `docs/go-migration/admin-web.md` + `docs/go-migration/admin-api.md` |
| `backend/` | `backend/CLAUDE.MD` — frozen Laravel, production maintenance only |
| `laravel-site/` | `laravel-site/CLAUDE.md` — marketing site |
| `application/` | `application/CLAUDE.md` — Android |

## AI features: always through the AI gate

Ritme sells AI as **credits («اعتبار»)**: a subscription period carries credits that end with it, extra credits
(packs) never expire, period credits are spent first, and nothing can be spent without a running subscription.
Design, rules and the API contract: **`docs/ai-credits.md`**.

Any code that calls an AI model, or any screen that starts an AI action, MUST go through the gates — never a
hand-written check, never a direct provider call:

- **Backend:** `ai.RegisterFeature` + `platform.guard.Chain(feature)` + `*ai.Client`
  → `backend-go/CLAUDE.md` § *AI gate*.
- **Frontend:** `useAiGate(featureKey)` / `<AiGate>` from `features/ai-gate`
  → `frontend/CLAUDE.md` § *AI gate*.
- **Admin:** models, per-feature model + credit cost, credit packs and plan credits are admin data
  (admin panel → هوش مصنوعی); never hard-code a model, a price or a cost.

The gates are what keep "no subscription", "out of credits" and "feature off" a managed sheet instead of a raw
error, so a new AI feature that skips them is a bug.
