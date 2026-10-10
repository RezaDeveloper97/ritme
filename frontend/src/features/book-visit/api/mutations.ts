'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import {
  bookResultSchema,
  bookingKeys,
  bookingOneSchema,
  messageSchema,
  prescriptionItemSchema,
  verifySchema,
  type ConsentScope,
  type ForWhom,
  type VisitMode,
} from '@/entities/doctor';
import { type ApiEnvelope, apiClient } from '@/shared/api';

export interface BookInput {
  doctorId: number;
  mode: VisitMode;
  /** `Y-m-d H:i` wall clock of the slot (Tehran). */
  startsAt: string;
  forWhom: ForWhom;
  childId?: number;
  patientName?: string;
  reason?: string;
  note?: string;
  /** Only for «خودم» visits — the server rejects sharing for child / other (QUESTIONS #125). */
  share?: ConsentScope[];
}

/** Only an absolute http(s) URL from the API is followed to the gateway. */
export function goToGateway(url: string): boolean {
  if (!/^https?:\/\//i.test(url)) return false;
  window.location.assign(url);
  return true;
}

function useInvalidate() {
  const client = useQueryClient();
  return () => client.invalidateQueries({ queryKey: bookingKeys.all });
}

/** POST /telemed/bookings — holds the slot (10 min) and starts the payment; `payment` null = confirmed already. */
export function useCreateBooking() {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: async (i: BookInput) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/telemed/bookings', {
        doctor_id: i.doctorId,
        mode: i.mode,
        starts_at: i.startsAt,
        for_whom: i.forWhom,
        child_id: i.forWhom === 'child' ? i.childId : undefined,
        patient_name: i.forWhom === 'other' ? i.patientName : undefined,
        reason: i.reason || undefined,
        note: i.note?.trim() || undefined,
        share: i.forWhom === 'self' && i.share?.length ? i.share : undefined,
      });
      return bookResultSchema.parse(data.data);
    },
    onSuccess: () => void invalidate(),
  });
}

/** POST /telemed/bookings/verify — settles the gateway return server to server (idempotent). */
export function useVerifyBooking() {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: async (r: { reference: string; authority: string }) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/telemed/bookings/verify', r);
      return verifySchema.parse(data.data);
    },
    onSuccess: () => void invalidate(),
  });
}

/** POST /telemed/bookings/{id}/cancel — free until 2 h before, with a refund. */
export function useCancelBooking(id: number) {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: async () => bookingOneSchema.parse((await apiClient.post<ApiEnvelope<unknown>>(`/telemed/bookings/${id}/cancel`, {})).data.data),
    onSuccess: () => void invalidate(),
  });
}

/** POST /telemed/bookings/{id}/reschedule `{starts_at}`. */
export function useRescheduleBooking(id: number) {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: async (startsAt: string) =>
      bookingOneSchema.parse((await apiClient.post<ApiEnvelope<unknown>>(`/telemed/bookings/${id}/reschedule`, { starts_at: startsAt })).data.data),
    onSuccess: () => void invalidate(),
  });
}

/** PUT /telemed/bookings/{id}/consent — only the scopes sent change. */
export function useUpdateConsent(id: number) {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: async (scopes: Partial<Record<ConsentScope, boolean>>) =>
      bookingOneSchema.parse((await apiClient.put<ApiEnvelope<unknown>>(`/telemed/bookings/${id}/consent`, scopes)).data.data),
    onSuccess: () => void invalidate(),
  });
}

/** POST /telemed/bookings/{id}/messages `{body}` — appended to the thread cache by the next poll. */
export function useSendMessage(id: number) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (body: string) => {
      const { data } = await apiClient.post<ApiEnvelope<{ message: unknown }>>(`/telemed/bookings/${id}/messages`, { body });
      return messageSchema.parse(data.data?.message);
    },
    onSuccess: () => void client.invalidateQueries({ queryKey: bookingKeys.messages(id) }),
  });
}

/** POST …/prescription/{item}/reminder — «افزودن به یادآور» creates the medication reminder. */
export function useAddPrescriptionReminder(id: number) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (itemId: number) => {
      const { data } = await apiClient.post<ApiEnvelope<{ item: unknown }>>(`/telemed/bookings/${id}/prescription/${itemId}/reminder`, {});
      return prescriptionItemSchema.parse(data.data?.item);
    },
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: bookingKeys.prescription(id) });
      void client.invalidateQueries({ queryKey: ['care'] });
    },
  });
}
