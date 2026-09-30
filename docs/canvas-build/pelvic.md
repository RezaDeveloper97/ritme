# Pelvic floor — catalog groups (CB-PELV-01)

Seeded in `backend-go/db/migrations/00010_pelvic_floor.sql` (seed kept in the schema migration because the task
allowed one migration; Laravel twin `2026_10_01_000001_create_pelvic_tables.php`). All rows `needs_review = 1`.

| group | codes | meta |
|---|---|---|
| `pelvic_levels` | `level_1` … `level_4` | `{week_from, hold_sec, rest_sec, reps, sets}` — level = last row with `week_from ≤ current week`; `session_sec = sets × reps × (hold + rest)` |
| `pelvic_alerts` | `uti_warning`, `program_suitability` | `{severity: "urgent" \| "info"}` |

Default levels `[needs clinical review]`: weeks 1–2 → 3 s/3 s, 3–4 → 5 s/5 s, 5–6 → 8 s/8 s, 7–8 → 10 s/10 s, 10 reps × 3 sets.
