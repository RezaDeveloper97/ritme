/** A short buzz where the device supports it (Android Chrome); a silent no-op elsewhere (iOS Safari). */
export function buzz(pattern: number | number[] = 15): void {
  try {
    if (typeof navigator !== 'undefined' && typeof navigator.vibrate === 'function') navigator.vibrate(pattern);
  } catch {
    // Vibration blocked (no user activation, permissions policy): haptics are a nicety only.
  }
}
