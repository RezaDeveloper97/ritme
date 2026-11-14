/** Browser Push API helpers (no network): the app-specific POST/DELETE live in features/push-notifications. */
import { urlBase64ToBytes } from './urls';

export interface PushSubscriptionBody {
  endpoint: string;
  keys: { p256dh: string; auth: string };
}

export function currentPermission(): NotificationPermission {
  return typeof Notification === 'undefined' ? 'denied' : Notification.permission;
}

export async function getRegistration(): Promise<ServiceWorkerRegistration | null> {
  try {
    // `ready` never settles when no worker is registered (dev mode); don't hang the UI on it.
    const existing = await navigator.serviceWorker.getRegistration();
    if (!existing) return null;
    return await navigator.serviceWorker.ready;
  } catch {
    return null;
  }
}

export async function getExistingSubscription(): Promise<PushSubscription | null> {
  const registration = await getRegistration();
  return registration ? registration.pushManager.getSubscription() : null;
}

/** Subscribes (or returns the existing subscription). Permission must already be granted. */
export async function subscribeToPush(publicKey: string): Promise<PushSubscription | null> {
  const registration = await getRegistration();
  if (!registration) return null;
  const existing = await registration.pushManager.getSubscription();
  if (existing) return existing;
  return registration.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey: urlBase64ToBytes(publicKey),
  });
}

/** `PushSubscription.toJSON()` narrowed to what the API takes; null if a key is missing. */
export function toSubscriptionBody(subscription: PushSubscription): PushSubscriptionBody | null {
  const json = subscription.toJSON();
  const p256dh = json.keys?.p256dh;
  const auth = json.keys?.auth;
  if (!json.endpoint || !p256dh || !auth) return null;
  return { endpoint: json.endpoint, keys: { p256dh, auth } };
}
