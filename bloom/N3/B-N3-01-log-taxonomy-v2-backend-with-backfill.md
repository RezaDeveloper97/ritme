---
id: B-N3-01
title: Log taxonomy v2 (backend) with backfill
milestone: N3
type: backend
status: done
depends_on: [B-N2-11]
parallel_group: N3-A
touches: [backend-go/db,backend-go/internal/healthlog,backend-go/internal/enums,backend-go/api,backend-go/resources/translations]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N3-01 — Log taxonomy v2 (backend) with backfill

## Why
The design defines a full logging taxonomy (Log_Taxonomy artboard is the dev reference).

## Design
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Log_Taxonomy.dc.html` (+ `nbd_Log_Taxonomy`)

## Scope
- Categories/params per the taxonomy table and mode columns: bleeding (intensity incl. spotting, colour, clots, odour), pain (locations, 1–10, relief), mood (10), energy/appetite/cravings, sleep (duration band, quality), activity (type, duration, intensity), urinary/genital, libido & sex/protection, skin & hair, measurements (weight, BBT, LH, pregnancy test), meds (from care), note, custom items.
- Backfill existing `health_logs` into the new shape without loss; old endpoints keep working until N3-03 switches.
- Enums endpoint with fa/en labels; OpenAPI; tests.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Backfill test on a copy of seeded data
- `verify` green
