/**
 * Client-side frequency cap (B-N10-02): an ad is shown at most MAX_PER_DAY times a calendar day on this device.
 * Kept in localStorage only — never sent to the server, never part of a profile.
 */
export const MAX_PER_DAY = 4;
const KEY = 'ritme.ads.freq';

type Counts = { day: string; n: Record<string, number> };

function today(now: Date): string {
  return `${now.getFullYear()}-${now.getMonth() + 1}-${now.getDate()}`;
}

function read(now: Date): Counts {
  try {
    const raw = window.localStorage.getItem(KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as Counts;
      if (parsed.day === today(now) && parsed.n) return parsed;
    }
  } catch {
    /* storage blocked: no cap */
  }
  return { day: today(now), n: {} };
}

export function capReached(id: number, now: Date = new Date()): boolean {
  return (read(now).n[String(id)] ?? 0) >= MAX_PER_DAY;
}

export function recordShown(id: number, now: Date = new Date()): void {
  const c = read(now);
  c.n[String(id)] = (c.n[String(id)] ?? 0) + 1;
  try {
    window.localStorage.setItem(KEY, JSON.stringify(c));
  } catch {
    /* ignore */
  }
}

/** Paths where an ad must never render, even if a slot is mounted there by mistake. */
const BLOCKED = /^\/(?:[a-z]{2}\/)?(?:teen|loss|emergency|safety|consent|health-record|labs|doctor|assistant|telemed)(?:\/|$)/;

export function pathAllowsAds(pathname: string): boolean {
  return !BLOCKED.test(pathname);
}
