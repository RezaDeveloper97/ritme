import {
  clearConfig,
  DEFAULT_LOCK_TIMEOUT,
  type KeyValueStore,
  type LockConfig,
  type LockTimeout,
  readConfig,
  readHidePreview,
  writeConfig,
  writeHidePreview,
} from './config';
import { digestEquals, fromBase64, hashPasscode, isValidPasscode, PBKDF2_ITERATIONS, randomSalt, toBase64 } from './passcode';

/**
 * The app-lock state machine (B-N1-12), free of React and the DOM so it can be
 * tested as a whole (`controller.test.ts`).
 *
 * The one rule that makes the lock unbypassable: **«unlocked» exists only in
 * this object's memory.** It is never written to storage, the URL, history
 * state or a cookie. A reload or a new tab builds a new controller, which
 * starts locked whenever a config exists; back/forward navigation never
 * rebuilds it and never touches `locked`. The gate is mounted in the locale
 * layout, above every route, so no route renders while `locked` is true.
 */

/** Wrong passcodes allowed before the back-off starts. */
export const FREE_ATTEMPTS = 5;
/** Wrong passcodes in a row after which the session is ended (like «فراموشی رمز»). */
export const MAX_FAILURES = 10;
const BACKOFF_BASE_MS = 30_000;
const BACKOFF_MAX_MS = 5 * 60_000;

export interface LockSnapshot {
  /** A passcode is configured. */
  enabled: boolean;
  /** The app content must not render. */
  locked: boolean;
  /** Biometric (WebAuthn platform) unlock is set up. */
  biometric: boolean;
  timeoutMin: LockTimeout;
  /** Digits of the configured passcode (4–6). */
  length: number;
  /** Epoch ms until which passcodes are refused (0 = not blocked). */
  blockedUntil: number;
  /** MAX_FAILURES wrong passcodes: the UI must end the session (logout + wipe). */
  lockedOut: boolean;
  /** Blur the app when it goes to the background (app switcher preview). */
  hidePreview: boolean;
  /** The app is in the background right now (drives the preview veil). */
  hidden: boolean;
}

export type UnlockResult = 'ok' | 'wrong' | 'blocked' | 'lockedOut';

export interface LockControllerDeps {
  storage: KeyValueStore;
  now: () => number;
}

export interface EnableOptions {
  passcode: string;
  credentialId?: string | null;
  timeoutMin?: LockTimeout;
}

export function backoffMs(failures: number): number {
  if (failures < FREE_ATTEMPTS) return 0;
  return Math.min(BACKOFF_BASE_MS * 2 ** (failures - FREE_ATTEMPTS), BACKOFF_MAX_MS);
}

export function createLockController({ storage, now }: LockControllerDeps) {
  let config: LockConfig | null = readConfig(storage);
  // A fresh controller = a fresh page load: locked whenever a lock is set up.
  let locked = config !== null;
  let hidePreview = readHidePreview(storage);
  let hidden = false;
  let hiddenAt: number | null = null;
  let snapshot = build();
  const listeners = new Set<() => void>();

  function build(): LockSnapshot {
    return {
      enabled: config !== null,
      locked,
      biometric: Boolean(config?.credentialId),
      timeoutMin: config?.timeoutMin ?? DEFAULT_LOCK_TIMEOUT,
      length: config?.length ?? 4,
      blockedUntil: config && config.blockedUntil > now() ? config.blockedUntil : 0,
      lockedOut: (config?.failures ?? 0) >= MAX_FAILURES,
      hidePreview,
      hidden,
    };
  }

  function emit() {
    snapshot = build();
    for (const l of listeners) l();
  }

  function save(next: LockConfig | null) {
    config = next;
    if (next) writeConfig(storage, next);
    else clearConfig(storage);
  }

  /** Count a wrong passcode (persisted); at MAX_FAILURES the app locks and the session must end. */
  function recordFailure(current: LockConfig): UnlockResult {
    const failures = current.failures + 1;
    const wait = backoffMs(failures);
    save({ ...current, failures, blockedUntil: wait ? now() + wait : 0 });
    if (failures >= MAX_FAILURES) {
      locked = true;
      emit();
      return 'lockedOut';
    }
    emit();
    return wait ? 'blocked' : 'wrong';
  }

  /** The stored config is the truth: another tab may have counted failures or changed the passcode. */
  function reload() {
    config = readConfig(storage);
  }

  return {
    subscribe(listener: () => void): () => void {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    getSnapshot(): LockSnapshot {
      return snapshot;
    },

    /** Check a passcode; on success the app unlocks (in memory only). */
    async unlockWithPasscode(passcode: string): Promise<UnlockResult> {
      reload();
      if (!config) {
        locked = false;
        emit();
        return 'ok';
      }
      if (config.failures >= MAX_FAILURES) return recordFailure(config);
      if (config.blockedUntil > now()) return 'blocked';
      const digest = isValidPasscode(passcode)
        ? await hashPasscode(passcode, fromBase64(config.salt), config.iterations)
        : '';
      if (digest && digestEquals(digest, config.hash)) {
        save({ ...config, failures: 0, blockedUntil: 0 });
        locked = false;
        emit();
        return 'ok';
      }
      return recordFailure(config);
    },

    /** The WebAuthn credential id of the biometric unlock, if set up. */
    getCredentialId(): string | null {
      return config?.credentialId ?? null;
    },

    /** The UI verified the platform authenticator (WebAuthn assertion) — unlock. */
    unlockWithBiometric(): void {
      if (!config?.credentialId) return;
      save({ ...config, failures: 0, blockedUntil: 0 });
      locked = false;
      emit();
    },

    /** Verify the current passcode without changing the lock state (to turn the lock off). */
    async verifyPasscode(passcode: string): Promise<boolean> {
      reload();
      if (!config || config.blockedUntil > now()) return false;
      const digest = isValidPasscode(passcode)
        ? await hashPasscode(passcode, fromBase64(config.salt), config.iterations)
        : '';
      if (digest && config.failures < MAX_FAILURES && digestEquals(digest, config.hash)) {
        save({ ...config, failures: 0, blockedUntil: 0 });
        emit();
        return true;
      }
      recordFailure(config);
      return false;
    },

    /** Turn the lock on with a new passcode. The current session stays unlocked. */
    async enable({ passcode, credentialId = null, timeoutMin }: EnableOptions): Promise<void> {
      if (!isValidPasscode(passcode)) throw new Error('app-lock: invalid passcode');
      const salt = randomSalt();
      save({
        v: 1,
        salt: toBase64(salt),
        hash: await hashPasscode(passcode, salt, PBKDF2_ITERATIONS),
        iterations: PBKDF2_ITERATIONS,
        length: passcode.length,
        timeoutMin: timeoutMin ?? config?.timeoutMin ?? DEFAULT_LOCK_TIMEOUT,
        credentialId,
        failures: 0,
        blockedUntil: 0,
      });
      locked = false;
      emit();
    },

    /** Turn the lock off (the caller verified the passcode first). */
    disable(): void {
      save(null);
      locked = false;
      emit();
    },

    setTimeoutMin(timeoutMin: LockTimeout): void {
      if (!config) return;
      save({ ...config, timeoutMin });
      emit();
    },

    setBiometric(credentialId: string | null): void {
      if (!config) return;
      save({ ...config, credentialId });
      emit();
    },

    setHidePreview(on: boolean): void {
      hidePreview = on;
      writeHidePreview(storage, on);
      emit();
    },

    /** Lock now (e.g. from a «قفل کن» action). */
    lock(): void {
      if (!config) return;
      locked = true;
      emit();
    },

    /** The page went to the background (`visibilitychange` hidden / `pagehide`). */
    onHidden(): void {
      if (hidden) return;
      hidden = true;
      hiddenAt = now();
      emit();
    },

    /**
     * The page is visible again (`visibilitychange` visible / `pageshow`,
     * including a bfcache restore). Locks when it was away for the timeout.
     */
    onVisible(): void {
      const awayFor = hiddenAt === null ? 0 : now() - hiddenAt;
      hidden = false;
      hiddenAt = null;
      // The config may have changed in another tab (turned on/off).
      config = readConfig(storage);
      if (!config) locked = false;
      else if (!locked && awayFor >= config.timeoutMin * 60_000) locked = true;
      emit();
    },

    /** The session ended: the lock belongs to it and goes with it. */
    reset(): void {
      save(null);
      locked = false;
      emit();
    },
  };
}

export type LockController = ReturnType<typeof createLockController>;
