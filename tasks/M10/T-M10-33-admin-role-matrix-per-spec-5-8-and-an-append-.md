---
id: T-M10-33
title: Admin role matrix per SPEC 5.8 and an append-only, hash-chained audit log
milestone: M10
type: backend
status: done
depends_on: [T-M10-25, T-M10-02]
parallel_group: M10-D
touches: [backend-go/internal/admin/httpadmin/permissions.go, backend-go/internal/admin/httpadmin/audit.go, backend-go/internal/admin/httpadmin/health.go, backend-go/internal/admin/rbac, backend-go/internal/admin/auditchain, backend-go/db/queries/admin, docs/engine-v6/jobs.md, backend-go/internal/admin/store, backend-go/internal/admin/httpadmin/kit.go, backend-go/internal/admin/httpadmin/audit_test.go, backend-go/internal/http/routes_admin_rbac.go, backend-go/internal/admin/content/admintest, backend-go/internal/admin/users/users.go, backend-go/internal/admin/users/v2.go, backend-go/internal/admin/clinicalcfg/api_int_test.go, backend-go/internal/okr/admin_int_test.go, backend-go/internal/cityservices/admin_dir_int_test.go, backend-go/internal/shop/admin/admin_int_test.go, docs/go-migration/admin-api.md]
skills: []
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/admin/... && make test-int PKG="./internal/admin/httpadmin/... ./internal/admin/rbac/..."
---

# T-M10-33 — Admin role matrix per SPEC 5.8 and an append-only, hash-chained audit log
**Model:** opus (security).

## Why
SPEC §5.8 role table and "audit log: every create, edit, approve, publish, export and health-data access; cannot be
deleted". Default matrices are broader than the spec (`permissions.go:106-140`): the analyst sees the user list, the
product manager manages users and has export, engineering manages config where the spec says "technical config, view
thresholds". The audit log is append-only by convention only, and a failed audit insert on a health-data read is
logged and ignored (`audit.go:64-66`). DECISIONS.md D-182.

## Scope
1. Default matrices per SPEC §5.8: product manager (dashboard, KPI, OKR, masked users view, non-clinical messages,
   challenges); medical advisor (clinical approval of messages and thresholds, safety publishing, dashboard view);
   support (masked users, subscriptions, tickets); clinical support (+ the 30-minute health grant); analyst (KPI and
   anonymous export only); marketing (product/marketing messages, challenges, segments); engineering (technical config
   manage, thresholds view). Map each spec row to capabilities in a table in the code comment. Matrices a super admin
   already edited are not overwritten: list differences in PROGRESS.md and ask the user.
2. Hash chain (new `internal/admin/auditchain`, used by `httpadmin.Audit/AuditAs`): inside one short transaction,
   lock `admin_audit_chain_head` (`FOR UPDATE`), compute `row_hash = sha256(last_hash ‖ canonical JSON of admin_id,
   action, target_type, target_id, ip, detail, created_at)` with `created_at` set by the writer (the id is not hashed:
   the row cannot be updated after insert), insert with `prev_hash = last_hash`, move the head. Rows written before the
   migration stay unchanged: a one-off job hashes them oldest first into `genesis_hash` / `genesis_last_id` and the
   chain starts from that hash. `GET /api/admin/v1/audit/verify?from=` (super) walks rows by id and checks every
   `prev_hash` against the previous `row_hash` (a row without hashes after genesis is a break); nightly job
   `admin.audit_verify` raises a metric on a break and writes the head hash to the structured log as an off-database
   anchor.
3. Health-data reads fail closed: if the audit insert fails, the response is 503 and no data is returned.
4. Every export and every message publish/approve writes an audit row (verify, add where missing).

## Tests
- Matrix table per role and capability; marketing cannot create a clinical message (403, with T-M10-02); engineering
  can view and not approve thresholds; analyst gets 403 on the user list.
- UPDATE/DELETE on the audit table → error (trigger from T-M10-25).
- Chain verification passes; a row changed with triggers dropped in the test DB is detected.
- Health read with a failing audit insert → 503, no body data.

## Out of scope
- Admin-web roles screen (T-M10-35).

## Acceptance
- Verify passes; PROGRESS.md cites D-182.
