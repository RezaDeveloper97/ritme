'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import {
  checkoutSchema,
  plusKeys,
  restoreSchema,
  statusSchema,
  verificationSchema,
  type GatewayReturn,
  type PlusCheckout,
  type PlusRestore,
  type PlusStatus,
  type PlusVerification,
} from '@/entities/plus';
import { type ApiEnvelope, apiClient } from '@/shared/api';

/*
 * The Plus purchase actions (B-N2-04 + B-N2-05). The server prices everything:
 * the body never carries an amount. Every write that can change the tier
 * refreshes the whole `plus` cache (status, history, quotes).
 */

export interface CheckoutInput {
  planId: number;
  /** A code already accepted by the preview, or null. */
  discountCode: string | null;
}

/** POST /plus/checkout — a pending invoice plus the gateway redirect (or a settled one at 100% off). */
export async function startCheckout({ planId, discountCode }: CheckoutInput): Promise<PlusCheckout> {
  const { data } = await apiClient.post<ApiEnvelope<unknown>>('/plus/checkout', {
    plan_id: planId,
    ...(discountCode ? { discount_code: discountCode } : {}),
  });
  return checkoutSchema.parse(data.data);
}

/**
 * Sends the browser to the gateway. Only an absolute http(s) URL from the API
 * is followed — anything else is a broken response, not a place to go.
 */
export function goToGateway(url: string): boolean {
  if (!/^https?:\/\//i.test(url)) return false;
  window.location.assign(url);
  return true;
}

function useInvalidatePlus() {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: plusKeys.all });
}

export function useCheckout() {
  const invalidate = useInvalidatePlus();
  return useMutation({
    mutationFn: startCheckout,
    // A 100% discount settles at once; a pending invoice changes nothing yet.
    onSuccess: (result) => (result.payment ? undefined : invalidate()),
  });
}

/** POST /plus/verify — settles the invoice the gateway sent the user back with (idempotent). */
export function useVerifyPayment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (ret: GatewayReturn): Promise<PlusVerification> => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/plus/verify', ret);
      return verificationSchema.parse(data.data);
    },
    onSuccess: (result) => {
      queryClient.setQueryData(plusKeys.status(), result.status);
      void queryClient.invalidateQueries({ queryKey: plusKeys.history() });
    },
  });
}

function useStatusWrite(path: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (): Promise<PlusStatus> => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>(path);
      return statusSchema.parse(data.data);
    },
    onSuccess: (status) => {
      queryClient.setQueryData(plusKeys.status(), status);
      void queryClient.invalidateQueries({ queryKey: plusKeys.all });
    },
  });
}

/** POST /plus/cancel — auto-renew off; Plus stays until the period ends. */
export function useCancelPlus() {
  return useStatusWrite('/plus/cancel');
}

/** POST /plus/trial/start — the one free trial. */
export function useStartTrial() {
  return useStatusWrite('/plus/trial/start');
}

/** POST /plus/restore — re-checks pending checkouts with the gateway. */
export function useRestorePlus() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (): Promise<PlusRestore> => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/plus/restore');
      return restoreSchema.parse(data.data);
    },
    onSuccess: (result) => {
      queryClient.setQueryData(plusKeys.status(), result.status);
      void queryClient.invalidateQueries({ queryKey: plusKeys.all });
    },
  });
}
