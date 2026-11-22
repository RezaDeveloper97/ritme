import { z } from 'zod';

/**
 * Every knob of the autopilot, in one place. Change behaviour HERE — not in the runner, not in a driver.
 *
 * Layers, weakest first: these defaults → the chosen preset's timing → the script's `options` (only the
 * keys in `safety.scriptMayOverride`) → a step's `pace` → the `runAutopilot(…, options)` argument.
 */

/** All durations are milliseconds. `jitter` is a 0–1 share of each duration, added or removed at random. */
const timingSchema = z.object({
  /** Pause before a step starts (reading the caption). */
  beforeStepMs: z.number().min(0),
  /** Pause after a step, before the next one. */
  afterStepMs: z.number().min(0),
  /** Pointer travel speed; the trip is clamped to [cursorMinMs, cursorMaxMs]. */
  cursorSpeedPxPerSec: z.number().positive(),
  cursorMinMs: z.number().min(0),
  cursorMaxMs: z.number().min(0),
  /** How long the pointer rests on a control before it presses. */
  hoverMs: z.number().min(0),
  /** How long the press itself is held (the ripple). */
  pressHoldMs: z.number().min(0),
  /** Per typed character / extra pause after a word. */
  typeCharMs: z.number().min(0),
  typeWordPauseMs: z.number().min(0),
  /** Settling time after scrolling a control into view. */
  scrollSettleMs: z.number().min(0),
  /** Settling time after a route change / a sheet opening. */
  afterNavigateMs: z.number().min(0),
  afterSheetMs: z.number().min(0),
  jitter: z.number().min(0).max(1),
});
export type PilotTiming = z.infer<typeof timingSchema>;

const HUMAN: PilotTiming = {
  beforeStepMs: 260,
  afterStepMs: 320,
  cursorSpeedPxPerSec: 900,
  cursorMinMs: 220,
  cursorMaxMs: 700,
  hoverMs: 140,
  pressHoldMs: 130,
  typeCharMs: 55,
  typeWordPauseMs: 90,
  scrollSettleMs: 260,
  afterNavigateMs: 450,
  afterSheetMs: 420,
  jitter: 0.25,
};

const BRISK: PilotTiming = {
  beforeStepMs: 80,
  afterStepMs: 110,
  cursorSpeedPxPerSec: 2200,
  cursorMinMs: 110,
  cursorMaxMs: 320,
  hoverMs: 50,
  pressHoldMs: 70,
  typeCharMs: 14,
  typeWordPauseMs: 0,
  scrollSettleMs: 140,
  afterNavigateMs: 260,
  afterSheetMs: 300,
  jitter: 0.1,
};

/** Everything at once: no pointer, no pauses. Waiting for the app itself (a route, a sheet) still happens. */
const INSTANT: PilotTiming = {
  beforeStepMs: 0,
  afterStepMs: 0,
  cursorSpeedPxPerSec: 1_000_000,
  cursorMinMs: 0,
  cursorMaxMs: 0,
  hoverMs: 0,
  pressHoldMs: 0,
  typeCharMs: 0,
  typeWordPauseMs: 0,
  scrollSettleMs: 0,
  afterNavigateMs: 0,
  afterSheetMs: 0,
  jitter: 0,
};

const presetName = z.enum(['human', 'brisk', 'instant']);
export type PilotPreset = z.infer<typeof presetName>;

const configSchema = z.object({
  /** Master switch: off = `runAutopilot` reports `disabled` and touches nothing. */
  enabled: z.boolean().default(true),
  defaultPreset: presetName.default('human'),
  presets: z
    .object({ human: timingSchema, brisk: timingSchema, instant: timingSchema })
    .default({ human: HUMAN, brisk: BRISK, instant: INSTANT }),

  /** What the user sees while it runs. With preset `instant` the pointer, ring and ripple are skipped anyway. */
  visuals: z
    .object({
      cursor: z.boolean().default(true),
      highlight: z.boolean().default(true),
      ripple: z.boolean().default(true),
      caption: z.boolean().default(true),
      statusBar: z.boolean().default(true),
    })
    .prefault({}),

  typing: z
    .object({
      /** `perChar` types like a person, `perWord` in bursts, `instant` pastes. */
      mode: z.enum(['perChar', 'perWord', 'instant']).default('perChar'),
      /** `visual` never focuses the input, so the phone keyboard stays down; `real` focuses it. */
      focus: z.enum(['visual', 'real']).default('visual'),
    })
    .prefault({}),

  /** The user touching the screen mid-run. */
  interrupt: z
    .object({
      onUserInput: z.enum(['pause', 'cancel', 'ignore']).default('pause'),
    })
    .prefault({}),

  errors: z
    .object({
      /** `stop` ends the run on the first failed step; `skip` carries on with the next one. */
      onStepError: z.enum(['stop', 'skip']).default('stop'),
      /** Extra attempts for a step whose target didn't show up. */
      retries: z.number().int().min(0).max(5).default(1),
    })
    .prefault({}),

  timeouts: z
    .object({
      targetMs: z.number().positive().default(6000),
      navigationMs: z.number().positive().default(10_000),
      /** How long a control may take to show the state we asked for. */
      verifyMs: z.number().positive().default(1200),
    })
    .prefault({}),

  /** With reduced motion on in the OS: `instant` drops all motion, `keep` runs the preset. */
  reducedMotion: z.enum(['instant', 'keep']).default('instant'),

  safety: z
    .object({
      maxSteps: z.number().int().positive().default(60),
      /** Route prefixes a script may open. `/` alone allows every in-app path. */
      allowedRoutes: z.array(z.string()).default(['/']),
      /** Routes no script may open, whatever `allowedRoutes` says (prefix match). */
      blockedRoutes: z.array(z.string()).default(['/dev', '/otp', '/signup', '/splash', '/welcome', '/onboarding']),
      /** Target ids (exact or `family:*`) the engine refuses to touch. */
      blockedTargets: z.array(z.string()).default([]),
      /** Target ids (exact or `family:*`) that need the user's «ادامه» before the engine touches them. */
      confirmTargets: z
        .array(z.string())
        .default([
          // Steps that commit the user to something outside the app, or switch their life-stage.
          'mode.confirm-go',
          'city-book.submit',
          'city-request.submit',
          'city-join.submit',
          'doctor-booked.move-confirm',
          'ins-quest.confirm',
          // Share / invite / consent / export / send / join: health data or a request leaves the user's hands.
          // The catalog test (`catalog.test.ts`) fails when a new id of that kind is in neither list.
          'invite.create',
          'invite.consent',
          'companion.join-submit',
          'companion.join-confirm',
          'companion.renew-invite',
          'teen.share:*',
          'report.send',
          'privacy.export',
          'privacy.export-json',
          'privacy.export-pdf',
          'privacy.consent:*',
          'assistant.export-pdf',
          'assistant.share:*',
          'ckh.export',
          'vtr.share',
          'labr.share',
          'rec.share',
          'doctor-book.share:*',
          'doctor-booked.consent:*',
          'home-lab-form.consent-share',
          'home-lab-form.consent-import',
          'home-lab-request.consent',
          'city-detail.report-send',
          'city-review.send',
          'shop-checklist.share',
          'programs.join:*',
          'programs.bleeding-join',
          'programs.pain-join',
          'programs.pmdd-join',
          'pelvic.program-join',
          'city-home.join',
          'city-detail.join',
        ]),
      /** `{ "css": … }` locators. Off: a script from a model must not be able to click arbitrary DOM. */
      allowRawSelectors: z.boolean().default(false),
      /**
       * `{ "role", "name" }` locators for screens that aren't instrumented yet. Off: they reach any visible
       * control by its label. When on, a control whose own `data-pilot` id is blocked or needs a yes is still refused.
       */
      allowRoleSelectors: z.boolean().default(false),
      /** Top-level option keys a script may set in its own `options`. */
      scriptMayOverride: z.array(z.string()).default(['preset']),
    })
    .prefault({}),

  /** Console trace of step kinds and target ids. Values are never logged (§11). */
  debug: z.boolean().default(false),
});

export type AutopilotConfig = z.infer<typeof configSchema>;
export type AutopilotConfigInput = z.input<typeof configSchema>;

/** Build a config from partial overrides (tests, the dev lab). */
export function defineAutopilotConfig(overrides: AutopilotConfigInput = {}): AutopilotConfig {
  return configSchema.parse(overrides);
}

/**
 * THE app config. Edit the object below to retune the autopilot for everyone, e.g.
 * `{ defaultPreset: 'brisk', interrupt: { onUserInput: 'cancel' } }`.
 */
export const AUTOPILOT_CONFIG: AutopilotConfig = defineAutopilotConfig({});

/** What a caller may change for one run. */
export interface RunOptions {
  preset?: PilotPreset;
  /** Timing fields that win over the preset. */
  pace?: Partial<PilotTiming>;
  onUserInput?: AutopilotConfig['interrupt']['onUserInput'];
  onStepError?: AutopilotConfig['errors']['onStepError'];
}

export interface ResolvedConfig extends AutopilotConfig {
  preset: PilotPreset;
  timing: PilotTiming;
  /** True when nothing should animate: preset `instant`, or reduced motion with `reducedMotion: 'instant'`. */
  still: boolean;
}

function cleanPace(pace: Record<string, unknown> | undefined): Partial<PilotTiming> {
  if (!pace) return {};
  const parsed = timingSchema.partial().safeParse(pace);
  return parsed.success ? parsed.data : {};
}

/**
 * Folds the layers into the config one run uses. Pure: `reducedMotion` is passed in, not read here.
 */
export function resolveConfig(
  base: AutopilotConfig,
  input: { scriptOptions?: Record<string, unknown>; run?: RunOptions; reducedMotion?: boolean } = {},
): ResolvedConfig {
  const allowed = new Set(base.safety.scriptMayOverride);
  const script = input.scriptOptions ?? {};
  const scriptPreset = allowed.has('preset') ? presetName.safeParse(script.preset) : null;
  const scriptPace = allowed.has('pace') ? cleanPace(script.pace as Record<string, unknown> | undefined) : {};

  let preset: PilotPreset = input.run?.preset ?? (scriptPreset?.success ? scriptPreset.data : base.defaultPreset);
  if (input.reducedMotion && base.reducedMotion === 'instant') preset = 'instant';

  const timing = { ...base.presets[preset], ...scriptPace, ...cleanPace(input.run?.pace) };
  return {
    ...base,
    interrupt: { onUserInput: input.run?.onUserInput ?? base.interrupt.onUserInput },
    errors: { ...base.errors, onStepError: input.run?.onStepError ?? base.errors.onStepError },
    preset,
    timing,
    still: preset === 'instant',
  };
}

/** A step's own `pace` on top of the run's timing. */
export function paceFor(timing: PilotTiming, pace: Record<string, unknown> | undefined): PilotTiming {
  return pace ? { ...timing, ...cleanPace(pace) } : timing;
}
