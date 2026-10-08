---
id: B-N8-02b
title: Media security fixes — atomic quota, disk reserve, stuck processing sweep, stream headers, prod log hygiene
milestone: N8
type: backend
status: done
depends_on: [B-N8-02]
parallel_group: N8-B2
touches: [backend-go/internal/media,backend-go/db,deploy]
skills: [security-review]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N8-02b — Media security fixes — atomic quota, disk reserve, stuck processing sweep, stream headers, prod log hygiene

## Why

## Scope
Security audit of B-N8-02 (internal/media):
- M1 Create: count active uploads + sum bytes + insert in one tx with `SELECT … FROM learning_instructors WHERE id=? FOR
  UPDATE` (parallel POSTs can't exceed 5 active / 100 GiB); race test.
- M2 disk reserve: `Statfs(root)` refuse when free space after the upload < reserve (10 GiB or 10%); global sum cap;
  document a separate filesystem for `MEDIA_STORAGE_PATH`.
- L1 `DefaultURLTTL` 5–10 min (player re-mints on 410).
- L2 add `location ^~ /api/v1/media/ { access_log off; … }` to deploy/vhost-api.inc + go-routes.inc (prod, for cutover).
- L3 stream responses: `Content-Security-Policy: sandbox; default-src 'none'`, `Cross-Origin-Resource-Policy:
  same-site`; tighter WebM DocType check.
- L4 sweep rows stuck in `processing` > 1 h → failed + removed; `Create` clears processing priors.
- L5 document single-replica assumption (or DB lock in remove()).

## Out of scope
- 

## Acceptance
- 
