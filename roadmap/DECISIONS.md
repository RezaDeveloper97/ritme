# Canvas-build — user decisions (2026-09-30)

Binding for every `CB-*` task. Source canvas: «ریتمی — یائسگی» (https://claude.ai/artifact/RT9a6Xy7hG9azjjwZnQnBs),
snapshot in `docs/design/canvas-v1/`. All 15 pages reviewed (no "old version archive" page exists in this canvas).

| # | Topic | Decision |
|---|---|---|
| 1 | Scope | All four groups: core health (IA, conditions, contraception, pelvic, privacy), new modes (menopause, IVF, loss, teen), data & record (record, insurance, voice), marketplace (directory + shop incl. web). |
| 2 | Relation to `bloom/` | **Build on top of bloom.** Anything a `B-N*` task delivers (shell/nav, mode switcher, log sheet v2, voice logging, companion, children, AI adapter, labs storage, record summary, report builder + share links, services hub, app lock, city services, shop entry, insurance info) is reused and extended, never rebuilt. CB tasks depend on `B-N*` ids; `roadmap/bin/next.sh` resolves them in `bloom/`. Never edit `bloom/` files from this queue. |
| 3 | Order | IA/nav alignment first, then menopause, then the rest by dependency. |
| 4 | Platforms | Web frontend + backend (+ admin-web) only. **No Android / native app of any kind.** Desktop web boards (`W_*`) are built in the same `frontend/`. |
| 5 | Dropped | Wearables (Health Connect, Apple Health…), disguised app icon, hide-in-recents — native only, removed from the queue. |
| 6 | Colours | The app's current light AND dark theme (Night & Bloom tokens once bloom B-N1-02 is done). Board hex values are not the spec; layout/copy/flow are. |
| 7 | Clinical/list content | Admin-editable in admin-web (catalog `CB-CORE-03/04`, taxonomy, message_contents), fa + en, marked `[needs clinical review]` until reviewed. |
| 8 | Missing designs | Minimal only (postpartum/child, companion, courses, assistant, Plus, task list, PCOS): reuse bloom where it has them, otherwise hide or keep minimal. |
| 9 | Voice | Server-side AI through bloom's AI adapter (Gemini provider, fake by default). Audio never stored. |
| 10 | Record files | Server storage (encrypted, bloom labs storage generalised) + AI extraction (OCR) with user review. |
| 11 | Plus | All canvas features free; **only AI features** (document extraction, voice) are Plus via bloom entitlements. |
| 12 | Insurance | User self-tracking only (policy, caps, claims, status she updates). No insurer API, nothing sent to insurers. |
| 13 | Marketplace depth | MVP **without online payment**: directory bookings are requests; shop orders are pay-on-delivery. |
| 14 | Marketplace ops | Everything (place approval, slots, booking confirmation, orders, returns, reviews) is handled by the Ritme team in admin-web. No business/seller panel. |
| 15 | Shop model | **Internal shop (cart, COD orders, tracking)** — supersedes bloom's partner link-out purchase path; bloom's shop entry/categories are kept. |
| 16 | Map | Neshan (`NEXT_PUBLIC_NESHAN_KEY`); list fallback without a key. |
| 17 | Teen ↔ mother | Invite by link/code on bloom's companion system, new `parent` type with teen-only grants; mother sees a read-only card. |
| 18 | Keys | Server env only (`GEMINI_API_KEY` via the AI adapter config, `NEXT_PUBLIC_NESHAN_KEY`, storage key). The user sets them on the server; never in the repo, `.env` files or logs. Locally the key may be read from `~/.gemini_key` in the shell only. |
| 19 | Task size | Medium (~95 tasks): schema/API, admin, 1–3 screens per frontend task, one QA task per epic, release tasks last. |
| 20 | Workflow | One commit per task (on the current branch, no push unless asked); auto-continue to the end of the current epic; parallel subagents allowed (max 3) when touches don't conflict — including against in-progress bloom tasks. Stage/prod deploys only when the user asks. |
