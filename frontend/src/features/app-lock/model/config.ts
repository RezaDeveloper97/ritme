import { z } from 'zod';

/**
 * Device-local app-lock settings (B-N1-12). Never sent to the server: the lock
 * protects this device's copy of the session, so it lives next to the token in
 * localStorage and is wiped with the session (`onSessionEnd`, see `controller.ts`).
 */

/** localStorage key of the lock config; read by the pre-paint script too. */
export const LOCK_KEY = 'ritme_app_lock';
/** «پنهان کردن پیش‌نمایش» — a device preference, not per-user data. */
export const HIDE_PREVIEW_KEY = 'ritme_hide_preview';

/** Minutes in the background after which a resume asks again. 0 = every time. */
export const LOCK_TIMEOUTS = [0, 1, 5, 15] as const;
export type LockTimeout = (typeof LOCK_TIMEOUTS)[number];
export const DEFAULT_LOCK_TIMEOUT: LockTimeout = 1;

const configSchema = z.object({
  v: z.literal(1),
  salt: z.string().min(1),
  hash: z.string().min(1),
  iterations: z.number().int().positive(),
  length: z.number().int().min(4).max(6),
  timeoutMin: z.union([z.literal(0), z.literal(1), z.literal(5), z.literal(15)]),
  /** WebAuthn platform credential id (base64url) when biometrics are on. */
  credentialId: z.string().nullable(),
  /** Wrong passcodes in a row; persisted so a reload cannot reset the back-off. */
  failures: z.number().int().min(0),
  /** Epoch ms until which no passcode is accepted (back-off after 5 failures). */
  blockedUntil: z.number().int().min(0),
});

export type LockConfig = z.infer<typeof configSchema>;

/** The storage the lock needs — `window.localStorage` in the app, a Map in tests. */
export interface KeyValueStore {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
  removeItem(key: string): void;
}

/**
 * The stored config, or null when the lock is off. A config that exists but
 * cannot be parsed is treated as **on with no usable passcode**: returning null
 * there would let anyone disable the lock by corrupting the entry, so the caller
 * gets a sentinel that no passcode matches (recovery is «فراموشی رمز» → sign in).
 */
export function readConfig(storage: KeyValueStore): LockConfig | null {
  let raw: string | null;
  try {
    raw = storage.getItem(LOCK_KEY);
  } catch {
    return null;
  }
  if (raw === null) return null;
  try {
    return configSchema.parse(JSON.parse(raw));
  } catch {
    return {
      v: 1,
      salt: 'AA==',
      hash: '!', // never produced by hashPasscode
      iterations: 1,
      length: 4,
      timeoutMin: 0,
      credentialId: null,
      failures: 0,
      blockedUntil: 0,
    };
  }
}

export function writeConfig(storage: KeyValueStore, config: LockConfig): void {
  storage.setItem(LOCK_KEY, JSON.stringify(config));
}

export function clearConfig(storage: KeyValueStore): void {
  try {
    storage.removeItem(LOCK_KEY);
  } catch {
    // Blocked storage: nothing was stored either.
  }
}

export function readHidePreview(storage: KeyValueStore): boolean {
  try {
    return storage.getItem(HIDE_PREVIEW_KEY) === '1';
  } catch {
    return false;
  }
}

export function writeHidePreview(storage: KeyValueStore, on: boolean): void {
  try {
    if (on) storage.setItem(HIDE_PREVIEW_KEY, '1');
    else storage.removeItem(HIDE_PREVIEW_KEY);
  } catch {
    // Blocked storage: the preference just doesn't stick.
  }
}

/**
 * Inline pre-paint script (runs before React): marks `<html data-app-locked>`
 * when a lock is configured, so CSS hides the app shell until the gate has
 * decided — no flash of content on open or reload. Plain string, no imports.
 */
export const appLockInitScript = `(function(){try{if(localStorage.getItem(${JSON.stringify(LOCK_KEY)}))document.documentElement.setAttribute('data-app-locked','');}catch(e){}})();`;
