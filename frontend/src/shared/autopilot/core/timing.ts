import type { PilotTiming } from '../config';

/** 0 ≤ n < 1. Injected so tests are deterministic. */
export type Rng = () => number;

/** `ms` moved by up to ±`jitter` of itself — the unevenness that makes a pause look human. */
export function jittered(ms: number, jitter: number, rng: Rng): number {
  if (ms <= 0 || jitter <= 0) return Math.max(0, ms);
  return Math.max(0, Math.round(ms * (1 + (rng() * 2 - 1) * jitter)));
}

/** How long the pointer takes to cross `distancePx`, clamped to the configured window. */
export function cursorTravelMs(distancePx: number, timing: PilotTiming): number {
  if (timing.cursorMaxMs <= 0) return 0;
  const raw = (Math.abs(distancePx) / timing.cursorSpeedPxPerSec) * 1000;
  return Math.round(Math.min(timing.cursorMaxMs, Math.max(timing.cursorMinMs, raw)));
}

const JOINERS = /[‌‍̀-ًͯ-ٰٟ]/;

/**
 * The successive states of a field while `text` is typed into it.
 *
 * Splits by code point, not UTF-16 unit, and keeps a ZWNJ / combining mark with the letter before it, so
 * Persian never shows a half-typed «می‌» or a floating diacritic.
 */
export function typingFrames(text: string, mode: 'perChar' | 'perWord' | 'instant'): string[] {
  if (text === '') return [''];
  if (mode === 'instant') return [text];
  const units: string[] = [];
  for (const ch of Array.from(text)) {
    if (units.length && JOINERS.test(ch)) units[units.length - 1] += ch;
    else units.push(ch);
  }
  const frames: string[] = [];
  let acc = '';
  units.forEach((unit, i) => {
    acc += unit;
    const wordEnd = /\s/.test(unit) || i === units.length - 1;
    if (mode === 'perChar' || wordEnd) frames.push(acc);
  });
  return frames;
}

/** Pause after showing `frame`: a character's time, plus a breath at the end of a word. */
export function typingDelay(frame: string, timing: PilotTiming, rng: Rng): number {
  const wordEnd = /\s$/.test(frame);
  return jittered(timing.typeCharMs + (wordEnd ? timing.typeWordPauseMs : 0), timing.jitter, rng);
}
