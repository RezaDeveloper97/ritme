/**
 * Which way this device can get a reminder at dose time.
 *
 * - `native`  — inside the Android shell (`window.RitmeNative` bridge, local alarms).
 * - `webpush` — a browser / installed PWA with the Push API.
 * - `unsupported` — everything else (including an OLD shell build without the
 *   bridge: the shell has no Push API, so Web Push is never attempted there).
 *
 * Pure: takes the environment as data so every branch is unit-testable.
 */
import { isShellUserAgent } from '../shell';

export type NotificationChannel = 'native' | 'webpush' | 'unsupported';

/** What the UI shows for «this device». */
export type DeviceNotificationState =
  | 'on' // reminders will reach this device
  | 'off' // can be turned on (permission still askable, or granted but not yet subscribed)
  | 'blocked' // the user (or the system) said no; only device settings can undo it
  | 'install' // iOS Safari outside the home screen: push needs the installed app first
  | 'unsupported'; // no way to deliver here (or the server has push disabled)

export interface ChannelEnv {
  userAgent: string;
  hasNativeBridge: boolean;
  hasServiceWorker: boolean;
  hasPushManager: boolean;
  hasNotification: boolean;
  standalone: boolean;
}

export function detectChannel(env: ChannelEnv): NotificationChannel {
  if (env.hasNativeBridge) return 'native';
  // The shell without the bridge (old build): no Push API there — never try it.
  if (isShellUserAgent(env.userAgent)) return 'unsupported';
  if (env.hasServiceWorker && env.hasPushManager && env.hasNotification) return 'webpush';
  return 'unsupported';
}

/** iPhone / iPad (iPadOS reports itself as a Mac with touch). */
export function isIosDevice(userAgent: string, maxTouchPoints = 0): boolean {
  return /iPad|iPhone|iPod/.test(userAgent) || (/Macintosh/.test(userAgent) && maxTouchPoints > 1);
}

/**
 * iOS only exposes the Push API to a web app added to the home screen. In a
 * plain Safari tab `PushManager` is missing, so the channel reads `unsupported`
 * — this tells that case apart so the UI says «install first» instead of
 * showing a dead button.
 */
export function needsInstallForPush(env: ChannelEnv, maxTouchPoints = 0): boolean {
  return (
    !env.hasNativeBridge &&
    !isShellUserAgent(env.userAgent) &&
    isIosDevice(env.userAgent, maxTouchPoints) &&
    !env.standalone &&
    !env.hasPushManager
  );
}

/** Map the browser's `Notification.permission` (+ whether a subscription exists) to the UI state. */
export function webPushState(permission: NotificationPermission, subscribed: boolean): DeviceNotificationState {
  if (permission === 'denied') return 'blocked';
  if (permission === 'granted' && subscribed) return 'on';
  return 'off';
}

/** Map the bridge's `notificationsState()`. */
export function nativeState(state: string): DeviceNotificationState {
  if (state === 'granted') return 'on';
  if (state === 'denied') return 'blocked';
  return 'off';
}

export function readChannelEnv(): ChannelEnv {
  if (typeof window === 'undefined') {
    return {
      userAgent: '',
      hasNativeBridge: false,
      hasServiceWorker: false,
      hasPushManager: false,
      hasNotification: false,
      standalone: false,
    };
  }
  const nav = window.navigator as Navigator & { standalone?: boolean };
  const bridge = (window as unknown as { RitmeNative?: { scheduleReminders?: unknown } }).RitmeNative;
  return {
    userAgent: nav.userAgent,
    hasNativeBridge: !!bridge && typeof bridge.scheduleReminders === 'function',
    hasServiceWorker: 'serviceWorker' in nav,
    hasPushManager: 'PushManager' in window,
    hasNotification: 'Notification' in window,
    standalone:
      nav.standalone === true ||
      window.matchMedia('(display-mode: standalone), (display-mode: fullscreen), (display-mode: minimal-ui)').matches,
  };
}
