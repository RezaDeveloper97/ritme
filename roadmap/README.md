# roadmap/ — canvas-build queue

Build queue for the **«ریتمی — یائسگی» canvas** (menopause, IVF, loss, conditions, contraception, pelvic floor,
privacy, teen, health record, insurance, voice, mother & child services, shop). Driven by the `/canvas-build` skill
(`.claude/skills/canvas-build/SKILL.md`). Built **on top of `bloom/`** — see `DECISIONS.md` (binding).

Design snapshot: `docs/design/canvas-v1/` (boards, text digests, board map in its README).

| Command | Purpose |
|---|---|
| `bash roadmap/bin/next.sh` | runnable tasks + in progress + blocked + tasks waiting only on bloom |
| `bash roadmap/bin/next.sh --all` | every task and status |
| `bash roadmap/bin/next.sh --status CB-MENO-01 in_progress` | change status (`todo`/`in_progress`/`done`/`blocked`), appends to `LOG.md` |
| `bash roadmap/bin/next.sh --conflicts CB-A CB-B B-N3-03` | exit 1 if `touches` overlap (accepts bloom ids too) |
| `bash roadmap/bin/next.sh --check` | validate ids, deps (incl. `B-N*`) and statuses |
| `bash roadmap/bin/board.sh` | per-epic progress board |
| `bash roadmap/bin/new.sh …` | create a task file (see its header) |
| `bash roadmap/bin/index.sh` | regenerate `INDEX.md` |

## Frontmatter
`id` (`CB-<EPIC>-NN`, follow-ups `CB-<EPIC>-NNb`) · `title` · `epic` · `type` (`investigate` / `backend` / `frontend` /
`fullstack` / `qa` / `release`) · `status` · `depends_on` (CB or `B-N*` ids) · `parallel_group` · `touches` · `skills` ·
`boards` (files in `docs/design/canvas-v1/boards/`) · `verify`.
Body: **Why**, **Boards**, **Scope**, **Out of scope**, **Acceptance**.

## Epics
| Folder | Epic | What |
|---|---|---|
| E00-core | CORE | canvas→bloom map, extra primitives, content catalog + admin, shared file storage, Neshan map |
| E01-nav | NAV | global search, align bloom's shell with the canvas IA rules |
| E02-meno | MENO | full menopause mode |
| E03-ivf | IVF | IVF/IUI treatment tracker |
| E04-loss | LOSS | pregnancy-loss path |
| E05-cond | COND | endometriosis pain diary, PMDD, heavy bleeding (PBAC), PCOS minimal |
| E06-contra | CONTRA | contraception |
| E07-pelv | PELV | pelvic floor + bladder diary |
| E08-priv | PRIV | discreet notifications (lock = bloom) |
| E09-teen | TEEN | teen mode + mother sharing |
| E10-rec | REC | record documents, extraction, sharing, emergency card |
| E11-ins | INS | insurance self-tracking |
| E12-voice | VOICE | voice coverage for canvas items (base = bloom) |
| E13-dir | DIR | mother & child directory (MVP) + desktop web |
| E14-shop | SHOP | internal COD shop + layette checklist + desktop web |
| E15-rel | REL | stage / prod rollout — only on explicit request |

Files: `LOG.md` (status transitions, append-only), `PROGRESS.md` (per-task notes), `INDEX.md` (generated),
QA reports in `docs/qa/canvas/<epic>.md`.
