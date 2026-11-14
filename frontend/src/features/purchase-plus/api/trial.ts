'use client';

import { useCallback } from 'react';

import type { PlusCatalog } from '@/entities/plus';
import { useBillingChannel } from '@/shared/billing';

import { useStartTrial } from './mutations';
import { isStoreCancelled, useStorePurchase } from './store';

interface TrialCallbacks {
  onSuccess?: () => void;
  onError?: (error: unknown) => void;
}

/**
 * Starts the one free trial through the right channel: the store's trial SKU
 * in a Bazaar / Myket build when the server sells it there, otherwise
 * POST /plus/trial/start. A cancelled store screen is silent.
 */
export function useStartPlusTrial(catalog: PlusCatalog | undefined) {
  const { channel } = useBillingChannel();
  const web = useStartTrial();
  const store = useStorePurchase();
  const trialSku = channel !== 'web' ? (catalog?.storeTrialSku ?? null) : null;

  const start = useCallback(
    ({ onSuccess, onError }: TrialCallbacks = {}) => {
      if (channel !== 'web' && trialSku) {
        store.mutate(
          { store: channel, sku: trialSku },
          { onSuccess: () => onSuccess?.(), onError: (e) => (isStoreCancelled(e) ? undefined : onError?.(e)) },
        );
      } else {
        web.mutate(undefined, { onSuccess: () => onSuccess?.(), onError: (e) => onError?.(e) });
      }
    },
    [channel, trialSku, store, web],
  );

  return { start, isPending: web.isPending || store.isPending };
}
