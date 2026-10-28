# T-M9-42d - Today / calendar v6 screenshot QA

Local Go with `CYCLE_ENGINE=v6`, Next dev, 390x844 @2x, light + dark, logged in via OTP read from a private `ritme_dev` DB.
Clock: 2026-10-28. Contract-fixture personas had `cycle_histories`, profile dates, `daily_health_logs` and `fertility_logs`
shifted +34 days, `terms` + `health_data_processing` consents inserted, `user_revisions.engine_rev` bumped (the engine
caches by revision, so raw SQL edits are invisible until the bump). Files are `<screen>_<state>_<light|dark>.png`.

| Shot | Persona | What it proves |
|---|---|---|
| `today_overdue15` | 09900000011 (`overdue_20`, 21 days late) | `overdue` banner renders (no zod error), «تأخیر پریود ۲۱ روز» ring, both banner actions (test + last period date) |
| `today_ask_last_period` | 09900000002 (profile-only, stale onboarding anchor 2026-09-01) | `ask_last_period` banner and its button, ring + week strip still draw |
| `today_unknown_ring_pill` | 09900000004 + `contraception_methods` combined_pill | `main_phase=unknown` ring (no phase label, no fake phase), countdown to next period, hormonal-contraception state |
| `overdue_banner_pregnancy_test_tap` | 09900000011 | tapping «ثبت تست بارداری» opens the log sheet on the measurements panel with the pregnancy-test chips visible (scrolled to them) |
| `calendar_pill` | 09900000004 (pill) | month calendar: only period days marked, no fertile/ovulation paint under hormonal contraception |
| `calendar_onboarding_only` | 09900000003 (declared last period, no logged periods) | calendar renders predicted days from the onboarding anchor |
| `calendar_overdue15` | 09900000011 | calendar for a 21-days-late user |
| `prediction_card_calendar` | 09900000003 | card with calendar-estimated ovulation («حدود», «تخمینی»), regularity pill |
| `prediction_card_lh_confirmed` | 09900000012 (LH positive 2026-10-27) | ovulation plain date + «بر اساس تست LH», fertile window «در جریان», pill «منظم» |
| `prediction_card_suppressed` | 09900000004 (pill) | fertile window row hidden, ovulation row shown as «—», note «پیش‌بینی روش پیشگیری نیست», pill «منظم» |

Not captured: BBT-confirmed ovulation (LH-confirmed shown instead; same `ovulationSource` code path, label differs).
