---
id: T-M2-07
title: Enums — PHP→Go generator plus hand-ported enum logic
milestone: M2
type: backend
status: todo
depends_on: [T-M2-04]
parallel_group: M2-C
touches: [backend-go/cmd/enumgen, backend-go/internal/enums]
skills: []
verify: cd backend-go && go generate ./internal/enums/... && git diff --exit-code internal/enums && go test ./internal/enums/...
---

# T-M2-07 — Enums: PHP→Go generator plus hand-ported enum logic

## Why
57 string-backed enums define every allowed value, label and several pieces of cycle logic (domain-inventory §3).
Hand-copying them invites typos; generating the value/label tables and hand-porting only the logic keeps parity.

## Scope
1. `cmd/enumgen`: parses `backend/app/Enums/*.php` and `backend/app/Services/MessageSystem/Enums/*.php` (cases, backed
   values, `label()` match arms for fa/en, `icon`, `options`) and writes `internal/enums/zz_generated_<name>.go`:
   typed string, `Values()`, `IsValid()`, `Label(locale)` (fa → Persian, **any other locale → English** where the PHP
   does that; Persian-only labels stay Persian-only), `Options(locale)`. Keep exact value strings (e.g.
   `long_period_flag`). Enum order = PHP case order.
2. Hand-written logic (`internal/enums/*_logic.go`): `CycleSubphase.FertilityLevelV11/FertilityLevel/Canonical/
   ContentBacked`, `CyclePhase.Subphases`, `MainPhase.LegacyPhase`, `CycleVariability.FromStdDev/UncertaintyRange`,
   `RegularityStatus.FromCycleLengths`, `CycleWarning.RequiresUserInput`, `RecommendationTrigger.Matches/ActiveFor`
   (reads daily-log fields — take an interface), `EnergyLevel.Score`, `BmiCategory.FromBmi`,
   `ResolutionSource.IsPredicted`, `DataStatus.Priority`, `DataQualityFlag.ExcludesFromPrediction`,
   `ReminderType.Icon`, `iconFor/labelFor` helpers.
3. `go:generate` directive so regenerating is one command (the generator reads `../backend`; after T-M2-27 the
   generated files are kept and the generator deleted).

## Out of scope
Using the enums in endpoints.

## Acceptance
- Generated code is committed and reproducible (`verify:` checks no diff).
- A test compares every enum's `Values()` and fa/en labels with a JSON dump produced by a PHP script
  (`php artisan tinker`-free: a small `php -r` loader), committed as testdata.
- Logic methods have table tests mirroring the PHP branches (cite PHP file:line in comments).
