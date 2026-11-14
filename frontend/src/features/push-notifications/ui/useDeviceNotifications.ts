'use client';

import { useCallback, useEffect, useState } from 'react';

import { type DeviceNotificationState, NATIVE_NOTIFICATIONS_EVENT, type NotificationChannel } from '@/shared/pwa';

import { enable as enableChannel, getNotificationChannel, getState } from '../model/channel';

/** This device's notification state for settings rows; `null` until it has been read. */
export function useDeviceNotifications() {
  const [state, setState] = useState<DeviceNotificationState | null>(null);
  const [channel, setChannel] = useState<NotificationChannel>('unsupported');
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState(false);

  const refresh = useCallback(async () => {
    setChannel(getNotificationChannel());
    setState(await getState());
  }, []);

  useEffect(() => {
    void refresh();
    // The user may flip the permission in the system settings and come back.
    const onVisible = () => {
      if (document.visibilityState === 'visible') void refresh();
    };
    document.addEventListener('visibilitychange', onVisible);
    window.addEventListener(NATIVE_NOTIFICATIONS_EVENT, onVisible);
    return () => {
      document.removeEventListener('visibilitychange', onVisible);
      window.removeEventListener(NATIVE_NOTIFICATIONS_EVENT, onVisible);
    };
  }, [refresh]);

  const enable = useCallback(async () => {
    setBusy(true);
    setFailed(false);
    try {
      const next = await enableChannel();
      setState(next);
      setFailed(next !== 'on');
    } catch {
      setFailed(true);
    } finally {
      setBusy(false);
    }
  }, []);

  return { state, channel, busy, failed, enable };
}
