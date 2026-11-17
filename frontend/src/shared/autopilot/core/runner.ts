import { paceFor, type PilotTiming, type ResolvedConfig } from '../config';
import { findCatalogTarget, type PilotCatalog, type PilotSurface } from '@/shared/pilot';
import { checkStep, needsConfirm } from './safety';
import type { PilotScript, PilotStep, PilotValue } from '@/shared/pilot';
import { describeTarget, type TargetRef } from '@/shared/pilot';
import { jittered, type Rng } from './timing';

/**
 * The run loop: walks a validated script step by step against a {@link PilotEnv}. It knows nothing about
 * the DOM — the browser env (`dom/env.ts`) does the looking and pressing, a fake one drives the tests.
 */

export type StepCode =
  | 'target_not_found'
  | 'timeout'
  | 'blocked'
  | 'invalid_value'
  | 'disabled'
  | 'unsupported'
  | 'not_verified'
  | 'declined'
  | 'navigation_failed'
  | 'error';

export type RunCode = 'invalid_script' | 'unsafe_script' | 'busy' | 'disabled' | 'not_ready' | 'user_cancelled' | 'locked' | 'step_failed';

export type ActResult = { ok: true } | { ok: false; code: StepCode };
export const OK: ActResult = { ok: true };
export const fail = (code: StepCode): ActResult => ({ ok: false, code });

export interface StepReport {
  index: number;
  do: PilotStep['do'];
  /** The target id (or locator kind) — never a value. */
  target?: string;
  status: 'ok' | 'skipped' | 'failed';
  code?: StepCode;
}

export interface RunReport {
  id?: string;
  status: 'done' | 'cancelled' | 'failed';
  code?: RunCode;
  /** Validation / safety problems, when the script never started. */
  issues?: string[];
  steps: StepReport[];
}

/** What a driver gets for one action. `wait` is the only sleep it may use: it honours pause and cancel. */
export interface ActContext {
  timing: PilotTiming;
  config: ResolvedConfig;
  wait: (ms: number) => Promise<void>;
  rng: Rng;
}

export type PilotVerb = 'tap' | 'set' | 'expand' | 'collapse';

export interface PilotEnv {
  sleep(ms: number): Promise<void>;
  /** The app lock closed over the screen: nothing may be driven. */
  locked(): boolean;
  exists(target: TargetRef): boolean;
  waitFor(target: TargetRef, state: 'visible' | 'hidden', timeoutMs: number, ctx: ActContext): Promise<boolean>;
  surfaceOpen(surface: PilotSurface): boolean;
  open(where: { route: string; query?: Record<string, string> } | { sheet: string; arg?: string }, ctx: ActContext): Promise<ActResult>;
  close(ctx: ActContext): Promise<ActResult>;
  act(verb: PilotVerb, target: TargetRef, value: PilotValue, ctx: ActContext): Promise<ActResult>;
  caption(text: string | null): void;
  progress(index: number, total: number): void;
  /** Ask the user before a `confirmTargets` step. */
  confirm(targetId: string, ctx: ActContext): Promise<boolean>;
}

class Cancelled extends Error {}

/** Pause / resume / cancel for one run. */
export class RunControl {
  private paused = false;
  private cancelled = false;
  private resumers: Array<() => void> = [];
  private cancelers: Array<() => void> = [];
  onChange: ((state: 'running' | 'paused') => void) | null = null;

  get isPaused(): boolean {
    return this.paused;
  }

  get isCancelled(): boolean {
    return this.cancelled;
  }

  pause(): void {
    if (this.cancelled || this.paused) return;
    this.paused = true;
    this.onChange?.('paused');
  }

  resume(): void {
    if (!this.paused) return;
    this.paused = false;
    this.onChange?.('running');
    this.resumers.splice(0).forEach((r) => r());
  }

  cancel(): void {
    if (this.cancelled) return;
    this.cancelled = true;
    this.paused = false;
    this.resumers.splice(0).forEach((r) => r());
    this.cancelers.splice(0).forEach((c) => c());
  }

  /** Throws once cancelled; parks while paused. */
  async checkpoint(): Promise<void> {
    while (this.paused && !this.cancelled) await new Promise<void>((r) => this.resumers.push(r));
    if (this.cancelled) throw new Cancelled();
  }

  /** `promise`, cut short by a cancel. */
  async guard<T>(promise: Promise<T>): Promise<T> {
    if (this.cancelled) throw new Cancelled();
    let off: () => void = () => {};
    const cancelled = new Promise<never>((_, reject) => {
      off = () => reject(new Cancelled());
      this.cancelers.push(off);
    });
    try {
      return await Promise.race([promise, cancelled]);
    } finally {
      this.cancelers = this.cancelers.filter((c) => c !== off);
    }
  }
}

export interface RunInput {
  script: PilotScript;
  config: ResolvedConfig;
  catalog: PilotCatalog;
  env: PilotEnv;
  control: RunControl;
  rng?: Rng;
  log?: (line: string) => void;
}

/** A reveal may itself need revealing (param → section → sheet); deeper than this is a catalog loop. */
const MAX_DEPTH = 4;
const RETRYABLE: ReadonlySet<StepCode> = new Set<StepCode>(['target_not_found', 'timeout', 'not_verified']);

export async function runScript(input: RunInput): Promise<RunReport> {
  const { script, config, catalog, env, control } = input;
  const rng = input.rng ?? Math.random;
  const steps: StepReport[] = [];
  const confirmed = new WeakSet<PilotStep>();
  const report = (status: RunReport['status'], code?: RunCode): RunReport => ({
    ...(script.id ? { id: script.id } : {}),
    status,
    ...(code ? { code } : {}),
    steps,
  });

  const contextFor = (step: PilotStep): ActContext => ({
    timing: paceFor(config.timing, step.pace),
    config,
    rng,
    wait: async (ms) => {
      if (ms > 0) await control.guard(env.sleep(ms));
      await control.checkpoint();
    },
  });

  /** Brings a target on screen: its surface first, then the catalog's reveal steps, then a wait. */
  const bring = async (target: TargetRef, ctx: ActContext, timeoutMs: number, depth: number): Promise<ActResult> => {
    if (env.exists(target)) return OK;
    const match = typeof target === 'string' ? findCatalogTarget(catalog, target) : null;
    if (match && depth < MAX_DEPTH) {
      const { surface } = match.entry;
      if (surface && !env.surfaceOpen(surface)) {
        const where = surface.sheet ? { sheet: surface.sheet } : surface.route ? { route: surface.route, query: surface.query } : null;
        if (where) {
          const opened = await control.guard(env.open(where, ctx));
          if (!opened.ok) return opened;
          await control.checkpoint();
        }
      }
      for (const prerequisite of match.entry.reveal?.(match.key) ?? []) {
        if (env.exists(target)) break;
        // A blocked section stays closed, whatever sits inside it.
        if (checkStep(prerequisite, -1, config.safety)) return fail('blocked');
        // A reveal step that can't run isn't fatal by itself — the final wait decides.
        await execute(prerequisite, ctx, depth + 1);
      }
    }
    const found = await control.guard(env.waitFor(target, 'visible', timeoutMs, ctx));
    await control.checkpoint();
    return found ? OK : fail('target_not_found');
  };

  const execute = async (step: PilotStep, ctx: ActContext, depth: number): Promise<ActResult> => {
    const timeoutMs = step.timeoutMs ?? config.timeouts.targetMs;
    switch (step.do) {
      case 'open': {
        const where = step.sheet ? { sheet: step.sheet, arg: step.arg } : { route: step.route ?? '/', query: step.query };
        return control.guard(env.open(where, ctx));
      }
      case 'close':
        return control.guard(env.close(ctx));
      case 'wait': {
        if (step.ms !== undefined) await ctx.wait(step.ms);
        if (step.target === undefined) return OK;
        const seen = await control.guard(env.waitFor(step.target, step.state ?? 'visible', timeoutMs, ctx));
        return seen ? OK : fail('timeout');
      }
      default: {
        const here = await bring(step.target, ctx, timeoutMs, depth);
        if (!here.ok) return here;
        // Asked once per step: a retry after `not_verified` must not ask again.
        if (depth === 0 && typeof step.target === 'string' && needsConfirm(step, config.safety) && !confirmed.has(step)) {
          const allowed = await control.guard(env.confirm(step.target, ctx));
          if (!allowed) return fail('declined');
          confirmed.add(step);
        }
        return control.guard(env.act(step.do, step.target, step.do === 'set' ? step.value : null, ctx));
      }
    }
  };

  try {
    for (let index = 0; index < script.steps.length; index++) {
      const step = script.steps[index];
      const ctx = contextFor(step);
      const target = 'target' in step && step.target !== undefined ? describeTarget(step.target) : undefined;
      const base = { index, do: step.do, ...(target ? { target } : {}) };

      await control.checkpoint();
      if (env.locked()) return report('failed', 'locked');
      env.progress(index, script.steps.length);
      env.caption(step.say ?? null);
      input.log?.(`#${index} ${step.do}${target ? ` ${target}` : ''}`);
      await ctx.wait(jittered(ctx.timing.beforeStepMs, ctx.timing.jitter, rng));

      let result: ActResult = checkStep(step, index, config.safety) ? fail('blocked') : OK;
      if (result.ok) {
        for (let attempt = 0; ; attempt++) {
          try {
            result = await execute(step, ctx, 0);
          } catch (error) {
            if (error instanceof Cancelled) throw error;
            result = fail('error');
          }
          await control.checkpoint();
          if (result.ok || !RETRYABLE.has(result.code) || attempt >= config.errors.retries) break;
        }
      }

      if (result.ok) {
        steps.push({ ...base, status: 'ok' });
      } else if (step.optional || config.errors.onStepError === 'skip') {
        steps.push({ ...base, status: 'skipped', code: result.code });
      } else {
        steps.push({ ...base, status: 'failed', code: result.code });
        input.log?.(`#${index} failed: ${result.code}`);
        return report('failed', 'step_failed');
      }
      await ctx.wait(jittered(ctx.timing.afterStepMs, ctx.timing.jitter, rng));
    }
    return report('done');
  } catch (error) {
    if (error instanceof Cancelled) return report('cancelled', 'user_cancelled');
    return report('failed', 'step_failed');
  } finally {
    env.caption(null);
  }
}
