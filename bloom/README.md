# bloom/ — Night & Bloom build queue

A separate task queue (not `tasks/`) for the **Night & Bloom** redesign and every new feature in the canvas
«ریتمی — اقدام به بارداری و بارداری» (https://claude.ai/artifact/LQgpWxuwxq5oEuwmhCYsvf). Driven by the `/bloom-task`
skill (`.claude/skills/bloom-task/SKILL.md`). `/next-task` and `tasks/` are untouched and unaware of this queue.

- Artboards (light `nbl_`, dark `nbd_`, TTC `nb2_`=dark / `v19_`=light): `docs/design/night-bloom/<section>/`,
  each section has a `TEXT.md` with the extracted copy. The canvas page «آرشیو نسخه‌های قدیمی» is excluded.
- Shared reference produced by B-N1-01: `docs/night-bloom/README.md` (tokens, components, screen→route map).

## Decisions (user, 2026-09-30)

| Topic | Decision |
|---|---|
| Platforms | **Web frontend + backend only.** No Android work, builds or checks, ever. |
| Theme | Night & Bloom replaces the current palette; **every screen ships light and dark** (+ follow-system). |
| Scope | Everything in the canvas except the archive page: redesign of existing screens, Plus, companion/family, postpartum/child, analysis, health/AI, doctors, learning, admin v2, extras (ads, shop, insurance, city services, …). |
| External services | Payment, AI (OCR / STT / chat), video call, SMS invites sit behind **adapters with a `fake` provider** (default). Real providers are config-switched later. Café Bazaar / Myket billing is Android-only → out. |
| Instructor panel | Separate app `instructor-web/` → stage `instructor.ritmeapp.ir`, prod `instructor.ritme.app`. |
| Learning entry | Option A: cards on Home + «دوره‌های من» in Me (no conditional tab). |
| Order | N1 foundation → N2 onboarding + Plus → N3 logging/analysis → N4 companion → N5 postpartum/child → N6 health/AI/tools → N7 doctors/assistant → N8 learning/instructor → N9 admin v2 → N10 extras. |
| Milestone end | Fidelity audit (light + dark screenshots vs artboards) → verify-all → **stage** deploy + e2e. Never production without an explicit ask. |

## Commands

| Command | Purpose |
|---|---|
| `bash bloom/bin/next.sh` | runnable tasks now |
| `bash bloom/bin/next.sh --all` | every task + status |
| `bash bloom/bin/next.sh --status B-N1-01 in_progress` | change status (appends to `bloom/LOG.md`) |
| `bash bloom/bin/next.sh --conflicts B-N1-05 B-N1-07` | parallel safety (touches overlap → exit 1) |
| `bash bloom/bin/next.sh --check` | validate ids, deps, statuses |
| `bash bloom/bin/new.sh ID "Title" TYPE "deps" GROUP "touches" "skills" "verify"` | add a task |
| `bash bloom/bin/index.sh` | regenerate `bloom/INDEX.md` |
| `bash bloom/bin/dev-up.sh [api|web|all]` | (re)start local API :8020 (ritme_dev) + web :3000 detached; restart `api` after translation/migration changes |
| `node bloom/bin/shot.mjs --out docs/qa/bloom/<ID> --mobile 09900000001 /fa/home` | full-page light+dark screenshots of the local app (OTP read from `ritme_dev`); `--files <artboard.dc.html…>` renders artboards for side-by-side; header of the file lists options |

Task ids: `B-N<milestone>-<nn>` (follow-ups get a letter suffix, `B-N1-06b`). Frontmatter and body sections are the same
as `tasks/README.md` (Why / Design / Scope / Out of scope / Acceptance).

## Milestones

| | Theme | Tasks |
|---|---|---|
| N1 | Foundation — tokens, primitives, nav, redesign of existing screens (cycle, calendar, me, TTC, pregnancy, reminders, checkups) | 17 |
| N2 | Onboarding v2, 6 life-stage modes, Ritme Plus (plans, payment adapter, trial, gating, admin) | 11 |
| N3 | Logging v2 (taxonomy, quick tiles, body map, customization, voice) + Analysis tab | 14 |
| N4 | Companion «همدم» & family, male path | 10 |
| N5 | Postpartum mode, children (WHO growth, vaccines, milestones), feeding, kick counter, contraction timer | 11 |
| N6 | Vitals, health record + doctor PDF/share link, AI platform, lab analysis, to-do list | 10 |
| N7 | Services hub, doctors & booking & chat, AI assistant clinic, telemedicine admin | 10 |
| N8 | Courses (user side), media pipeline, offline downloads, `instructor-web` app | 10 |
| N9 | Admin v2 — theme, roles/audit, overview, users, messages v2, engine simulator, challenges, KPI, OKR, safety, config | 13 |
| N10 | Extras — ads, shop, home lab, city services, insurance, devices/backup | 8 |
