'use client';

import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { healthRecordKeys } from './keys';

/*
 * The emergency card (CB-REC-03, Go only). Health data (§11): never log a payload or an error body.
 * - GET /health-record/emergency-card — the owner's full view (CB-REC-05 builds the editing screen on it).
 * - GET /health-record/emergency-card/lock — what the app-lock screen may read (CB-PRIV-01, D-73): `{enabled:false}`
 *   while «نمایش روی صفحه قفل» is off, else the minimal card (first name, no gyn conditions / onboarding meds /
 *   insurance). The lock screen uses only this one.
 */

const strings = z.array(z.string()).catch([]);
const nullableString = z.string().nullable().catch(null);

/** The minimal card fields both views share. */
const cardFields = {
  name: nullableString,
  blood_type: nullableString,
  allergies: z.array(z.string()).nullable().catch(null),
  conditions: z.object({ chronic_illnesses: strings }).passthrough(),
  medications: z.array(z.object({ title: z.string(), dose: nullableString })).catch([]),
  pregnancy: z.object({ week: z.number().int() }).nullable().catch(null),
  emergency_contact: z
    .object({ name: nullableString, relation: nullableString, phone: nullableString })
    .nullable()
    .catch(null),
};

type RawCard = z.infer<z.ZodObject<typeof cardFields>> & {
  insurance?: { label: string | null; masked: string } | null;
};

function toCard(raw: RawCard) {
  return {
    name: raw.name,
    bloodType: raw.blood_type,
    allergies: raw.allergies,
    conditions: raw.conditions.chronic_illnesses,
    medications: raw.medications,
    pregnancyWeek: raw.pregnancy?.week ?? null,
    contact: raw.emergency_contact,
    insurance: raw.insurance ? { label: raw.insurance.label, masked: raw.insurance.masked } : null,
  };
}

export type EmergencyCard = ReturnType<typeof toCard>;

export const emergencyCardSchema = z
  .object({
    card: z.object({
      ...cardFields,
      insurance: z.object({ label: nullableString, masked: z.string() }).passthrough().nullable().catch(null),
    }),
    settings: z.object({ show_on_lock_screen: z.boolean().catch(false) }).passthrough(),
  })
  .transform((raw) => ({ ...toCard(raw.card), showOnLockScreen: raw.settings.show_on_lock_screen }));

export type OwnerEmergencyCard = z.output<typeof emergencyCardSchema>;

/** The lock-screen answer: null = the owner did not put the card on the lock screen. */
export const lockEmergencyCardSchema = z
  .object({ enabled: z.boolean(), card: z.object(cardFields).optional() })
  .transform((raw): EmergencyCard | null => (raw.enabled && raw.card ? { ...toCard(raw.card), insurance: null } : null));

/** The owner's emergency card (full view). Never used by the lock screen. */
export function useEmergencyCard() {
  return useQuery({
    queryKey: healthRecordKeys.emergencyCard(),
    queryFn: async (): Promise<OwnerEmergencyCard> => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/health-record/emergency-card');
      return emergencyCardSchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: 1,
  });
}

/**
 * The lock screen's card (minimal, or null when not shown there). `enabled` lets the lock screen stop asking once
 * the session is ending.
 */
export function useLockEmergencyCard({ enabled = true }: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: healthRecordKeys.emergencyCardLock(),
    queryFn: async (): Promise<EmergencyCard | null> => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/health-record/emergency-card/lock');
      return lockEmergencyCardSchema.parse(data.data);
    },
    enabled: enabled && isAuthenticated(),
    staleTime: 60_000,
    retry: 1,
  });
}
