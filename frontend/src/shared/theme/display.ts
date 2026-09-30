'use client';

import { create } from 'zustand';

/**
 * Display preferences next to the theme (B-N1-10, `nbl_Me_Appearance`):
 * text size and reduced motion. Client/UI state only (CLAUDE.md §8) —
 * persisted to localStorage and mirrored onto <html> so CSS can react:
 *
 * - `--text-scale` on <html style> — globals.css zooms each screen's scroll
 *   area by it. The app sizes type in px (tokens), so changing the root
 *   font-size alone would move nothing; zoom scales the reading surface as a
 *   whole, which is what the artboard's «اندازه متن» slider previews.
 * - `data-motion` on <html>: `reduce` switches animations/transitions off
 *   app-wide (CSS block in globals.css), whatever the OS says. Absent =
 *   follow the OS (`prefers-reduced-motion`).
 *
 * The pre-paint script (`themeInitScript`) applies both before first paint,
 * so a user with large text never sees the page jump.
 */
export const TEXT_SCALE_KEY = 'ritme_text_scale';
export const MOTION_KEY = 'ritme_motion';

/** Slider stops, smallest first; the middle one is the default (artboard thumb at 50%). */
export const TEXT_SCALES = [0.9, 0.95, 1, 1.1, 1.2] as const;
export const DEFAULT_TEXT_SCALE_INDEX = 2;

/** `system` follows prefers-reduced-motion; `reduce` forces it on. */
export type MotionPreference = 'system' | 'reduce';

const REDUCED_QUERY = '(prefers-reduced-motion: reduce)';

const isBrowser = (): boolean => typeof window !== 'undefined';

export function clampTextScaleIndex(value: unknown): number {
  const n = typeof value === 'number' ? value : Number.parseInt(String(value), 10);
  if (!Number.isInteger(n) || n < 0 || n >= TEXT_SCALES.length) return DEFAULT_TEXT_SCALE_INDEX;
  return n;
}

export function isMotionPreference(value: unknown): value is MotionPreference {
  return value === 'system' || value === 'reduce';
}

/** Does the OS ask for reduced motion right now (false on the server)? */
export function systemReducesMotion(): boolean {
  if (!isBrowser() || typeof window.matchMedia !== 'function') return false;
  return window.matchMedia(REDUCED_QUERY).matches;
}

/** Whether motion is reduced for this preference, given the OS setting. */
export function reducesMotion(preference: MotionPreference, system: boolean = systemReducesMotion()): boolean {
  return preference === 'reduce' || system;
}

export function applyTextScale(index: number): void {
  if (!isBrowser()) return;
  document.documentElement.style.setProperty('--text-scale', String(TEXT_SCALES[clampTextScaleIndex(index)]));
}

export function applyMotion(preference: MotionPreference): void {
  if (!isBrowser()) return;
  const root = document.documentElement;
  if (preference === 'reduce') root.dataset.motion = 'reduce';
  else delete root.dataset.motion;
}

function read(key: string): string | null {
  if (!isBrowser()) return null;
  try {
    return window.localStorage.getItem(key);
  } catch {
    return null;
  }
}

function write(key: string, value: string): void {
  if (!isBrowser()) return;
  try {
    window.localStorage.setItem(key, value);
  } catch {
    // Private mode / blocked storage: the choice still applies this session.
  }
}

interface DisplayState {
  /** Index into {@link TEXT_SCALES}. */
  textScale: number;
  motion: MotionPreference;
  setTextScale: (index: number) => void;
  setMotion: (motion: MotionPreference) => void;
}

const storedMotion = read(MOTION_KEY);

export const useDisplayStore = create<DisplayState>((set) => ({
  textScale: clampTextScaleIndex(read(TEXT_SCALE_KEY)),
  motion: isMotionPreference(storedMotion) ? storedMotion : 'system',
  setTextScale: (index) => {
    const textScale = clampTextScaleIndex(index);
    write(TEXT_SCALE_KEY, String(textScale));
    applyTextScale(textScale);
    set({ textScale });
  },
  setMotion: (motion) => {
    write(MOTION_KEY, motion);
    applyMotion(motion);
    set({ motion });
  },
}));

/** Live OS reduced-motion changes; returns the unsubscribe. */
export function watchSystemMotion(onChange: () => void): () => void {
  if (!isBrowser() || typeof window.matchMedia !== 'function') return () => {};
  const media = window.matchMedia(REDUCED_QUERY);
  media.addEventListener('change', onChange);
  return () => media.removeEventListener('change', onChange);
}
