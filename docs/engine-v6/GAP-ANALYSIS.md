# Engine v6 — gap → task map

Every finding and proposed task of the gap reports in [`gap/`](gap/) maps to a `/next-task` task in `tasks/M9`,
`tasks/M10` or `tasks/M11`. Gap-local ids (left column) are the ids used inside each report; they are not task ids.
Open questions of each report became rows of [`DECISIONS.md`](DECISIONS.md), asked by the first task of each
milestone.

| Milestone | Tasks | First task (asks the user) | Release tasks |
|---|---|---|---|
| M9 — cycle engine, speed, safety | 51 | T-M9-01 (E, I, X rows) | T-M9-48 stage · T-M9-49 perf gate · T-M9-50 beta · T-M9-51 retire v1.1 |
| M10 — messages, health, analysis, KPI, admin | 36 | T-M10-01 (M, H, A rows; advisor sheet) | T-M10-36 |
| M11 — family | 23 | T-M11-01 (F rows) | T-M11-23 |

T-M9-02 (family privacy hotfix) also asks the user before it changes what members see. Production stays on Laravel
(T-M2-26/27 blocked); beta deploys always wait for the user's OK.

## Decisions

| Gap report questions | DECISIONS.md section | Asked in |
|---|---|---|
| 01 cycle core — decisions 1–22 (+ spec items found later) | E-01 … E-29 | T-M9-01 |
| 02 inputs & consumers — input questions | I-01 … I-11 | T-M9-01 |
| 07 infra — cache, queue, rollout, budgets | X-01 … X-11 | T-M9-01 |
| 03 message engine — Q1–Q13 | M-01 … M-13 (+ M-14 … M-19 added) | T-M10-01 |
| 04 health modules — clinical and product items | H-01 … H-18 (clinical rows → `clinical-signoff.md`) | T-M10-01 |
| 06 analysis/KPI/admin — decisions | A-01 … A-12 | T-M10-01 |
| 05 family — A1–A21 | F-01 … F-21 (one to one) + F-22 … F-26 added | T-M11-01 |

## 01 — cycle core (`gap/01-cycle-core.md` §5)

| Gap id | Task |
|---|---|
| ENG-01 skeleton, params, enums, metrics | T-M9-15 |
| ENG-02 core resolve | T-M9-16 |
| ENG-03 quality layer | T-M9-18 |
| ENG-04 ovulation sources (BBT, LH, luteal) | T-M9-17 |
| ENG-05 future cycles, stale anchor, skipped cycle | T-M9-20 |
| ENG-06 limiting conditions | T-M9-19 |
| ENG-07 Today UI (§2.43) | T-M9-21 |
| ENG-08 benchmarks, invariants, goldens | T-M9-22 |
| SVC-01 input loader, params, cache key | T-M9-25, T-M9-31 |
| SVC-02 `cycle_view` v6 payload | T-M9-32 |
| SVC-03 skip endpoint | T-M9-29 |
| SVC-04 `/cycle/month` on `Range` | T-M9-33 |
| SVC-05 daily card on v6 | T-M9-34 |
| SVC-06 onboarding seed as a declared anchor | T-M9-27 |
| CON-01 backend consumers | T-M9-34, T-M9-35, T-M9-36, T-M9-37, T-M9-38 |
| CON-02 frontend | T-M9-40 … T-M9-47 |
| CON-03 retire v1.1 | T-M9-51 |

## 02 — engine inputs and consumers (`gap/02-inputs-consumers.md`)

| Gap id | Task |
|---|---|
| IC-01 decisions | T-M9-01 |
| IC-02 schema for v6 inputs | T-M9-24 |
| IC-03 input snapshot loader | T-M9-25 |
| IC-04 declared onboarding anchor | T-M9-27 |
| IC-05 bleeding-log bridge | T-M9-28 |
| IC-06 skipped-cycle endpoint | T-M9-29 |
| IC-07 one cache key for every input | T-M9-03 (quick fix), T-M9-31 (revision cache) |
| IC-08 user-local day | T-M9-26 |
| IC-09a month and `/home` | T-M9-33, T-M9-34 |
| IC-09b fertility, analysis, health record | T-M9-35 |
| IC-09c small consumers | T-M9-36, T-M9-37 |
| IC-09d messages | T-M9-38 |
| IC-10 `user_profiles.user_id` UNIQUE | T-M9-23 |
| IC-13 `pregnancy_week` fact | T-M9-03 |

## 02b / 02c — frontend (`gap/02b-frontend-consumers.md` §9, `gap/02c-frontend-impact.md` §F)

| Gap id | Task |
|---|---|
| 02b F1 · 02c F1 tolerant cycle schema | T-M9-40 |
| 02b F2 stop fabricating engine output | T-M9-45, T-M9-46 |
| 02b F3 · 02c F3 hide fertility under hormonal contraception | T-M9-44 |
| 02b F4 · 02c F2 Today screen §2.43 and lateness tiers | T-M9-42 |
| 02b F5 prediction card v6 | T-M9-43 |
| 02b F6 · 02c F4 calendar and ring from `future_cycles` | T-M9-45 |
| 02b F7 clinical flags and "normal" copy audit | T-M9-46 |
| 02c F5 remaining browser math | T-M9-46 |
| 02b F8 cycle-dependent invalidation | T-M9-41 |
| Skip flow end to end | T-M9-47 |

## 03 — message engine (`gap/03-message-engine.md` §5)

| Gap id | Task |
|---|---|
| MSG-01 message model v6, versioned approval | T-M10-02 |
| MSG-02 fact set v6 | T-M9-38 (v6 facts in today's builder), T-M10-03 |
| MSG-03 selection pipeline v6 | T-M10-04 |
| MSG-04 rule cache with pub/sub | T-M9-13, T-M10-05 |
| MSG-05 feed, server-side events, retention | T-M10-06 |
| MSG-06 alerts inbox | T-M10-07 |
| MSG-07 pregnancy alerts into the engine | T-M10-08 |
| MSG-08 menopause messages and condition nudges | T-M10-09 |
| MSG-09 family slots `member_alert` / `member_tip` | T-M11-12, T-M11-13 |
| MSG-10 challenges | T-M10-10 |
| MSG-11 simulator and conflict check | T-M10-11 |
| MSG-12 CTR per version and CTR-drop alert | T-M10-06 (rollup), T-M10-30, T-M10-31 |
| ADM-01 rule editor | T-M10-12 |
| ADM-02 simulator screen · ADM-03 inbox and challenge admin | T-M10-13 |
| WEB-01 home reads the feed | T-M10-14 |
| WEB-02 generic alerts inbox | T-M10-15 |

## 04 — health modules (`gap/04-health-modules.md` §6)

| Gap id | Task |
|---|---|
| H01 clinical registry v2 | T-M10-16 |
| H02 trimester, Naegele, ACOG re-dating | T-M10-18 |
| H03 pregnancy-aware vitals classifier | T-M9-05 (emergency tiers), T-M10-19 |
| H04 fetal movement and contraction emergency paths | T-M9-05 |
| H05 postpartum red flags and visit reminders | T-M9-06 |
| H06 vitals completion | T-M10-17 (schema), T-M10-19 |
| H07 labs v6 | T-M10-17 (schema), T-M10-20 |
| H08 checkups on USPSTF | T-M10-17 (schema), T-M10-21 |
| H09 child growth, CDC 2022 bands, versioned vaccines | T-M10-22 |
| H10 consent model v6 | T-M10-23 |
| H11 emergency-path guarantees and nightly self-test | T-M9-05, T-M9-06, T-M10-24 |
| H12 retire the v1 pregnancy alert engine | T-M10-08 |

## 05 — family (`gap/05-family-access.md` §3, §6)

| Gap id | Task |
|---|---|
| FAM-00 decisions and family doc | T-M11-01 |
| FAM-01a schema: links, grants, owner state, activity | T-M11-02 (schema), T-M11-05 (grant data migration) |
| FAM-01b schema: proposals, share requests, nudges, member alerts and notices | T-M11-03 (consents use `consent_events` from T-M10-23) |
| FAM-01c schema: guardians, actor columns | T-M11-04 |
| FAM-02 `family.Authorizer` | T-M11-05 |
| FAM-03 member projections | T-M9-02 (leak hotfix), T-M11-06 |
| FAM-04 mode change and owner review | T-M11-07 |
| FAM-05 invites v6 and share requests | T-M11-08 |
| FAM-06 proposals | T-M11-09 |
| FAM-07 nudges | T-M11-10 |
| FAM-08 co-guardianship, undo, deletion transfer | T-M11-11 |
| FAM-09 urgent-alert relay | T-M11-12 |
| FAM-10 member home, member tips, member notifications | T-M11-13 |
| FAM-11 household to-do (+ child medication log, F-26) | T-M11-04 (schema), T-M11-14 |
| FAM-12 admin family tab and KPIs | T-M11-15 |
| FAM-13 Plus for view-level members | T-M11-16 (optional, F-18) |
| FE-FAM-1 people hub and add-member flow | T-M11-17 |
| FE-FAM-2 member acceptance | T-M11-18 |
| FE-FAM-3 activity and approvals | T-M11-19 |
| FE-FAM-4 member home set | T-M11-20 |
| FE-FAM-5 child co-guardian UI | T-M11-21 |
| ADMIN-FAM-1 admin-web family tab | T-M11-22 |

| Finding | Task |
|---|---|
| FAM-H1 mood shared | T-M9-02, T-M11-06 |
| FAM-H2 member writes land in her records | T-M11-09 |
| FAM-H3 stale pregnancy grant | T-M9-02, T-M11-05, T-M11-07 |
| FAM-H4 owner deletion deletes the children | T-M11-04, T-M11-11 |
| FAM-M1 cycle view exposes numbers · FAM-M2 private free text | T-M9-02, T-M11-06 |
| FAM-M3 postpartum counted as pregnancy | T-M9-02 |
| FAM-M4 no per-domain maximum | T-M11-05 |
| FAM-M5 code-only invites activate at once | T-M11-08 |
| FAM-M6 TOCTOU between check and write | T-M11-05, T-M11-09, T-M11-11 |
| FAM-M7 no scoped consent | T-M10-23, T-M11-05, T-M11-08 |
| FAM-M8 male-only companion home | T-M11-13, T-M11-20 |
| FAM-L1 public default pepper · FAM-L3 accept oracle and lock DoS | T-M11-08 |
| FAM-L2 blocked owner readable | T-M11-05 |
| FAM-L4 member notice purge by JSON scan | T-M11-03 |
| FAM-L5 grant change without diff | T-M11-05 |
| FAM-L6 no `no-store` on member responses | T-M9-02, T-M11-06 |
| FAM-I1 raw SQL in the admin user summary | T-M10-32, T-M11-15 |

## 06 — analysis, KPI, admin (`gap/06-analysis-kpi-admin.md` §5)

| Gap id | Task |
|---|---|
| AK-01 analysis methods per §4.1 · AK-03 top finding and "N more days" | T-M10-26 |
| AK-02 analysis reads engine verdicts | T-M9-35 |
| AK-04 postpartum analysis hub | T-M10-27 |
| AK-05 analysis cache and warm job | T-M10-28 |
| AK-06 Plus quotas | T-M10-29 |
| AK-07 consent ledger | T-M10-23 |
| AK-08 engine daily snapshots | T-M9-39 |
| AK-09 prediction log and real MAE | T-M9-39 (log), T-M10-30 (KPI) |
| AK-10 KPI completion · AK-14 OKR status rule | T-M10-30 |
| AK-11 automatic alerts | T-M10-25 (schema), T-M10-31 |
| AK-12 jobs and indexes | T-M9-11 (jobs), T-M10-25 (indexes) |
| AK-13 admin user monitoring | T-M10-32 |
| AK-15 roles, message authority, immutable audit | T-M10-02 (authority), T-M10-33 |
| AW-01 dashboard alerts and engine health · AW-02 KPI tiles · AW-04 OKR | T-M10-34 |
| AW-03 users, segments, engine card · AW-05 roles screen | T-M10-35 (editor gating in T-M10-12) |

## 07 — infrastructure and performance (`gap/07-infra-performance.md` §3, §8)

| Gap id | Task |
|---|---|
| P01 observability | T-M9-07 |
| P02 benchmarks and query budgets | T-M9-08 |
| P03 engine invalidation (per-user revision; params fingerprint instead of a global epoch) | T-M9-24, T-M9-30 |
| P04 engine state cache (L1 + L2 + singleflight) | T-M9-31 |
| P05 all modules on the engine state | T-M9-32 … T-M9-38 |
| P06 config caches and the rules N+1 | T-M9-13 |
| P07 auth hot path | T-M9-09 |
| P08 period-write concurrency (B2) | T-M9-04 |
| P09 queue v2 and scheduler | T-M9-11 |
| P10 nightly snapshots and analysis precompute | T-M9-39, T-M10-28 |
| P11 worker process | T-M9-12 |
| P12 HTTP hardening and conditional GETs | T-M9-10, T-M9-31 |
| P13 Redis and MariaDB tuning (B4) | T-M9-10 |
| P14 k6 load tests and SLO gate | T-M9-14, T-M9-49 |
| P15 user-local day (B3) | T-M9-26 |
| B1 stale cache after a clinical-config change | T-M9-03, T-M9-31 |
| B5 language registry without TTL | T-M9-03 |
| B6 safety results only in memory | T-M10-24 (+ table in T-M10-25) |
| B7 version bump only on a valid period | T-M9-30 |
| B8 wasted reads | T-M9-25, T-M9-31, T-M10-28 |
| RabbitMQ / NATS / Kafka | Not used — asynq on Redis (README §4, X-02) |
