// Public API of the `app-lock` feature (B-N1-12). Import only from here (§3.3).
// The gate is mounted once by the locale layout; the Privacy screen uses the rest.
export { AppLockGate } from './ui/AppLockGate';
export { PasscodeSheet } from './ui/PasscodeSheet';
export { appLockInitScript, LOCK_TIMEOUTS, type LockTimeout } from './model/config';
export { getLockController, useAppLock } from './model/store';
export type { LockSnapshot } from './model/controller';
export { isBiometricAvailable, registerBiometric } from './model/webauthn';
