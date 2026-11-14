import { getApiErrorStatus } from '@/shared/api';
import {
  createGate,
  currentPermission,
  detectChannel,
  type DeviceNotificationState,
  getExistingSubscription,
  hasNativeBridge,
  nativeNotificationsState,
  nativeState,
  needsInstallForPush,
  type NotificationChannel,
  readChannelEnv,
  requestNativeNotifications,
  scheduleNativeReminders,
  subscribeToPush,
  toSubscriptionBody,
  webPushState,
} from '@/shared/pwa';

import {
  deletePushSubscription,
  fetchPushPublicKey,
  fetchUpcomingReminders,
  postPushSubscription,
} from '../api/requests';

/**
 * One small interface over both delivery channels so screens never branch on
 * platform: `getNotificationChannel()`, `getState()`, `enable()`, `sync()`.
 * Never logs anything (notification text is health data, §11).
 */

export function getNotificationChannel(): NotificationChannel {
  return detectChannel(readChannelEnv());
}

/** The VAPID key, fetched once per page load (null = push disabled on the server; failures aren't cached). */
let publicKeyPromise: Promise<string | null> | null = null;
function publicKey(): Promise<string | null> {
  publicKeyPromise ??= fetchPushPublicKey().catch((error: unknown) => {
    publicKeyPromise = null;
    throw error;
  });
  return publicKeyPromise;
}

export async function getState(): Promise<DeviceNotificationState> {
  const env = readChannelEnv();
  const channel = detectChannel(env);
  if (channel === 'native') return nativeState(nativeNotificationsState());
  if (channel === 'unsupported') return needsInstallForPush(env, navigator.maxTouchPoints) ? 'install' : 'unsupported';
  try {
    if ((await publicKey()) === null) return 'unsupported';
  } catch {
    return 'unsupported';
  }
  const permission = currentPermission();
  if (permission !== 'granted') return webPushState(permission, false);
  return webPushState(permission, (await getExistingSubscription()) !== null);
}

/** Native: the shell's local alarms, replaced wholesale from the server's schedule. */
async function syncNative(): Promise<void> {
  if (nativeNotificationsState() !== 'granted') return;
  scheduleNativeReminders(await fetchUpcomingReminders(168));
}

/** Web push: (re)register this browser's subscription with the server. Silent: never prompts. */
async function syncWebPush(): Promise<void> {
  if (currentPermission() !== 'granted') return;
  const key = await publicKey();
  if (key === null) return;
  const subscription = await subscribeToPush(key);
  const body = subscription ? toSubscriptionBody(subscription) : null;
  if (body) await postPushSubscription(body);
}

/** Bring the device in line with the server. Safe to call often; never prompts. */
export async function sync(): Promise<void> {
  try {
    const channel = getNotificationChannel();
    if (channel === 'native') await syncNative();
    else if (channel === 'webpush') await syncWebPush();
  } catch (error) {
    // A flaky network or a 401 must not surface from a background sync.
    if (getApiErrorStatus(error) === 401) return;
  }
}

/**
 * Ask in context (the caller shows its own explainer first), then arm the
 * channel. Returns the resulting state.
 */
export async function enable(): Promise<DeviceNotificationState> {
  const channel = getNotificationChannel();
  if (channel === 'native') {
    if (nativeNotificationsState() !== 'granted') await requestNativeNotifications();
    await sync();
    return getState();
  }
  if (channel === 'webpush') {
    if (currentPermission() === 'default') {
      try {
        await Notification.requestPermission();
      } catch {
        // Older Safari takes a callback only; the state read below tells the truth.
      }
    }
    await sync();
  }
  return getState();
}

/** Logout / session end. `authed` = the token still works, so the server copy can go too. */
export async function teardown(authed: boolean): Promise<void> {
  if (hasNativeBridge()) {
    scheduleNativeReminders([]);
    return;
  }
  if (getNotificationChannel() !== 'webpush') return;
  try {
    const subscription = await getExistingSubscription();
    if (!subscription) return;
    if (authed) await deletePushSubscription(subscription.endpoint).catch(() => undefined);
    await subscription.unsubscribe();
  } catch {
    // Best effort: the next sign-in re-registers anyway.
  }
}

/** Minimum gap between background syncs: native is cheap and alarm-critical, web push only re-registers. */
export const SYNC_GAP_MS = { native: 60_000, webpush: 6 * 60 * 60_000, unsupported: Number.POSITIVE_INFINITY } as const;

export function createSyncGate(channel: NotificationChannel) {
  return createGate(SYNC_GAP_MS[channel]);
}

/** Is a permission prompt still possible (so an in-context explainer makes sense)? */
export async function canAsk(): Promise<boolean> {
  const channel = getNotificationChannel();
  if (channel === 'native') return nativeNotificationsState() === 'prompt';
  if (channel !== 'webpush') return false;
  if (currentPermission() !== 'default') return false;
  try {
    return (await publicKey()) !== null;
  } catch {
    return false;
  }
}
