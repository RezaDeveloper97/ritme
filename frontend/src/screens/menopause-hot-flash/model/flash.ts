import {
  HOT_FLASH_SEVERITIES,
  type HotFlashDetails,
  type HotFlashSeverity,
  type HotFlashTrigger,
  type MenopauseFlash,
} from '@/entities/menopause';

/*
 * Pure helpers of the hot-flash timer screen (CB-MENO-07, nbl_Meno_HotFlash).
 */

/** The API closes a timer forgotten for an hour (docs/canvas-build/menopause.md §7). */
export const FLASH_CAP_S = 3600;

export const EMPTY_DETAILS: HotFlashDetails = { severity: null, sweat: false, triggers: [] };

/** The running timer's length now: the server's elapsed at fetch + local ticks, capped at the API's hour. */
export function flashElapsedSeconds(serverElapsedS: number, sinceFetchMs: number): number {
  return Math.min(FLASH_CAP_S, Math.max(0, serverElapsedS + Math.floor(sinceFetchMs / 1000)));
}

/** The details a flash already carries (resuming a running timer, editing the one just stopped). */
export function detailsOf(flash: MenopauseFlash): HotFlashDetails {
  return { severity: flash.severity, sweat: flash.sweat, triggers: [...flash.triggers] };
}

/** «نمی‌دانم» excludes the named causes and the other way round. */
export function toggleTrigger(current: readonly HotFlashTrigger[], code: HotFlashTrigger): HotFlashTrigger[] {
  if (current.includes(code)) return current.filter((c) => c !== code);
  if (code === 'unknown') return ['unknown'];
  return [...current.filter((c) => c !== 'unknown'), code];
}

/**
 * Wall-clock `HH:mm` of the start as the API wrote it (Tehran offset) — read
 * from the string, so a device in another zone still shows her local time.
 */
export function startClock(startedAt: string): string | null {
  const m = /T(\d{2}):(\d{2})/.exec(startedAt);
  return m ? `${m[1]}:${m[2]}` : null;
}

export type FlashLength = { unit: 'minutes' | 'seconds'; value: number };

/** A length for the list and the average tile: whole minutes from a minute up, else seconds. */
export function flashLength(seconds: number): FlashLength {
  const s = Math.max(0, Math.round(seconds));
  return s >= 60 ? { unit: 'minutes', value: Math.round(s / 60) } : { unit: 'seconds', value: s };
}

/** 0–3 position of a severity on the scale (drives the dot / tint ramp); −1 = not set. */
export function severityIndex(severity: HotFlashSeverity | null): number {
  return severity ? HOT_FLASH_SEVERITIES.indexOf(severity) : -1;
}
