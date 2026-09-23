# tasks/ — Ritme build queue

One task = one Markdown file under `tasks/M<n>/`. **Status lives only in each task file's frontmatter.**
Driven from Claude Code by the `/next-task` skill (`.claude/skills/next-task/SKILL.md`).

| Command | Purpose |
|---|---|
| `bash tasks/bin/next.sh` | runnable tasks now (todo + all deps done), plus in_progress / blocked |
| `bash tasks/bin/next.sh --all` | every task and status |
| `bash tasks/bin/next.sh --status T-M1-01 in_progress` | change status (`todo`/`in_progress`/`done`/`blocked`), appends to `LOG.md` |
| `bash tasks/bin/next.sh --conflicts T-M1-02 T-M1-03` | exit 1 if the tasks' `touches` overlap (parallel safety) |
| `bash tasks/bin/next.sh --check` | validate ids, deps, statuses |
| `bash tasks/bin/new.sh ...` | create a new task file (see its header) |
| `bash tasks/bin/index.sh` | regenerate `INDEX.md` |

## Frontmatter

| Field | Meaning |
|---|---|
| `id` | `T-M<milestone>-<nn>`; an inserted follow-up gets a letter suffix (`T-M1-03b`) |
| `type` | `investigate` / `backend` / `frontend` / `fullstack` / `release` (no Android work in this queue — user decision 2026-09-23) |
| `depends_on` | hard order — a task is runnable only when all of these are `done` |
| `parallel_group` | tasks in the same group *may* run concurrently if their `touches` don't overlap |
| `touches` | paths the task owns; an agent must not edit outside them without reporting |
| `skills` | skills to load before starting |
| `verify` | the narrowest command that proves the task; must be green before `done` |

Body sections: **Why**, **Scope**, **Out of scope**, **Acceptance** (the definition of done).

Files: `LOG.md` = status transitions (append-only, written by `next.sh`), `PROGRESS.md` = per-task notes
(what shipped, env vars, migrations, open items), `INDEX.md` = generated overview.

## Milestones
- **M1** — Session lifetime (1 year), PWA correctness, performance & cleanup.
- **M2** — Backend migration Laravel → Go Fiber v3 (`backend-go/`), strangler cutover, new Next.js admin
  (`admin-web/`). Decisions, inventories and deviations: `docs/go-migration/`.
- **M3** — Care reminders (medications, doctor appointments, today's reminders card) from the v13 design; Go-only
  API under `/api/v1/care/`. Spec: `docs/care-reminders/README.md`, artboards `docs/design/reminders-v13/`.
- **M4** — Periodic checkups (screening plan, records, self-exam guide, history/PDF) + admin catalog in admin-web;
  Go-only API `/api/v1/checkups`, `/api/admin/v1/checkup-types`. Spec: `docs/checkups/README.md`, artboards
  `docs/design/checkups-v14/`.
- **M5** — TTC fertility tracking: home quick tiles (LH / BBT / intercourse), day log, BBT chart, predictions;
  Go-only API `/api/v1/fertility`. Spec: `docs/fertility-ttc/README.md`, artboards `docs/design/ttc-v19/`.
- **M7** — Pregnancy mode v2: re-enable pregnancy in signup/profile, 6 redesigned screens, admin-defined week
  details / care plan / alert rules, pregnancy tips + alerts inside the main message engine; Go-only
  `/api/v1/pregnancy/v2/`. Spec: `docs/pregnancy-v2/README.md`, artboards `docs/design/pregnancy-v2/`.
