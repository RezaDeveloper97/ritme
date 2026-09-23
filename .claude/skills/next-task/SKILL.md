---
name: next-task
description: Run the next Ritme task(s) from tasks/ in dependency order — serial or in parallel with subagents. Use at the start of a work session or when the user says "/next-task", "تسک بعدی", "ادامه بده", "برو جلو". Supports a task id, --parallel, --dry-run, --status, --add.
argument-hint: "[T-Mx-NN] [--parallel] [--dry-run] [--status] [--add \"description\"]"
---

# next-task — task runner for Ritme

You drive the Ritme build queue in `tasks/`. Talk to the user in **Persian**; code, commits, task files and docs in
English. Project rules are in `frontend/CLAUDE.md` and `backend/CLAUDE.MD` — they are law for their side.
Queue format: `tasks/README.md`.

## 0. Arguments (`$ARGUMENTS`)
- *(empty)* → run the next runnable task (serial).
- `T-Mx-NN` → run that task (check its deps are `done`; if not, say which and stop).
- `--parallel` → fan out every safe runnable task (§2).
- `--dry-run` → show which task(s) you'd pick and a 5–10 line plan each; change nothing.
- `--status` → `bash tasks/bin/next.sh --all`, summarise per milestone (done / in progress / blocked / next), stop.
- `--add "…"` → turn the description into one or more task files with `tasks/bin/new.sh` (fill Why/Scope/
  Acceptance, realistic `touches`, `depends_on`, `verify`), run `bash tasks/bin/next.sh --check` and
  `bash tasks/bin/index.sh`, show the result, stop. Don't implement.

## 0.1 Hard rule — no Android
Never include Android in any task: no `android` type, no work in `android-shell/`, `application/` or `twa/`, no Android
smoke/verification steps in Scope, Acceptance or verify, and no Android build. If a task file mentions Android, drop
that part (note it in the task) instead of doing it. (User decision, 2026-09-23.)

## 1. Session start ritual
1. `bash tasks/bin/next.sh` and `tail -5 tasks/LOG.md`.
2. `git status --short | head` — uncommitted work from a crashed session must be triaged first (it usually belongs
   to the `in_progress` task).
3. If a task is `in_progress`: resume **that** one (read its file, its section in `tasks/PROGRESS.md`, `git diff`).
   Never start a new task while one is in progress unless the user says so.
4. Otherwise pick the first runnable task (or the given id). Tell the user in one line which task(s) you're
   picking, then go.

## 2. Parallel mode
Run tasks concurrently only when **all** hold:
1. `--parallel` was passed or the user asked for parallel work this session.
2. They're all runnable now and share a `parallel_group` (or are all `investigate` tasks).
3. `bash tasks/bin/next.sh --conflicts <ids…>` exits 0. On a conflict, drop the later task from the batch.
4. None of them writes a migration unless it's the only one in the batch that does (migration timestamps collide).

Then: `--status <id> in_progress` for each, and spawn one `Agent` per task **in a single message** so they start
together (`subagent_type: general-purpose`; max 3). Agent `description` = `<ID> - <short label>`, e.g.
`T-M1-02 - year-long tokens`. Prompt skeleton:

> You are implementing Ritme task <ID>. Read `<task file>` fully, then the CLAUDE.md for your side
> (`frontend/CLAUDE.md` and/or `backend/CLAUDE.MD`). Load every skill in its `skills:` list with the Skill tool
> before coding. Edit only paths under `touches:`; if you must edit anything else, stop and report instead.
> Follow **Scope** literally; **Acceptance** is the definition of done; nothing from other tasks. Run the task's
> `verify:` command and include its output tail. Do NOT commit, do NOT change task status, do NOT deploy.
> Report: files changed, verify output, acceptance checklist (✔/✘ each), open items.

When they return: `git diff --stat` per task to confirm each stayed inside its `touches`, run every task's `verify`
on the combined tree, then do §4 for each task (one commit per task — stage by path).

## 3. Serial path
1. `bash tasks/bin/next.sh --status <ID> in_progress`
2. Read the task file. Load every skill in `skills:` **now**. Read the CLAUDE.md sections relevant to the paths
   in `touches:`.
3. Show a 5–10 line plan, then implement — don't wait for approval unless the task conflicts with a CLAUDE.md
   rule or needs a destructive/outward action (prod DB, production deploy, deleting data) → ask first.
4. Scope discipline: do the **Scope**, satisfy **Acceptance**, nothing from later tasks. If the task is wrong or
   missing a step, edit its file (or add `T-Mx-NNb` via `tasks/bin/new.sh`) and mention it in the report.
5. `investigate` tasks change no code — they produce the doc named in `touches:` with evidence (commands +
   output), and update the scope of the downstream tasks they feed.

## 4. Finish a task
1. Run the task's `verify:` command (and `verify-all` if listed in `skills`). **Paste the output tail.** Red = not
   done: fix it, or set `blocked` and add a `## Blocked` section to the task file with the reason.
2. Append a `## <ID> — <title>` section to `tasks/PROGRESS.md`: what shipped, new commands/env vars/migrations,
   open items.
3. `bash tasks/bin/next.sh --status <ID> done` and `bash tasks/bin/index.sh`.
4. Commit — Conventional Commits with the id, e.g. `fix(session): year-long sliding tokens (T-M1-02)`; include
   the task file, `tasks/LOG.md`, `tasks/INDEX.md`, `tasks/PROGRESS.md`. One commit per task. Never push or deploy
   to production unless the user asks.
5. Report in Persian: what shipped, verify tail, next runnable tasks (`bash tasks/bin/next.sh`) and whether they
   can run in parallel.

## 5. Keep going?
After finishing, continue with the next runnable task only if the user asked for more than one ("همه رو برو",
"تا آخر M1"); otherwise stop and report.

## 6. Skill hygiene
If you repeated a procedure ≥2 times that later tasks will need, create or extend a skill under `.claude/skills/`
and tell the user:
> اسکیل `X` رو ساختم/آپدیت کردم. برای لود شدنش Claude Code رو ری‌استارت کن یا `/clear` بزن.
