---
id: T-M2-33
title: Deploy ordering so SSG pages bake the new messages
milestone: M2
type: release
status: done
depends_on: [T-M2-25]
parallel_group: M2-J
touches: [deploy-stage.sh,deploy.sh,frontend/Dockerfile,docker-compose.stage.yml,docker-compose.prod.yml,.claude/skills/deploy-stage,.claude/skills/deploy,docs/go-migration]
skills: [deploy-stage]
verify: bash -n deploy-stage.sh && bash -n deploy.sh
---

# T-M2-33 — Deploy ordering so SSG pages bake the new messages

## Why
Stage redeploy 2026-09-28 (docs/go-migration/stage-rollout-log.md § Stage redeploy + fix smoke, bug B1): `/[locale]/*`
pages are statically generated and fetch UI messages from the live API at build time (`shared/i18n/messages.ts`,
`app/[locale]/layout.tsx`); runtime bundle wins over bundled JSON. `deploy-stage.sh` builds the frontend while the
OLD backend-go still serves, then starts the new one, so changed copy stays stale in the HTML; a cached rebuild
doesn't help (Docker layer cache). Fixed on stage by hand with `build --no-cache frontend`.

## Scope
1. `deploy-stage.sh`: build + start backend-go first and wait for it healthy (and goose done), THEN build the frontend
   so its SSG step reads the new messages; make the frontend's `npm run build` layer re-run on every deploy (e.g. a
   `BUILD_REV`/commit build arg consumed right before the build step — not a blanket `--no-cache` of every layer).
2. Apply the same ordering in `deploy.sh` for when prod runs Go (keep today's Laravel prod behaviour working — no
   change in what prod runs; do NOT run deploy.sh).
3. Update the `deploy-stage` and `deploy` skills' docs with the ordering and why.
4. Run `./deploy-stage.sh` once (staging only — the user authorized staging deploys on 2026-09-28) and prove the
   frontend build step re-ran and pages show current copy (e.g. the fa slide label «اسلاید ۱ از ۳» in SSR HTML).

## Out of scope
- Making the pages dynamic/ISR (bigger change).
- Any production deploy.

## Acceptance
- Stage deployed with the new script; evidence (build log tail showing the frontend build step not CACHED, SSR HTML
  check) appended to docs/go-migration/stage-rollout-log.md; prod app containers not recreated.
