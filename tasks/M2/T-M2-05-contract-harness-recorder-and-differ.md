---
id: T-M2-05
title: Contract harness — golden recorder and JSON-aware differ
milestone: M2
type: backend
status: todo
depends_on: [T-M2-01, T-M2-02]
parallel_group: M2-B
touches: [backend-go/cmd/contract, backend-go/contract]
skills: []
verify: cd backend-go && go test ./cmd/contract/... && make contract-record ROUTES=public && make contract ROUTES=health
---

# T-M2-05 — Contract harness: golden recorder and JSON-aware differ

## Why
"Exactly compatible" has to be checked by a machine, not by eye. The harness records Laravel's responses for every
route × persona × locale with a fixed clock (T-M2-01) and diffs the Go responses against them. Every endpoint task's
Acceptance says "contract diff green for its routes". See README → Contract, api-inventory §7.

## Scope
1. `contract/cases/<group>.yaml`: case list per route group (health, public, auth, content, profile, reminders,
   healthlog, cycle, period, pregnancy, messages, home, notifications): method, path, query, body, persona,
   `Accept-Language` (none / fa / en / ar), `X-Test-Now`, expected status. Cover all 74 routes, including error paths
   (422 summaries, 404 unknown route / missing model, 405, 401 with each `error_code`, 429 on the throttled routes)
   and write sequences (a case can be a list of steps; the DB is reset per case or per sequence).
   Plus an **engine sweep** group `cycle-sweep`: for every persona, `/cycle/date/{d}` for 60 consecutive days around
   `CONTRACT_TODAY` and `/cycle/month/{y}/{m}` (full + calendar) for 3 months, in fa and en. T-M2-13/14 use these
   as their golden oracle.
2. `cmd/contract record`: boots against the T-M2-01 Laravel (127.0.0.1:8090); logs each persona in via
   send-otp → read the code from the DB → verify-otp; stores `contract/golden/<group>/<case>.json`
   (status, selected headers, body). Also exports the seeded DB as `contract/fixtures/dump.sql` so Go tests can load
   the same data without PHP.
3. `cmd/contract diff`: runs the same cases against a Go server (started by the harness against a DB loaded from
   `dump.sql`; the Passport key pair from the contract env is copied into `contract/fixtures/keys/`, test-only).
   Normalisation: parse both bodies as JSON (key order ignored, `\/` and `\uXXXX` equivalent), but **types are strict**
   (`"65.50"` ≠ `65.5`, `[]` ≠ `{}` ≠ `null`). Volatile fields (tokens, `jti`, created ids, `created_at` of rows
   created in the case) are matched by pattern via a per-case `ignore`/`pattern` list. Pretty vs compact whitespace is
   compared only where `strict_bytes: true` (framework error bodies).
4. `contract/allowlist/<group>.yaml` (one file per group so parallel domain tasks don't collide): documented
   exceptions; each entry must reference a `deviations.md` id.
5. Makefile: `make contract-record ROUTES=<group|all>`, `make contract ROUTES=<group|all>` (exit 1 on diff, prints a
   readable JSON path diff).

## Out of scope
Implementing endpoints. Load testing (T-M2-24).

## Acceptance
- Goldens for **all** groups are recorded and committed (they are the spec for T-M2-06…T-M2-18).
- Recording twice produces byte-identical goldens (determinism check in CI: `make contract-record && git diff --exit-code`).
- The differ has unit tests for normalisation (type strictness, `[]` vs `{}`, key order, escaping, patterns).
- `make contract ROUTES=health` is green against Go's `/up`.
