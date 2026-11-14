'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import { plusKeys } from '@/entities/plus';
import { useCurrentUser } from '@/entities/user';
import { getBillingChannel } from '@/shared/billing';

import { restoreStorePurchases } from '../api/store';

let syncedUserId: number | string | null = null;

/**
 * Mounted once in the `[locale]` layout. In a store build (Bazaar / Myket
 * shell) a signed-in user's store purchases are re-synced once per session, so
 * a renewal or a reinstall shows up without touching «بازیابی خرید». Failures
 * are silent — it is a best-effort background check. Renders nothing.
 */
export function StoreBillingSync() {
  const queryClient = useQueryClient();
  const userId = useCurrentUser().data?.id;

  useEffect(() => {
    if (userId === undefined || syncedUserId === userId) return;
    const store = getBillingChannel();
    if (store === 'web') return;
    syncedUserId = userId;
    void restoreStorePurchases(store)
      .then((result) => {
        if (result) {
          queryClient.setQueryData(plusKeys.status(), result.status);
          void queryClient.invalidateQueries({ queryKey: plusKeys.all });
        }
      })
      .catch(() => {
        syncedUserId = null; // let a later mount retry
      });
  }, [userId, queryClient]);

  return null;
}
