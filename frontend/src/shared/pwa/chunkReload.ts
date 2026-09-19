/**
 * Recovery from deploy skew: a tab still running the previous build asks for
 * a lazy chunk the new deploy deleted, the server answers 404 and webpack
 * throws `ChunkLoadError`. The fix is fresh HTML, i.e. one reload — guarded by
 * a sessionStorage timestamp so a chunk that is genuinely broken can never
 * cause a reload loop (pwa-audit S-13).
 */

const GUARD_KEY = 'ritme_chunk_reload_at';
// A second failure within this window means the reload did not help.
const GUARD_WINDOW_MS = 30_000;

const CHUNK_MESSAGE =
  /Loading (?:CSS )?chunk [\w./-]+ failed|Failed to fetch dynamically imported module|Importing a module script failed|error loading dynamically imported module/i;

export function isChunkLoadError(value: unknown): boolean {
  if (!value || typeof value !== 'object') return false;
  const { name, message } = value as { name?: unknown; message?: unknown };
  if (name === 'ChunkLoadError') return true;
  return typeof message === 'string' && CHUNK_MESSAGE.test(message);
}

/** Pure guard: reload unless one already happened inside the window. */
export function shouldReloadForChunkError(lastReloadAt: number | null, now: number): boolean {
  if (lastReloadAt === null) return true;
  // A timestamp from the future (clock change) cannot vouch for anything.
  if (now < lastReloadAt) return true;
  return now - lastReloadAt >= GUARD_WINDOW_MS;
}

function readGuard(): number | null {
  try {
    const raw = window.sessionStorage.getItem(GUARD_KEY);
    const at = raw === null ? Number.NaN : Number(raw);
    return Number.isFinite(at) ? at : null;
  } catch {
    return null;
  }
}

function writeGuard(at: number): boolean {
  try {
    window.sessionStorage.setItem(GUARD_KEY, String(at));
    return true;
  } catch {
    // Without storage we cannot prove there is no loop — do not reload.
    return false;
  }
}

/** Reloads once for a chunk error. Returns true when it triggered a reload. */
export function reloadOnChunkError(error: unknown): boolean {
  if (typeof window === 'undefined' || !isChunkLoadError(error)) return false;
  const now = Date.now();
  if (!shouldReloadForChunkError(readGuard(), now)) return false;
  if (!writeGuard(now)) return false;
  window.location.reload();
  return true;
}

let installed = false;

/**
 * Installs window-level listeners once per page. Deliberately never removed:
 * Next's built-in root error boundary unmounts the whole tree (including the
 * component that installed this) *before* it reports the uncaught error via
 * `window.reportError`, so a listener tied to a component would be gone.
 */
export function installChunkErrorRecovery(): void {
  if (installed || typeof window === 'undefined') return;
  installed = true;
  window.addEventListener('error', (event) => {
    reloadOnChunkError(event.error);
  });
  window.addEventListener('unhandledrejection', (event) => {
    reloadOnChunkError(event.reason);
  });
}
