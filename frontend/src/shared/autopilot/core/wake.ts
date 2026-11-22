/**
 * Lazy start of the autopilot runtime (T-M12-28). The layout keeps only this tiny module in the first-load
 * bundle; the catalog + engine + overlay are `import()`ed the first time something asks for them (a script
 * starts, or a screen reads the catalog). No imports on purpose: anything this file pulls in is first-load JS.
 */

let wanted = false;
let ready = false;
const wakeListeners = new Set<() => void>();

/**
 * The layout's `AutopilotMount` (app layer) cannot import this file (the barrel would pull the engine into the first-load
 * bundle), so the wake also goes out as this DOM event — keep the string in step with `app/autopilot/AutopilotMount.tsx`.
 */
export const AUTOPILOT_WAKE_EVENT = 'ritme:autopilot-wake';
const readyWaiters = new Set<(ok: boolean) => void>();

/** Ask for the runtime to be loaded and mounted. Idempotent. */
export function requestAutopilotRuntime(): void {
  if (wanted) return;
  wanted = true;
  wakeListeners.forEach((l) => l());
  if (typeof window !== 'undefined') {
    (window as { __ritmeAutopilotWanted?: boolean }).__ritmeAutopilotWanted = true; // for a host that mounts after the ask
    window.dispatchEvent(new Event(AUTOPILOT_WAKE_EVENT));
  }
}

export function autopilotWanted(): boolean {
  return wanted;
}

export function subscribeAutopilotWake(listener: () => void): () => void {
  wakeListeners.add(listener);
  return () => wakeListeners.delete(listener);
}

/** `mountAutopilotRuntime` calls this with `true` on mount and `false` on unmount. */
export function setAutopilotRuntimeReady(next: boolean): void {
  ready = next;
  if (next) {
    readyWaiters.forEach((w) => w(true));
    readyWaiters.clear();
  }
}

/** Resolves `true` once a runtime is mounted, `false` after `timeoutMs` (the chunk failed to load). */
export function whenAutopilotRuntimeReady(timeoutMs: number): Promise<boolean> {
  if (ready) return Promise.resolve(true);
  return new Promise((resolve) => {
    const waiter = (ok: boolean) => {
      clearTimeout(timer);
      readyWaiters.delete(waiter);
      resolve(ok);
    };
    const timer = setTimeout(() => waiter(false), timeoutMs);
    readyWaiters.add(waiter);
  });
}

/** Test seam: forget everything. */
export function resetAutopilotWake(): void {
  wanted = false;
  ready = false;
  readyWaiters.forEach((w) => w(false));
  readyWaiters.clear();
}
