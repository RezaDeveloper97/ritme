# UI autopilot

The autopilot takes a JSON script and operates the web app the way a person would: it opens the page or
sheet, finds the field, types the value, presses the chip or switch, and (if asked) saves. It is the base
for "say it and Ritme does it": a model turns the user's request into a script, the engine runs it.

It is frontend-only. The engine calls no model and costs no credits.

| Piece | Where |
|---|---|
| Vocabulary: `pilot()`, the script schema, catalog shape, manifest builder | `frontend/src/shared/pilot` |
| Engine: config, runner, DOM drivers, overlay, `runAutopilot` | `frontend/src/shared/autopilot` |
| **The catalog** (every route, sheet and target) | `frontend/src/app/autopilot/catalog.ts` + `groups/*.ts` (one file per area) |
| Catalog titles (`autopilotCatalog` namespace, loaded only where the manifest is built) | `frontend/messages/<locale>/autopilot-catalog.json` |
| Catalog as a table (generated) | `docs/autopilot-catalog.md` |
| Log-sheet ids and its manifest targets | `frontend/src/features/log-day/model/pilot-targets.ts` |
| Manifest for the signed-in user | `useAutopilotManifest()` in `frontend/src/widgets/autopilot-manifest` |
| Dev playground (404 in production) | `/dev/autopilot` |

## 1. The script (version 1)

```json
{
  "version": 1,
  "id": "optional, echoed in the report",
  "options": { "preset": "human" },
  "steps": [
    { "do": "set", "target": "log.param:bleeding.flow", "value": "heavy" },
    { "do": "set", "target": "log.param:mood.moods", "value": ["happy", "calm"] },
    { "do": "set", "target": "log.param:sleep.hours", "value": 7.5 },
    { "do": "tap", "target": "log.save", "say": "ذخیره می‌کنم" }
  ]
}
```

A step says what the screen should end up as, not where to click. The engine works out the kind of
control, opens whatever hides it, and leaves a field alone when it already holds the value, so a script
can run twice.

| `do` | Fields | Does |
|---|---|---|
| `open` | `route` (+ `query`) **or** `sheet` (+ `arg`) | Goes to a page, or opens a sheet, and waits for it |
| `set` | `target`, `value` | Brings a control to a value |
| `tap` | `target` | One press (button, tab, link) |
| `expand` / `collapse` | `target` | Opens / closes a section |
| `wait` | `ms` and/or `target` (+ `state`: `visible` \| `hidden`) | Pauses, or waits for something |
| `close` | none | Closes the top sheet |

Every step also takes `say` (a short caption shown while it runs), `optional` (a failure skips the step),
`timeoutMs`, and `pace` (timing overrides for that step).

`value` by kind of target:

| Kind | Value |
|---|---|
| `single` | one option code, or `null` to clear |
| `multi` | an array of option codes: the exact final set |
| `bool` | `true` / `false` |
| `number`, `integer` | a number inside the target's `min`..`max` |
| `text` | a string (`null` clears) |
| `date` | `"YYYY-MM-DD"` (Gregorian, whatever calendar is on screen) |
| `time` | `"HH:MM"` (24h) |

A page-only script is valid: `{"version":1,"steps":[{"do":"open","route":"/calendar"}]}`.

### Targets

A target is a stable name, `<area>.<type>` or `<area>.<type>:<key>`: `log.save`,
`log.param:bleeding.flow`, `nav.tab:calendar`. It sits on the DOM as `data-pilot` (the control, or the
group), `data-pilot-option` (one option of a group) and optionally `data-pilot-kind`.

Log-sheet ids are built from taxonomy codes (`log.param:<category>.<param>`, option = the value code),
the same codes `/logs/taxonomy`, `PUT /logs/days/{date}` and the `voice_log` parser use.

Two fallback locators exist for a place that is not instrumented yet, both off by default:
`{"role": "button", "name": "…"}` (accessible name, so it depends on the language; needs
`safety.allowRoleSelectors`) and `{"css": "…"}` (needs `safety.allowRawSelectors`).

### Reaching a target

If a target is not on screen, the engine looks it up in the catalog: it opens the target's `surface`
(a sheet, or a route with an optional `query`), runs the entry's `reveal` steps (expand the section, press
the button that opens its panel or inline sheet), then waits for it. A target with no `surface` (it lives on
a dynamic route such as `/children/[id]`) can only be used while the user is already there. So a script only needs `set log.param:bleeding.flow = heavy`.

### The report

`runAutopilot` always resolves with a report. It never contains a value.

```json
{ "status": "failed", "code": "step_failed",
  "steps": [ { "index": 0, "do": "set", "target": "log.param:bleeding.flow", "status": "ok" },
             { "index": 1, "do": "tap", "target": "log.cat:nope", "status": "failed", "code": "target_not_found" } ] }
```

- Run `status`: `done`, `cancelled`, `failed`. Run `code`: `invalid_script`, `unsafe_script`, `busy`,
  `disabled`, `not_ready`, `locked`, `user_cancelled`, `step_failed`.
- Step `status`: `ok`, `skipped`, `failed`. Step `code`: `target_not_found`, `timeout`, `blocked`,
  `invalid_value`, `disabled`, `unsupported`, `not_verified`, `declined`, `navigation_failed`, `error`.

## 2. Running one

```ts
import { runAutopilot } from '@/shared/autopilot';

const run = runAutopilot(script, { preset: 'brisk' }); // object or JSON text
const report = await run.done;                          // never rejects
run.pause(); run.resume(); run.cancel();
```

One run at a time (a second call reports `busy`). `useAutopilotStatus()` gives `status`, `step`, `total`.

## 3. Config: `frontend/src/shared/autopilot/config.ts`

Everything tunable is in `AUTOPILOT_CONFIG`. Change it there, never in the runner or a driver.

| Key | Meaning | Default |
|---|---|---|
| `enabled` | Master switch | `true` |
| `defaultPreset` | `human` (travels and types like a person), `brisk`, `instant` (no pointer, no pauses) | `human` |
| `presets.<name>` | The timing table of each preset: `beforeStepMs`, `afterStepMs`, `cursorSpeedPxPerSec`, `cursorMinMs`, `cursorMaxMs`, `hoverMs`, `pressHoldMs`, `typeCharMs`, `typeWordPauseMs`, `scrollSettleMs`, `afterNavigateMs`, `afterSheetMs`, `jitter` | see file |
| `visuals` | `cursor`, `highlight`, `ripple`, `caption`, `statusBar` on/off | all on |
| `typing.mode` | `perChar`, `perWord`, `instant` | `perChar` |
| `typing.focus` | `visual` keeps the phone keyboard down; `real` focuses the input | `visual` |
| `interrupt.onUserInput` | The user touches the screen mid-run: `pause`, `cancel`, `ignore` | `pause` |
| `errors.onStepError` / `errors.retries` | `stop` or `skip`; extra attempts for a missing target | `stop` / 1 |
| `timeouts` | `targetMs`, `navigationMs`, `verifyMs` | 6000 / 10000 / 1200 |
| `reducedMotion` | With «کاهش حرکت» on: `instant` or `keep` | `instant` |
| `safety.maxSteps` | Longest script accepted | 60 |
| `safety.allowedRoutes` / `blockedRoutes` | Route prefixes a script may / may never open | `['/']` / dev + sign-up flow |
| `safety.blockedTargets` | Target ids (or `family:*`) the engine refuses | `[]` |
| `safety.confirmTargets` | Target ids that need the user's «انجام بده» first | `[]` |
| `safety.allowRawSelectors` / `allowRoleSelectors` | Fallback locators | `false` / `false` |
| `safety.scriptMayOverride` | Option keys a script may set for itself (`preset`, `pace`) | `['preset']` |
| `debug` | Console trace of step kinds and target ids (never values) | `false` |

Layers, weakest first: defaults, the preset, the script's `options`, a step's `pace`, the
`runAutopilot(…, options)` argument.

## 4. The catalog and the agent manifest

`frontend/src/app/autopilot/catalog.ts` lists every route, sheet and target a script may use. It has two
readers:

- the **engine**, for `surface` and `reveal` (how to reach a target);
- the **manifest**, the vocabulary sent to the model next to the user's request.

`useAutopilotManifest()` builds the manifest for the signed-in user: the catalog, plus one target per
field of their own log taxonomy (their mode, their language), minus modules an admin switched off and
anything in `safety.blockedTargets` / `blockedRoutes`.

```json
{
  "version": 1,
  "guide": "Reply with ONE JSON object … (output rules for the model)",
  "actions": ["open", "set", "tap", "expand", "collapse", "wait", "close"],
  "routes":  [{ "route": "/calendar", "title": "تقویم" }],
  "sheets":  [{ "sheet": "log", "title": "ثبت روزانه", "arg": "YYYY-MM-DD (optional, today when absent)" }],
  "targets": [
    { "id": "log.param:bleeding.flow", "kind": "single", "label": "شدت خونریزی", "group": "خونریزی",
      "options": [{ "value": "light", "label": "کم" }, { "value": "heavy", "label": "شدید" }] },
    { "id": "log.param:sleep.hours", "kind": "number", "label": "ساعت خواب", "min": 0, "max": 24, "unit": "ساعت" },
    { "id": "log.save", "kind": "action", "label": "ذخیره‌ی ثبت روزانه" }
  ]
}
```

The manifest holds vocabulary only: nothing the user logged, no dates, no names.

**Sending it with a voice note (the log sheet's «ثبت با صدا»).** One round orb in the voice tab
(`features/voice-log`): tap, talk, tap. The clip and a manifest of the **log sheet's fields only**
(`logSheetManifest(logTargetsOf(taxonomy))`: `log.param:*` and `log.level:*`, action `set` — no routes, no
sheets, no save button) go to `POST /api/v1/logs/voice/pilot`. The server transcribes the clip with the model
of the internal `voice_stt` slot and asks the model of the `voice_log` slot for a script
(`backend-go/internal/voicelog/pilot.go`); the model is given a manifest the server rebuilt (its own rules, known fields only) and only `set`
steps on a listed id with a value that field accepts come back — never a model-written caption. The client
filters again (`voiceScript`: `set` only, on a listed id; a multi-select keeps what is already picked; `null`
never clears), switches to «ثبت دستی» and calls `runAutopilot(script, { preset: 'brisk', onStepError: 'skip' })`.
Nothing is saved: the user reads the filled form and presses «ذخیره» herself.

That request is an AI call: `platform.guard.Chain(ai.FeatureVoiceLog)` on the route, `useAiGate('voice_log')`
in `widgets/voice-log-slot` with `guard()` before recording, `handleError()` on failure and `refresh()` after
success (`docs/ai-credits.md`). The engine itself stays outside the gate. Without a running subscription the
tab wears «پلاس» and the gate sheet explains it when the orb is pressed.

Known limits: the fields of the detail panels (`log.panel-param:*`), the date strip and the period-date
editor are not offered to a voice note yet. On a full-page host (`/log`, `/pregnancy/log`, `/menopause/log`)
the fields' catalog surface is the `log` sheet, so the engine opens that sheet over the page and fills it there.

## 5. Instrumenting a screen

1. Give each interactive control a target id and put it on the element:
   - a button or link: `<button {...pilot('pay.submit')}>`;
   - a chip row: `<ChipGroup pilot={pilot(id, 'choice')}>` and `{...pilotOption(code)}` on each chip;
   - `Switch`, `Checkbox`, `Accordion`, `NumberStepper`, `SegmentedTabs`: their `pilot` prop;
   - a number input that commits on blur: `pilot(id, 'number')`; a text input: `pilot(id, 'text')`.
   A slice with many ids keeps them in its own `model/pilot-targets.ts`.
2. Register the id (or its `family:*`) in the right `app/autopilot/groups/*.ts` with its `surface`, its
   `reveal` if it can be hidden, a `titleKey` (an exact id) in `messages/*/autopilot-catalog.json`, and for a
   fixed choice its `options` (code + title key), for a number its `min` / `max` / `unitKey`.
3. Fields that come from data need a manifest builder like `logTargetsOf`, added to
   `useAutopilotManifest`.
4. `npm run autopilot:catalog` (regenerates `docs/autopilot-catalog.md`), then `npm run test`.

`src/app/autopilot/catalog.test.ts` fails when a marked id is not in the catalog, a catalog id is not
marked anywhere, a title is missing in `fa` or `en`, a route or sheet does not exist, or the generated
doc is stale.

Kinds the drivers know: `button`, `toggle` (`aria-pressed`), `switch` (`role=switch`, checkbox, radio,
tab), `choice`, `text`, `number`, `stepper`, `wheel` (WheelPicker), `ruler` (RulerPicker), `calendar`
(CalendarPicker), `select` (native), `disclosure` (`aria-expanded`). The kind is inferred
from the DOM when `data-pilot-kind` is absent.

## 6. Rules

- **Health data never leaves the run.** Reports, logs and the manifest carry ids and codes, never values.
- **A script is untrusted input.** It is validated (strict schema, no unknown fields), checked against
  `safety`, limited to in-app routes and catalogued sheets, and cannot run code or use CSS selectors.
- **Real events only.** Drivers click and type on the real elements; they never write React state. A
  control that refuses a value (a clamped number, a disabled chip) makes the step fail.
- **Only what a finger could press.** A control covered by a sheet, a dialog or the app lock is refused
  (`blocked`), and so is a link that leaves the app. A value the field rejects or clamps is undone.
- **The user stays in charge.** The bar can always pause or cancel; a touch on the screen pauses by
  default; the app lock stops a run.
- **Motion.** With reduced motion on, the run is instant and nothing animates.

## 7. Not built yet

- Voice beyond the log sheet: a surface that sends the whole manifest (navigation, other forms). Only the log
  sheet's voice tab exists (§4).
- Deliberately not instrumented (a misheard sentence must not do these): payment, purchase and order
  submit, cancel/renew subscription, delete account or data, sign-out, app lock, OTP, sign-up and
  onboarding, every delete/remove/revoke control, sending an assistant or doctor chat message, starting a
  lab or document analysis, file uploads, maps, emergency calls, the postpartum mood questionnaire, and the
  public desktop pages.
- Options that come from data are only in the manifest for the log sheet's accordion fields
  (`logTargetsOf`). The log sheet's detail-panel fields (`log.panel-param:*`) and API-driven lists
  elsewhere are named and drivable but their option codes are not listed for the agent yet.
- Controls under a `SectionTitle` action (menopause treatment "add", the IVF home's medication link), the
  body-map regions and `text_items` log params have no name yet.
- Six commit-type buttons ask the user first (`safety.confirmTargets` in the config).
