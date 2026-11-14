'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import { careKeys } from '@/entities/care-reminder';
import { reminderKeys } from '@/entities/reminder';
import { useCurrentUser } from '@/entities/user';
import { onBeforeLogout, onSessionEnd } from '@/shared/session';

import { createSyncGate, getNotificationChannel, sync, teardown } from '../model/channel';

/** The settings screen's query-key root (screens/notification-settings; a literal, features can't import screens). */
const NOTIFICATION_SETTINGS_ROOT = 'notification-settings';
/** Mutations land as a burst of invalidations; wait for it to settle, then sync once. */
const CHANGE_DEBOUNCE_MS = 2000;

/**
 * Mounted once in the `[locale]` layout. For a signed-in user it keeps this
 * device's reminders armed: on start, when the app returns to the foreground
 * (throttled), and after anything that changes the schedule — medications,
 * ticked doses, reminders, notification settings — detected through the query
 * cache so no mutation needs touching. On an explicit logout it removes this
 * device's push subscription / clears the shell's alarms. Renders nothing.
 */
export function NotificationSync() {
  const queryClient = useQueryClient();
  const userId = useCurrentUser().data?.id;

  useEffect(() => {
    if (userId === undefined) return;
    const channel = getNotificationChannel();
    if (channel === 'unsupported') return;
    const gate = createSyncGate(channel);
    const run = (force: boolean) => {
      if (gate.tryEnter(force)) void sync();
    };
    run(true);

    const onVisible = () => {
      if (document.visibilityState === 'visible') run(false);
    };
    document.addEventListener('visibilitychange', onVisible);
    window.addEventListener('focus', onVisible);

    let timer: ReturnType<typeof setTimeout> | undefined;
    const unsubscribeCache = queryClient.getQueryCache().subscribe((event) => {
      // Web push needs no re-sync per change: the server decides what to send. Only the shell's local alarms do.
      if (channel !== 'native' || event.type !== 'updated' || event.action.type !== 'success') return;
      const root = event.query.queryKey[0];
      const changes =
        root === careKeys.all[0] || root === reminderKeys.all[0] || root === NOTIFICATION_SETTINGS_ROOT;
      if (!changes) return;
      clearTimeout(timer);
      timer = setTimeout(() => run(true), CHANGE_DEBOUNCE_MS);
    });

    const offBeforeLogout = onBeforeLogout(() => teardown(true));
    const offSessionEnd = onSessionEnd(() => teardown(false));

    return () => {
      clearTimeout(timer);
      unsubscribeCache();
      offBeforeLogout();
      offSessionEnd();
      document.removeEventListener('visibilitychange', onVisible);
      window.removeEventListener('focus', onVisible);
    };
  }, [queryClient, userId]);

  return null;
}
