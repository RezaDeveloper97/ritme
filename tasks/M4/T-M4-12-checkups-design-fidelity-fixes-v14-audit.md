---
id: T-M4-12
title: Checkups — design fidelity fixes (v14 audit)
milestone: M4
type: frontend
status: done
depends_on: [T-M4-11]
parallel_group: M4-D
touches: [frontend/src/app/sheets/registry.tsx, frontend/src/screens/checkups,frontend/src/screens/checkup-detail,frontend/src/screens/checkup-history,frontend/src/screens/checkup-self-exam,frontend/src/screens/checkup-mark-done,frontend/src/screens/checkup-custom-form,frontend/src/widgets/checkups-card,frontend/src/features,frontend/src/entities/checkup,frontend/src/app/globals.css,frontend/messages,backend-go/internal/checkups,backend-go/db,backend-go/api/openapi.yaml,backend-go/contract,backend-go/resources/translations,backend-go/internal/i18n/testdata,admin-web/src,docs/checkups]
skills: [verify-all,check-colors]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && cd ../backend-go && go vet ./... && go test ./... && make contract ROUTES=all && cd ../admin-web && npm run typecheck && npm run test
---

# T-M4-12 — Checkups — design fidelity fixes (v14 audit)

## Why
Design fidelity audit (docs/checkups/design-audit.md) compared the implementation with `docs/design/checkups-v14/` and listed deviations by
severity with file:line and a suggested fix.

## Scope
- Fix EVERY high and med item in docs/checkups/design-audit.md, and the low items that are cheap (same file you're already in). For any
  item you decide not to fix, write the reason in the audit table (column "Resolution").
- Include the backend part of B1: the engine must put overdue cycle-based items in the `overdue` section (backend-go/internal/checkups/engine); keep the contract/OpenAPI consistent and add a Go test.
- Include known minor items 5a (attachment chip inside the record card), 5b (detail hero icon per type), 5c (admin-web Latin digits in fa — use the admin's number formatter), 5d (self-exam «موعدش رسیده» vs «۱۹ روز دیگر» contradiction).
- Palette rules (frontend/CLAUDE.md §10.2) win over design colours; layout/content/copy follow the design.
- New/changed copy in fa + en; sync the backend translation seed and i18n goldens (tests enforce it).

## Out of scope
- Clinical content; other features' screens.

## Acceptance
- Audit table updated with a Resolution per row; verify green; light + dark full-page screenshots re-taken locally
  next to the design ones in the audit folder (`*-impl.png`, same names) and visually checked.
