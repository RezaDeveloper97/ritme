/**
 * The Android shell's bridge (`window.RitmeNative`, synchronous
 * `@JavascriptInterface` methods). Older shell builds have no bridge at all —
 * every wrapper then reports "not there" instead of throwing.
 */
export const NATIVE_NOTIFICATIONS_EVENT = 'ritme:native-notifications';

interface RitmeNativeBridge {
  bridgeVersion?: () => number;
  scheduleReminders?: (json: string) => void;
  notificationsState?: () => string;
  requestNotifications?: () => void;
}

export interface NativeReminderItem {
  id: string | number;
  at_ms: number;
  title: string;
  body: string;
  url: string;
}

function bridge(): RitmeNativeBridge | null {
  if (typeof window === 'undefined') return null;
  const b = (window as unknown as { RitmeNative?: RitmeNativeBridge }).RitmeNative;
  return b && typeof b.scheduleReminders === 'function' ? b : null;
}

export function hasNativeBridge(): boolean {
  return bridge() !== null;
}

export function nativeNotificationsState(): 'granted' | 'denied' | 'prompt' {
  try {
    const s = bridge()?.notificationsState?.();
    return s === 'granted' || s === 'denied' ? s : 'prompt';
  } catch {
    return 'prompt';
  }
}

/** Replaces ALL local alarms. An empty list clears them. Returns false if the bridge refused. */
export function scheduleNativeReminders(items: readonly NativeReminderItem[]): boolean {
  try {
    const b = bridge();
    if (!b) return false;
    b.scheduleReminders?.(JSON.stringify({ items }));
    return true;
  } catch {
    return false;
  }
}

/** Shows the Android 13+ prompt; resolves with the answer (the event also fires when no prompt was needed). */
export function requestNativeNotifications(timeoutMs = 60_000): Promise<'granted' | 'denied' | 'prompt'> {
  return new Promise((resolve) => {
    const finish = () => {
      window.removeEventListener(NATIVE_NOTIFICATIONS_EVENT, onEvent);
      clearTimeout(timer);
      resolve(nativeNotificationsState());
    };
    const onEvent = () => finish();
    const timer = setTimeout(finish, timeoutMs);
    window.addEventListener(NATIVE_NOTIFICATIONS_EVENT, onEvent);
    try {
      bridge()?.requestNotifications?.();
    } catch {
      finish();
    }
  });
}
