'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
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
      insurance: z
        .object({ label: nullableString, masked: z.string(), last4: z.string().catch('') })
        .passthrough()
        .nullable()
        .catch(null),
    }),
    settings: z
      .object({
        show_on_lock_screen: z.boolean().catch(false),
        // CB-REC-05: the owner screen's switches + the allergy flag (edited through PUT /health-record/extras).
        show_pregnancy: z.boolean().catch(false),
        allergies_on_card: z.boolean().catch(false),
      })
      .passthrough(),
    public_link: z
      .object({
        enabled: z.boolean().catch(false),
        enabled_at: nullableString,
        views: z.number().catch(0),
        last_viewed_at: nullableString,
      })
      .catch({ enabled: false, enabled_at: null, views: 0, last_viewed_at: null }),
  })
  .transform((raw) => ({
    ...toCard(raw.card),
    showOnLockScreen: raw.settings.show_on_lock_screen,
    showPregnancy: raw.settings.show_pregnancy,
    allergiesOnCard: raw.settings.allergies_on_card,
    insuranceLast4: raw.card.insurance?.last4 ?? null,
    publicLink: {
      enabled: raw.public_link.enabled,
      enabledAt: raw.public_link.enabled_at,
      views: raw.public_link.views,
      lastViewedAt: raw.public_link.last_viewed_at,
    },
  }));

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

/* ── CB-REC-05: the owner's edit screen + the public card ─────────────────────────────────────────────────────────── */

/** PUT /health-record/emergency-card — partial: an absent key keeps its part, `null` clears contact / insurance. */
export interface EmergencyCardInput {
  show_on_lock_screen?: boolean;
  show_pregnancy?: boolean;
  emergency_contact?: { name: string; relation: string | null; phone: string } | null;
  insurance?: { label: string | null; last4: string } | null;
}

export function useSaveEmergencyCard() {
  const queryClient = useQueryClient();
  return useMutation<OwnerEmergencyCard, unknown, EmergencyCardInput>({
    mutationFn: async (input) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>('/health-record/emergency-card', input);
      return emergencyCardSchema.parse(data.data);
    },
    onSuccess: (card) => {
      queryClient.setQueryData(healthRecordKeys.emergencyCard(), card);
      void queryClient.invalidateQueries({ queryKey: healthRecordKeys.emergencyCardLock() });
    },
  });
}

/** The public link's answer to POST …/public-link: the token and path come back once (kept in mutation memory only). */
export const createdEmergencyLinkSchema = z
  .object({
    token: z.string().regex(/^[A-Za-z0-9_-]{43}$/),
    path: z.string().regex(/^\/[a-z]{2,3}\/emergency\/[A-Za-z0-9_-]{43}$/),
  })
  .passthrough()
  .transform((raw) => ({ token: raw.token, path: raw.path, card: emergencyCardSchema.parse(raw) }));

export type CreatedEmergencyLink = z.output<typeof createdEmergencyLinkSchema>;

/** POST /health-record/emergency-card/public-link (201) — a fresh link; the previous one stops working. */
export function useEnableEmergencyLink() {
  const queryClient = useQueryClient();
  return useMutation<CreatedEmergencyLink, unknown, void>({
    mutationFn: async () => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/health-record/emergency-card/public-link');
      return createdEmergencyLinkSchema.parse(data.data);
    },
    onSuccess: (created) => queryClient.setQueryData(healthRecordKeys.emergencyCard(), created.card),
  });
}

/** DELETE /health-record/emergency-card/public-link — the link answers 404 from now on. */
export function useDisableEmergencyLink() {
  const queryClient = useQueryClient();
  return useMutation<OwnerEmergencyCard, unknown, void>({
    mutationFn: async () => {
      const { data } = await apiClient.delete<ApiEnvelope<unknown>>('/health-record/emergency-card/public-link');
      return emergencyCardSchema.parse(data.data);
    },
    onSuccess: (card) => queryClient.setQueryData(healthRecordKeys.emergencyCard(), card),
  });
}

export const publicEmergencyCardSchema = z
  .object(cardFields)
  .transform((raw): EmergencyCard => ({ ...toCard(raw), insurance: null }));

/** GET /emergency-cards/{token} — public, minimal card; 404 unknown / turned off. Memory only (`gcTime: 0`). */
export function usePublicEmergencyCard(token: string, enabled = true) {
  return useQuery({
    enabled,
    queryKey: ['public-emergency-card', token] as const,
    queryFn: async (): Promise<EmergencyCard> => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/emergency-cards/${encodeURIComponent(token)}`);
      return publicEmergencyCardSchema.parse(data.data);
    },
    retry: false,
    staleTime: Infinity,
    gcTime: 0,
  });
}
