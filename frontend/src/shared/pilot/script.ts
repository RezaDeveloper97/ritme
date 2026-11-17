import { z } from 'zod';

import { isTargetId, type TargetRef } from './target';

/**
 * The autopilot script, version 1 — the JSON an agent (or a developer) hands to `runAutopilot`.
 *
 * Declarative on purpose: a step says what the screen should end up as (`set` this field to that
 * value), never where to click. The engine works out the control, opens whatever hides it and leaves
 * a field alone when it already holds the value, so the same script can run twice.
 */
export const SCRIPT_VERSION = 1;

const targetSchema = z.union([
  z.string().refine(isTargetId, 'not a target id'),
  z.object({ role: z.string().min(1), name: z.string().min(1) }).strict(),
  z.object({ css: z.string().min(1) }).strict(),
]) satisfies z.ZodType<TargetRef>;

/** What a control can be set to. `null` clears it; an array is the exact set of a multi-select. */
const valueSchema = z.union([z.string(), z.number().finite(), z.boolean(), z.null(), z.array(z.string())]);
export type PilotValue = z.infer<typeof valueSchema>;

/** Per-step speed override: any subset of the timing table (config.ts). */
const paceSchema = z.record(z.string(), z.number().min(0)).optional();

const common = {
  /** A short caption shown while the step runs. */
  say: z.string().max(160).optional(),
  /** A failure skips the step instead of ending the run. */
  optional: z.boolean().optional(),
  timeoutMs: z.number().int().positive().max(60_000).optional(),
  pace: paceSchema,
};

const openStep = z
  .object({
    do: z.literal('open'),
    route: z.string().optional(),
    query: z.record(z.string(), z.string()).optional(),
    sheet: z.string().min(1).optional(),
    arg: z.string().max(80).optional(),
    ...common,
  })
  .strict()
  .refine((s) => Boolean(s.route) !== Boolean(s.sheet), 'open needs exactly one of route / sheet');

const setStep = z.object({ do: z.literal('set'), target: targetSchema, value: valueSchema, ...common }).strict();
const tapStep = z.object({ do: z.literal('tap'), target: targetSchema, ...common }).strict();
const expandStep = z.object({ do: z.literal('expand'), target: targetSchema, ...common }).strict();
const collapseStep = z.object({ do: z.literal('collapse'), target: targetSchema, ...common }).strict();
const waitStep = z
  .object({
    do: z.literal('wait'),
    ms: z.number().int().min(0).max(30_000).optional(),
    target: targetSchema.optional(),
    state: z.enum(['visible', 'hidden']).optional(),
    ...common,
  })
  .strict()
  .refine((s) => s.ms !== undefined || s.target !== undefined, 'wait needs ms or target');
const closeStep = z.object({ do: z.literal('close'), ...common }).strict();

const stepSchema = z.union([openStep, setStep, tapStep, expandStep, collapseStep, waitStep, closeStep]);
export type PilotStep = z.infer<typeof stepSchema>;
export type PilotAction = PilotStep['do'];

export const PILOT_ACTIONS = ['open', 'set', 'tap', 'expand', 'collapse', 'wait', 'close'] as const satisfies readonly PilotAction[];

const scriptSchema = z
  .object({
    version: z.literal(SCRIPT_VERSION),
    /** Echoed in the report; never shown. */
    id: z.string().max(80).optional(),
    /** Overrides of the config — only the keys `safety.scriptMayOverride` lists are honoured. */
    options: z.record(z.string(), z.unknown()).optional(),
    steps: z.array(stepSchema).min(1),
  })
  .strict();
export type PilotScript = z.infer<typeof scriptSchema>;

export type ParseResult = { ok: true; script: PilotScript } | { ok: false; issues: string[] };

/** Validates without running. Accepts the object or its JSON text. */
export function parseAutopilotScript(input: unknown): ParseResult {
  let raw = input;
  if (typeof input === 'string') {
    try {
      raw = JSON.parse(input);
    } catch {
      return { ok: false, issues: ['not valid JSON'] };
    }
  }
  const parsed = scriptSchema.safeParse(raw);
  if (parsed.success) return { ok: true, script: parsed.data };
  // Paths and messages only — zod never echoes the offending value here, so no user data leaks (§11).
  return { ok: false, issues: parsed.error.issues.map((i) => `${i.path.join('.') || 'script'}: ${i.message}`) };
}

/** The target a step acts on, if it has one. */
export function targetOf(step: PilotStep): TargetRef | undefined {
  return 'target' in step ? step.target : undefined;
}
