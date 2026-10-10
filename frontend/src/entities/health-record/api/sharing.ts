'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { healthRecordKeys } from './keys';
import { type ReportSection, sharedReportSchema, type SharedReport } from './report';

/*
 * «چه کسی پرونده‌ات را می‌بیند؟» (CB-REC-03 API, D-71; CB-REC-05 screen): 24h doctor codes + QR, the companions'
 * record scope (meds & appointments only, never documents) and the access history (doctor-code / share-link opens
 * only — DECISIONS #12: nothing is ever sent to an insurer, so there are no insurer rows).
 * Security: the code, the token and the QR URL come back once and live only in react-query memory (`gcTime: 0` on the
 * public reads, mutation data on the owner side) — never persisted, never logged, never in a URL of this app's own.
 */

export type ShareCodeStatus = 'active' | 'expired' | 'revoked';

export interface ShareCode {
  id: number;
  label: string | null;
  status: ShareCodeStatus;
  sections: string[];
  from: string;
  to: string;
  createdAt: string;
  expiresAt: string;
  revokedAt: string | null;
  views: number;
  lastViewedAt: string | null;
}

export type FamilyLevel = 'none' | 'view' | 'edit';

export interface FamilyAccess {
  companionId: number;
  type: 'partner' | 'spouse' | 'parent';
  status: 'invited' | 'active';
  name: string | null;
  meds: FamilyLevel;
  appointments: FamilyLevel;
}

export interface ShareAccessEntry {
  id: number;
  linkId: number;
  kind: 'report' | 'summary';
  label: string | null;
  via: 'link' | 'code';
  device: 'mobile' | 'tablet' | 'desktop' | 'unknown';
  browser: string;
  viewedAt: string;
  linkStatus: ShareCodeStatus;
}

export interface RecordSharing {
  doctor: { available: boolean; ttlHours: number; activeCount: number; maxActive: number; items: ShareCode[] };
  family: FamilyAccess[];
  accessLog: ShareAccessEntry[];
}

const statusSchema = z.enum(['active', 'expired', 'revoked']);
const levelSchema = z.enum(['none', 'view', 'edit']).catch('none');

const shareCodeSchema = z
  .object({
    id: z.number(),
    label: z.string().nullable().catch(null),
    status: statusSchema,
    sections: z.array(z.string()).catch([]),
    range: z.object({ from: z.string(), to: z.string() }),
    created_at: z.string(),
    expires_at: z.string(),
    revoked_at: z.string().nullable().catch(null),
    views: z.number().catch(0),
    last_viewed_at: z.string().nullable().catch(null),
  })
  .transform(
    (c): ShareCode => ({
      id: c.id,
      label: c.label,
      status: c.status,
      sections: c.sections,
      from: c.range.from,
      to: c.range.to,
      createdAt: c.created_at,
      expiresAt: c.expires_at,
      revokedAt: c.revoked_at,
      views: c.views,
      lastViewedAt: c.last_viewed_at,
    }),
  );

const familySchema = z
  .object({
    companion_id: z.number(),
    type: z.enum(['partner', 'spouse', 'parent']),
    status: z.enum(['invited', 'active']),
    name: z.string().nullable().catch(null),
    meds: levelSchema,
    appointments: levelSchema,
  })
  .transform(
    (f): FamilyAccess => ({
      companionId: f.companion_id,
      type: f.type,
      status: f.status,
      name: f.name,
      meds: f.meds,
      appointments: f.appointments,
    }),
  );

const accessSchema = z
  .object({
    id: z.number(),
    link_id: z.number(),
    kind: z.enum(['report', 'summary']),
    label: z.string().nullable().catch(null),
    via: z.enum(['link', 'code']),
    client: z.object({
      device: z.enum(['mobile', 'tablet', 'desktop', 'unknown']).catch('unknown'),
      browser: z.string().catch('other'),
    }),
    viewed_at: z.string(),
    link_status: statusSchema,
  })
  .transform(
    (a): ShareAccessEntry => ({
      id: a.id,
      linkId: a.link_id,
      kind: a.kind,
      label: a.label,
      via: a.via,
      device: a.client.device,
      browser: a.client.browser,
      viewedAt: a.viewed_at,
      linkStatus: a.link_status,
    }),
  );

export const recordSharingSchema = z
  .object({
    doctor: z.object({
      available: z.boolean(),
      ttl_hours: z.number(),
      active_count: z.number(),
      max_active: z.number(),
      items: z.array(shareCodeSchema),
    }),
    family: z.array(familySchema).catch([]),
    access_log: z.array(accessSchema).catch([]),
  })
  .transform(
    (d): RecordSharing => ({
      doctor: {
        available: d.doctor.available,
        ttlHours: d.doctor.ttl_hours,
        activeCount: d.doctor.active_count,
        maxActive: d.doctor.max_active,
        items: d.doctor.items,
      },
      family: d.family,
      accessLog: d.access_log,
    }),
  );

/** What the owner picks for a doctor code: a window, the sections and up to 10 of her own documents. */
export interface ShareCodeInput {
  range: '3m' | '6m' | '1y';
  sections: ReportSection[];
  documentIds: number[];
}

export const MAX_SHARE_DOCUMENTS = 10;

export const createdShareCodeSchema = z
  .object({
    code: z.string(),
    token: z.string().regex(/^[A-Za-z0-9_-]{43}$/),
    qr_path: z.string().regex(/^\/[a-z]{2,3}\/shared\/report\/[A-Za-z0-9_-]{43}$/),
    qr_url: z.string().nullable().catch(null),
    documents_count: z.number().catch(0),
  })
  .passthrough()
  .transform((created) => ({
    code: created.code,
    token: created.token,
    qrPath: created.qr_path,
    qrUrl: created.qr_url,
    documentsCount: created.documents_count,
    meta: shareCodeSchema.parse(created),
  }));

export type CreatedShareCode = z.output<typeof createdShareCodeSchema>;

/** GET /health-record/sharing — doctor codes (metadata), family scope, the 10 latest opens. */
export function useRecordSharing() {
  return useQuery({
    queryKey: healthRecordKeys.sharing(),
    queryFn: async (): Promise<RecordSharing> => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/health-record/sharing');
      return recordSharingSchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 15_000,
    retry: 1,
  });
}

/** POST /health-record/share-codes (201) — the code + QR come back once. 409 = 5 live codes, 503 = unavailable. */
export function useCreateShareCode() {
  const queryClient = useQueryClient();
  return useMutation<CreatedShareCode, unknown, ShareCodeInput>({
    mutationFn: async (input) => {
      const body: Record<string, unknown> = { range: input.range, sections: input.sections };
      if (input.documentIds.length) body.document_ids = input.documentIds.slice(0, MAX_SHARE_DOCUMENTS);
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/health-record/share-codes', body);
      return createdShareCodeSchema.parse(data.data);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: healthRecordKeys.sharing() }),
  });
}

/** DELETE /health-record/share-codes/{id} — the code and its QR answer 404 from now on. */
export function useRevokeShareCode() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, number>({
    mutationFn: async (id) => {
      await apiClient.delete(`/health-record/share-codes/${id}`);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: healthRecordKeys.sharing() }),
  });
}

/**
 * POST /shared-reports/code — public: the doctor types the code. Every refusal is the same 404; 429 while throttled.
 * A mutation (not a query) so the code and the report stay in this component's memory only.
 */
export function useOpenSharedReportByCode() {
  return useMutation<SharedReport, unknown, string>({
    mutationFn: async (code) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/shared-reports/code', { code });
      return sharedReportSchema.parse(data.data);
    },
    retry: false,
  });
}
