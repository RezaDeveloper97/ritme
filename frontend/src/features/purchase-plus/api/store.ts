'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import {
  plusKeys,
  storeIntentSchema,
  storeSyncSchema,
  storeVerificationSchema,
  type PlusStoreIntent,
  type PlusStoreSync,
  type PlusStoreVerification,
} from '@/entities/plus';
import { track } from '@/shared/analytics';
import { type ApiEnvelope, apiClient, getApiErrorMessage } from '@/shared/api';
import {
  StoreBillingError,
  storePurchase,
  storeQueryPurchases,
  type StoreChannel,
  type StorePurchase,
} from '@/shared/billing';

/*
 * Cafe Bazaar / Myket purchases (android-shell store builds). The server owns
 * pricing and trust: it hands out the SKU payload (intent) and verifies the
 * signed purchase the store returns (verify / sync).
 */

export interface StorePurchaseInput {
  store: StoreChannel;
  sku: string;
}

/** POST /plus/store/intent */
export async function fetchStoreIntent(store: StoreChannel, sku: string): Promise<PlusStoreIntent> {
  const { data } = await apiClient.post<ApiEnvelope<unknown>>('/plus/store/intent', { store, sku });
  return storeIntentSchema.parse(data.data);
}

/** POST /plus/store/verify */
export async function verifyStorePurchase(store: StoreChannel, purchase: StorePurchase): Promise<PlusStoreVerification> {
  const { data } = await apiClient.post<ApiEnvelope<unknown>>('/plus/store/verify', {
    store,
    signed_data: purchase.signed_data,
    signature: purchase.signature,
  });
  return { ...storeVerificationSchema.parse(data.data), message: data.message ?? null };
}

/** POST /plus/store/sync — re-checks everything the store account owns. */
export async function syncStorePurchases(store: StoreChannel, purchases: readonly StorePurchase[]): Promise<PlusStoreSync> {
  const { data } = await apiClient.post<ApiEnvelope<unknown>>('/plus/store/sync', {
    store,
    purchases: purchases.map((p) => ({ signed_data: p.signed_data, signature: p.signature })),
  });
  return storeSyncSchema.parse(data.data);
}

/** intent → the store's purchase screen → verify the purchase of that SKU. */
export async function buyThroughStore({ store, sku }: StorePurchaseInput): Promise<PlusStoreVerification> {
  const intent = await fetchStoreIntent(store, sku);
  const purchases = await storePurchase(intent.sku, intent.payload);
  const bought = purchases.find((p) => p.sku === intent.sku);
  if (!bought) throw new StoreBillingError('failed');
  return verifyStorePurchase(store, bought);
}

/** The user closed the store screen: nothing to tell her. */
export function isStoreCancelled(error: unknown): boolean {
  return error instanceof StoreBillingError && error.reason === 'cancelled';
}

/** The server's message for a failed intent / verify, else null (store-side failures have none). */
export function storeErrorMessage(error: unknown): string | null {
  return getApiErrorMessage(error) ?? null;
}

/** Buys a plan or the trial SKU through the store the shell reports. */
export function useStorePurchase() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: buyThroughStore,
    onSuccess: (result) => {
      queryClient.setQueryData(plusKeys.status(), result.status);
      void queryClient.invalidateQueries({ queryKey: plusKeys.all });
      if (result.invoice?.status === 'paid' && !result.alreadyVerified) {
        track({
          name: 'plus_purchase',
          props: { plan_months: Math.min(12, Math.max(1, result.invoice.durationMonths)) },
        });
      }
    },
  });
}

/** Restores purchases already owned in the store (silent app-start sync). */
export async function restoreStorePurchases(store: StoreChannel): Promise<PlusStoreSync | null> {
  const owned = await storeQueryPurchases();
  return owned.length ? syncStorePurchases(store, owned) : null;
}
