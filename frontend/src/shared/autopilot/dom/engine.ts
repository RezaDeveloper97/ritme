import { systemReducesMotion } from '@/shared/theme';

import { AUTOPILOT_CONFIG, resolveConfig, type ResolvedConfig, type RunOptions } from '../config';
import { RunControl, runScript, type RunCode, type RunReport } from '../core/runner';
import { checkScript } from '../core/safety';
import { parseAutopilotScript } from '@/shared/pilot';
import { EMPTY_CATALOG } from '@/shared/pilot';
import { useAutopilotCatalogStore, useAutopilotStore } from '../core/store';
import { createBrowserEnv, type PilotRuntime } from './env';

/** The engine's front door. One run at a time; the mounted `AutopilotHost` supplies the runtime. */

export interface RunHandle {
  /** Always resolves — a failure is a report, not a rejection. */
  done: Promise<RunReport>;
  pause: () => void;
  resume: () => void;
  cancel: () => void;
}

let runtime: PilotRuntime | null = null;
let active: { control: RunControl; config: ResolvedConfig } | null = null;

/** Called by `AutopilotHost` on mount; returns the unmount. */
export function mountAutopilotRuntime(next: PilotRuntime): () => void {
  runtime = next;
  useAutopilotCatalogStore.setState({ catalog: next.catalog });
  return () => {
    if (runtime !== next) return;
    active?.control.cancel();
    runtime = null;
    useAutopilotCatalogStore.setState({ catalog: EMPTY_CATALOG });
  };
}

const refused = (code: RunCode, issues?: string[]): RunHandle => ({
  done: Promise.resolve({ status: 'failed', code, ...(issues ? { issues } : {}), steps: [] }),
  pause: () => {},
  resume: () => {},
  cancel: () => {},
});

/**
 * Runs a version-1 script (object or JSON text) against the live UI.
 *
 * This function calls no model and costs no credits. A caller that got the script FROM a model must have
 * gone through `useAiGate` to get it (frontend/CLAUDE.md §11.5) — the gate belongs to that request.
 */
export function runAutopilot(script: unknown, options: RunOptions = {}): RunHandle {
  if (typeof window === 'undefined' || !runtime) return refused('not_ready');
  if (!AUTOPILOT_CONFIG.enabled) return refused('disabled');
  if (active) return refused('busy');

  const parsed = parseAutopilotScript(script);
  if (!parsed.ok) return refused('invalid_script', parsed.issues);

  const config = resolveConfig(AUTOPILOT_CONFIG, {
    scriptOptions: parsed.script.options,
    run: options,
    reducedMotion: systemReducesMotion(),
  });
  const unsafe = checkScript(parsed.script, config.safety);
  if (unsafe.length) return refused('unsafe_script', unsafe.map((i) => `${i.step}: ${i.code}`));

  const store = useAutopilotStore.getState();
  const control = new RunControl();
  control.onChange = (status) => useAutopilotStore.getState().set({ status });
  active = { control, config };
  store.reset();
  store.set({ status: 'running', total: parsed.script.steps.length, shield: config.interrupt.onUserInput !== 'ignore' });

  const done = runScript({
    script: parsed.script,
    config,
    catalog: runtime.catalog,
    env: createBrowserEnv(runtime),
    control,
    // Step kinds and target ids only — never a value (§11).
    log: config.debug ? (line) => console.debug(`[autopilot] ${line}`) : undefined,
  }).finally(() => {
    active = null;
    useAutopilotStore.getState().reset();
  });

  return { done, pause: () => control.pause(), resume: () => control.resume(), cancel: () => control.cancel() };
}

export function pauseAutopilot(): void {
  active?.control.pause();
}

export function resumeAutopilot(): void {
  active?.control.resume();
}

export function cancelAutopilot(): void {
  // A pending «ادامه؟» question would otherwise hold the run open.
  useAutopilotStore.getState().confirm?.answer(false);
  active?.control.cancel();
}

/** The user touched the screen while the autopilot was driving. */
export function userInterrupted(): void {
  if (!active || active.control.isPaused) return;
  if (active.config.interrupt.onUserInput === 'cancel') cancelAutopilot();
  else if (active.config.interrupt.onUserInput === 'pause') pauseAutopilot();
}
