# VOICE — design fidelity (canvas-build §5)

Board renders: `docs/qa/canvas/boards/nbl_Voice_*.png` (`roadmap/bin/shot-board.sh`). Screens: 390 px, fa, headless
Chrome over CDP with Chrome's **fake microphone** (`--use-fake-device-for-media-stream`, auto-granted permission) against
the local Go API :8020 (rebuilt from this tree so `/logs/voice/commit` exists; `AI_PROVIDER` fake) + Next dev :3000.
Test user **`09120005055`** (menopause; a Plus **trial** was started on it for this QA via `POST /plus/trial/start`).
Cycle shots: life-stage switched to `cycle` for the run and back to `menopause` afterwards. The fake mic records a
tone, so the fake transcriber answers its `default` fixture (belly pain · bloating · «بی‌حوصله» with sad / fatigue
options); the menopause run tags the upload with `RITME-FAKE:menopause` (test-only `FormData` hook in the QA script)
so the diary path (hot flashes → `POST /logs/voice/commit`) is exercised. Network order checked in both runs:
`POST /logs/voice` → `PUT /logs/days/{date}` → `POST /logs/voice/commit`. The reminder switch was toggled on (light)
and off (dark) — `PUT /profile/cycle-settings` 200 — then restored to the user's original `daily_log` 22:00 on.
Colours come from the token map, never the board hex.

## CB-VOICE-02

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Voice_Entry` | log sheet «ثبت با صدا» tab (`/log`, `?sheet=log`); menopause: `/menopause/log` «با صدا بگو» card | [cycle](voice/CB-VOICE-02/cycle-entry.light.png) · [menopause card](voice/CB-VOICE-02/meno-card.light.png) → [entry](voice/CB-VOICE-02/meno-entry.light.png) | [cycle](voice/CB-VOICE-02/cycle-entry.dark.png) · [menopause card](voice/CB-VOICE-02/meno-card.dark.png) → [entry](voice/CB-VOICE-02/meno-entry.dark.png) | ✔ | Same order and copy: «بگو امروز چطور بودی» + lead, 112 px solid `--brand-fill` mic in a `--brand-soft` halo, «برای شروع بزن · تا یک دقیقه» (server cap is 60 s, so not the board's «۲ دقیقه»), «مثلاً بگو» three quote pills, «ثبت‌های امروز» list (cycle: پریود · درد · خواب; menopause: علائم بدن · خواب · پریود — real summaries from the day, «هنوز ثبت نشده» when empty; a row opens that section in «ثبت دستی»), privacy note with shield. Privacy copy keeps the honest «… به یک سرویس هوش مصنوعی فرستاده می‌شود …» (board's shorter line omits the third party). Header/date strip/tabs are bloom's log sheet; the menopause card (CB-MENO-06) opens the same panel with «‹ ثبت دستی» back. Plus lock unchanged (bloom `PlusFeatureGate`). |
| `nbl_Voice_Record` | same, after tapping the mic | [cycle](voice/CB-VOICE-02/cycle-record.light.png) · [menopause](voice/CB-VOICE-02/meno-record.light.png) | [cycle](voice/CB-VOICE-02/cycle-record.dark.png) · [menopause](voice/CB-VOICE-02/meno-record.dark.png) | ✔ | Full-height step (date strip + tabs hidden): «در حال گوش دادن», red dot «در حال ضبط» + Lalezar timer, live brand waveform, «متن زنده» card, chips, then «II» (MediaRecorder pause/resume, timer holds) · `--period` stop · «لغو» (drops the clip, nothing uploaded). **Deliberate difference:** `POST /logs/voice` takes the whole clip — no streaming — so «متن زنده» says the words appear after stop (with a caret) and the chips are «دنبال این‌ها می‌گردیم» (neutral, unchecked) instead of «تا الان فهمیدیم» with checks; nothing is shown as understood before the server answers (no fake data). While uploading the same screen shows «در حال فهمیدن صدای تو…» + skeleton. |
| `nbl_Voice_Review` | same, after stop | [cycle](voice/CB-VOICE-02/cycle-review.light.png) · [menopause](voice/CB-VOICE-02/meno-review.light.png) | [cycle](voice/CB-VOICE-02/cycle-review.dark.png) · [menopause](voice/CB-VOICE-02/meno-review.dark.png) | ✔ | «این‌ها را فهمیدیم / درست است؟»; «خلاصه» card (the confirmed chip labels joined — the API returns no prose summary) + «متن کامل صحبتت» disclosure with the transcript; one card with a section per category / diary (icon disc, title, pencil → merge + open the section in «ثبت دستی»; diaries have no pencil), chips toggle an item in/out; short single-choice params get the board's segmented row (e.g. bleeding flow); ambiguity: «مطمئن نیستیم» pill + «منظورت کدام بود؟» chooser from `options[]` (cycle shot: بی‌حوصله / غمگین / خستگی); «چیزی جا افتاده؟ اضافه کن» dashed button (merge → manual); «یک نکته» `--period-soft` note only when heavy / very heavy flow is included [needs clinical review of the copy]; sticky glass footer «تأیید و ثبت N مورد» + «دوباره بگو». **Per-item quote («از حرفت: …») omitted:** the API returns no per-item quote (CB-VOICE-01 open item); the full transcript is one tap away. Save: log items merge into the sheet draft → `PUT /logs/days` with `voice_params`, then diary items → `POST /logs/voice/commit` (pain score needs its location first). |
| `nbl_Voice_Saved` | same, after confirming | [cycle](voice/CB-VOICE-02/cycle-saved.light.png) · [menopause](voice/CB-VOICE-02/meno-saved.light.png) | [cycle](voice/CB-VOICE-02/cycle-saved.dark.png) · [menopause](voice/CB-VOICE-02/meno-saved.dark.png) | ✔ | Check disc (`--data-soft` / `--data-deep`), «N مورد ثبت شد» + body, saved list (server labels split «title · caption»; a label without caption shows under its category, e.g. «حال / بی‌حوصله», diaries under «گرگرفتگی»), «دفعه بعد سریع‌تر» card with a switch = the existing `daily_log` reminder of `/profile/cycle-settings` at 21:00 (no new backend; on = enabled at 21:00, off = disabled) — copy says «یادآور ثبت امروز» since the push text is the daily-log one, not «امروز را بگو»; «برگشت به خانه» (sheet closes / `/log` and `/menopause/log` go home). |

## CB-VOICE-03 — epic QA

Re-run on the current tree (HEAD `dce95bf` + the two fixes below) against the same local Go API :8020 (already
rebuilt from this tree; `POST /logs/voice/commit` answers 401 unauthenticated → route present; `AI_PROVIDER` fake) and
Next dev :3000, same headless-Chrome fake-mic script, test user `09120005055` (menopause, Plus trial). New run =
menopause path with the upload tagged `RITME-FAKE:pelvic`, so the **bladder diary** (leak + night voids, not exercised
in CB-VOICE-02) goes through review → `POST /logs/voice/commit` → saved. The cycle path was not re-shot: the two
fixes are mode-independent CSS / button order, and the CB-VOICE-02 cycle shots are otherwise the current tree.
Restored afterwards: the `pelvic_bladder_logs` row the run wrote was deleted (the user had none); mode untouched
(menopause throughout); the reminder switch was not toggled this time.

**Fixed here (small drift):**
1. `nbl_Voice_Record` — control order was mirrored: the board (RTL) has «لغو» at the start edge and «II» at the end
   edge; ours had them swapped. `VoiceRecording.tsx` now renders cancel · stop · pause.
2. `nbl_Voice_Entry` «ثبت‌های امروز» + `nbl_Voice_Saved` list — the shared row's own radius bent the 1px divider into
   an arc; straight dividers like the board via `.vlog-today / .vlog-saved-list .nb-list-rows > * + * { border-radius: 0 }`
   (same fix other screens use; CB-VOICE-02 CSS block).

### Final verdicts

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Voice_Entry` | log sheet «ثبت با صدا» tab (`/log`, `?sheet=log`); menopause `/menopause/log` «با صدا بگو» card | [menopause card](voice/CB-VOICE-03/meno-leak-card.light.png) → [entry](voice/CB-VOICE-03/meno-leak-entry.light.png) · cycle [CB-VOICE-02](voice/CB-VOICE-02/cycle-entry.light.png) | [menopause card](voice/CB-VOICE-03/meno-leak-card.dark.png) → [entry](voice/CB-VOICE-03/meno-leak-entry.dark.png) · cycle [CB-VOICE-02](voice/CB-VOICE-02/cycle-entry.dark.png) | ✔ | Divider arc fixed (above). Known, accepted: «تا یک دقیقه» (server cap 60 s, board «۲ دقیقه»); longer privacy copy naming the AI service; header/date strip/tabs are bloom's log sheet. |
| `nbl_Voice_Record` | same, after tapping the mic | [menopause](voice/CB-VOICE-03/meno-leak-record.light.png) · cycle [CB-VOICE-02](voice/CB-VOICE-02/cycle-record.light.png) | [menopause](voice/CB-VOICE-03/meno-leak-record.dark.png) · cycle [CB-VOICE-02](voice/CB-VOICE-02/cycle-record.dark.png) | ✔ | Control order fixed (above; CB-VOICE-02 shots predate it). Known, accepted (no fake data): one-shot API, so no live transcript and «دنبال این‌ها می‌گردیم» neutral chips instead of «تا الان فهمیدیم» with checks. |
| `nbl_Voice_Review` | same, after stop | [menopause · bladder](voice/CB-VOICE-03/meno-leak-review.light.png) · [menopause · hot flash](voice/CB-VOICE-02/meno-review.light.png) · [cycle](voice/CB-VOICE-02/cycle-review.light.png) | [menopause · bladder](voice/CB-VOICE-03/meno-leak-review.dark.png) · [menopause · hot flash](voice/CB-VOICE-02/meno-review.dark.png) · [cycle](voice/CB-VOICE-02/cycle-review.dark.png) | ✔ | Bladder diary section «دفترچه مثانه» (leak «با سرفه یا عطسه» + night voids «۲ بار»), keep/remove chips, «تأیید و ثبت ۲ مورد». Known, accepted: no per-item quote «از حرفت: …» (API has none), «خلاصه» = joined labels, diary items not editable (keep/remove only). |
| `nbl_Voice_Saved` | same, after confirming | [menopause · bladder](voice/CB-VOICE-03/meno-leak-saved.light.png) · cycle [CB-VOICE-02](voice/CB-VOICE-02/cycle-saved.light.png) | [menopause · bladder](voice/CB-VOICE-03/meno-leak-saved.dark.png) · cycle [CB-VOICE-02](voice/CB-VOICE-02/cycle-saved.dark.png) | ✔ | Divider arc fixed (above). Saved rows «نشت ادرار / با سرفه یا عطسه», «بیدار شدن شبانه برای ادرار / ۲ بار». Network: `POST /logs/voice` 200 → `POST /logs/voice/commit` 200 (no `PUT /logs/days`: no log items in this recording). Switch off = the user's `daily_log` reminder is at 22:00, not 21:00 (as designed in CB-VOICE-02). |

No ✘. Page errors: none in either theme.

### Parser accuracy — 10 Persian fixtures (fake provider)

`backend-go/internal/voicelog/accuracy_int_test.go` (`TestVoice_AccuracyFixtures`, integration — `make test-int
PKG=./internal/voicelog/...`): each sentence is injected as a fake transcript (`RITME-FAKE:<key>`), sent through
`POST /logs/voice` (fake transcriber + fake rule-based `ParseLog` — **no Gemini call**), then saved like the client:
log items via `PUT /logs/days` with `voice_params`, diary items via `POST /logs/voice/commit`. Cycle users are
enrolled in the endo programme and on a combined pill (so pain diary + pill are offered; bladder is for everyone);
menopause users are in menopause mode (hot flash + bladder). «Expected» = what a careful person would log. The test
pins today's misses / extras, so a parser change shows up as a diff here.

| # | Mode | Utterance | Expected | Parsed (target: item = value) | Miss / extra | Save |
|---|---|---|---|---|---|---|
| c1 | cycle | «امروز پریودم شروع شد، خونریزیم زیاده و دلم خیلی درد می‌کنه» | bleeding flow heavy · abdomen pain severe | — nothing | ✘ flow (no bleeding lexicon in the fake) · ✘ abdomen («دلم **خیلی** درد» — an adverb between breaks the phrase) | nothing to save |
| c2 | cycle | «از صبح کمرم درد می‌کنه، هفت از ده. ساعت دو یه ژلوفن خوردم ولی اثر نکرد» | back pain severe · painkiller · pain diary score 7, ژلوفن, 14:00, no effect | log: back = severe, painkiller · pain_diary: score 7, analgesic ژلوفن, time **02:00**, effect no | ~ time «ساعت دو» read as 02:00 (afternoon not inferred) | PUT 200 · commit 200 (6 items) |
| c3 | cycle | «قرص ضد بارداریم رو امروز صبح خوردم، یه کم هم سرم درد می‌کنه» | pill taken · head pain mild | pill: taken · log: head = mild | — | PUT 200 · commit 200 |
| c4 | cycle | «امروز وقتی عطسه کردم یه کم ادرارم نشت کرد، شب هم سه بار برای دستشویی بیدار شدم» | leak (cough/sneeze) · night voids 3 | bladder: leak = cough, night_voids = 3 | — | commit 200 |
| c5 | cycle | «امروز خیلی بی‌حوصله‌ام و نفخ دارم، دیشب هم بد خوابیدم» | mood bored (ambiguous) · bloating · sleep poor | log: mood bored **+2 options** (غمگین / خستگی) · bloating · sleep quality poor | — | PUT 200 |
| m1 | menopause | «امروز پنج بار گرگرفتگی داشتم، بیشترش بعد از چای داغ» | hot flashes × 5 · trigger hot drink | hot_flash: count 5 · log: trigger hot_drink · **cravings sour** | + extra «ترش» matched inside «بیشترش» (label substring match) | PUT 200 · commit 200 |
| m2 | menopause | «دیشب دو بار با گرگرفتگی و عرق شبانه از خواب پریدم و بی‌خوابی داشتم» | hot flashes × 2 at night · night sweats · insomnia | hot_flash: count **1**, night · log: night sweats, insomnia | ~ count: «دو بار **با** گرگرفتگی» not matched → default 1 | PUT 200 · commit 200 |
| m3 | menopause | «زانوهام درد می‌کنه، حدود پنج از ده، یه استامینوفن خوردم و کمک کرد» | joint pain moderate · painkiller (no endo enrolment → no pain diary) | log: joints = moderate (score 5 → level), painkiller | — | PUT 200 |
| m4 | menopause | «وقتی خندیدم یه کم ادرارم چکه کرد و خشکی واژن هم اذیتم می‌کنه» | leak (laugh) · vaginal dryness | bladder: leak = cough · log: vaginal dryness · **sex › dryness** | + extra duplicate dryness under «رابطه» (label «خشکی» substring) | PUT 200 · commit 200 |
| m5 | menopause | «امروز بی‌حوصله‌ام، تمرکز ندارم و حواسم پرته» | mood bored (ambiguous) · brain fog | log: mood bored **+2 options** | ✘ brain fog («حواس**م** پرته» / «تمرکز ندارم» not in the lexicon) | PUT 200 |

**Result:** 22 / 27 expected items parsed with the right value — **81 % recall**; 22 / 26 suggestions correct —
**85 % precision**; 4 / 10 sentences exactly right (c3, c4, c5, m3). Targets were always right (diary vs log;
pain diary only for the enrolled user, hot flash only in menopause mode). Every save that had items succeeded
(9 / 9 — 8 `PUT /logs/days` 200, 6 `POST /logs/voice/commit` 200). Both ambiguous moods produced the chooser with 2
options. This measures the **fake** rule-based parser (dev / stage / tests); production quality depends on the
Gemini prompt and needs its own run with keys.

### Open items (none blocking)
- Fake-parser gaps found by the fixtures → `CB-VOICE-03b` (proposed): bleeding / period phrases, intensity adverbs
  inside a phrase («دلم خیلی درد»), «N بار با گرگرفتگی», «حواسم پرت / تمرکز ندارم», afternoon reading of «ساعت دو»,
  whole-word label matching (no «ترش» in «بیشترش», no second «خشکی» under «رابطه»).
- Same doubt for Gemini: label substring / duplicate-category suggestions should be checked in a real-provider run
  (needs keys; not done here).
- Carried from CB-VOICE-01/02: per-item quote, live transcript (streaming), diary items not editable, «چیزی جا افتاده؟»
  drops unsaved diary items, offline-queued day save fails the diary commit, «یک نکته» copy [needs clinical review],
  21:00 reminder riding on `daily_log`.
