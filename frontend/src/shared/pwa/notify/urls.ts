/** Same-origin in-app path only: one leading "/", nothing protocol-relative. Mirrors `safeAppPath` in scripts/sw.template.js. */
export function sanitizeAppPath(value: unknown, fallback = '/'): string {
  if (typeof value !== 'string' || value.length === 0 || value.length > 2000) return fallback;
  if (value[0] !== '/' || value[1] === '/' || value[1] === '\\') return fallback;
  if (/[\u0000-\u001f\u007f]/.test(value)) return fallback;
  try {
    const parsed = new URL(value, 'https://app.invalid');
    if (parsed.origin !== 'https://app.invalid') return fallback;
    return parsed.pathname + parsed.search + parsed.hash;
  } catch {
    return fallback;
  }
}

/** VAPID public key (base64url) → the bytes `pushManager.subscribe` wants. */
export function urlBase64ToBytes(base64Url: string): Uint8Array<ArrayBuffer> {
  const padding = '='.repeat((4 - (base64Url.length % 4)) % 4);
  const raw = atob((base64Url + padding).replace(/-/g, '+').replace(/_/g, '/'));
  const bytes = new Uint8Array(new ArrayBuffer(raw.length));
  for (let i = 0; i < raw.length; i += 1) bytes[i] = raw.charCodeAt(i);
  return bytes;
}

/** Allows a run at most once per `minGapMs` (`force` bypasses the gap but still records the run). */
export function createGate(minGapMs: number, now: () => number = Date.now) {
  let last = Number.NEGATIVE_INFINITY;
  return {
    tryEnter(force = false): boolean {
      const t = now();
      if (!force && t - last < minGapMs) return false;
      last = t;
      return true;
    },
    reset(): void {
      last = Number.NEGATIVE_INFINITY;
    },
  };
}
