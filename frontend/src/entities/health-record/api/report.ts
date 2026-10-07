'use client';

import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type {
  Allergies,
  Basics,
  Conditions,
  CycleSummary,
  Medications,
  Pregnancies,
  RecordCheckup,
  RecordLab,
  RecordPerson,
  Section,
  VitalsSummary,
} from '../model/types';
import { healthRecordKeys } from './keys';
import { type MenopauseSection, menopauseSectionSchema } from './menopause-section';
import { rawSection, SECTION_SCHEMAS, type SectionKey } from './schema';

/*
 * The doctor report «گزارش برای پزشک» (bloom B-N6-04, D-65): the record built by the API for the *share* audience
 * (no row ids, no free-text notes) over a chosen window and chosen sections — the preview, the on-device PDF and the
 * 7-day share link all read this shape. Health data (§11): never logged, never in a URL (the selection travels in
 * memory; only the share token is in the public URL).
 */

export const REPORT_RANGES = ['3m', '6m', '1y', 'custom'] as const;
export type ReportRange = (typeof REPORT_RANGES)[number];
export const REPORT_SECTIONS = [
  'basics',
  'conditions',
  'medications',
  'allergies',
  'cycle',
  'vitals',
  'pregnancies',
  'checkups',
  'labs',
] as const satisfies readonly SectionKey[];
export type ReportSection = (typeof REPORT_SECTIONS)[number];
/** Sections other domains plug into the builder (CB-MENO-03 `menopause`); requested on their own screens. */
export const REPORT_PROVIDER_SECTIONS = ['menopause'] as const;
export type ReportProviderSection = (typeof REPORT_PROVIDER_SECTIONS)[number];
export const MAX_REPORT_QUESTION = 300;

/** What the owner picked on «گزارش برای پزشک». `from` (Gregorian `YYYY-MM-DD`) only for `custom`. */
export interface ReportSelection {
  range: ReportRange;
  from: string | null;
  sections: (ReportSection | ReportProviderSection)[];
  question: string;
}

export interface ReportWindow {
  key: string;
  from: string;
  to: string;
  days: number;
}

/** A report: only the chosen sections are present. */
export interface HealthReport {
  range: ReportWindow;
  date: string;
  person: RecordPerson;
  basics?: Section<Basics>;
  conditions?: Section<Conditions>;
  medications?: Section<Medications>;
  allergies?: Section<Allergies>;
  cycle?: Section<CycleSummary>;
  vitals?: Section<VitalsSummary>;
  pregnancies?: Section<Pregnancies>;
  checkups?: Section<{ items: RecordCheckup[] }>;
  labs?: Section<{ items: RecordLab[] }>;
  /** CB-MENO-11: present only when requested, for a user in menopause mode. */
  menopause?: Section<MenopauseSection>;
}

/** The frozen report behind a share link (public view). */
export interface SharedReport extends HealthReport {
  question: string | null;
  createdAt: string;
  expiresAt: string;
}

const windowSchema = z.object({ key: z.string(), from: z.string(), to: z.string(), days: z.number() });

const recordSchema = z.object({
  date: z.string(),
  person: z.object({ name: z.string().nullable(), age: z.number().nullable(), gender: z.string().nullable(), life_mode: z.string() }),
  sections: z.array(rawSection),
});

function toReport(range: z.output<typeof windowSchema>, rec: z.output<typeof recordSchema>): HealthReport {
  const out: HealthReport = {
    range,
    date: rec.date,
    person: { name: rec.person.name, age: rec.person.age, gender: rec.person.gender, lifeMode: rec.person.life_mode },
  };
  const target = out as unknown as Record<string, unknown>;
  for (const s of rec.sections) {
    if (s.key === 'menopause') {
      out.menopause = { editable: false, empty: s.empty, data: menopauseSectionSchema.parse(s.data) };
      continue;
    }
    if (!(REPORT_SECTIONS as readonly string[]).includes(s.key)) continue; // a later provider section: ignored here
    const key = s.key as ReportSection;
    target[key] = { editable: false, empty: s.empty, data: SECTION_SCHEMAS[key].parse(s.data) };
  }
  return out;
}

export const healthReportSchema = z
  .object({ range: windowSchema, record: recordSchema })
  .transform((d): HealthReport => toReport(d.range, d.record));

export const sharedReportSchema = z
  .object({
    range: windowSchema,
    record: recordSchema,
    question: z.string().nullable(),
    created_at: z.string(),
    expires_at: z.string(),
  })
  .transform((d): SharedReport => ({
    ...toReport(d.range, d.record),
    question: d.question,
    createdAt: d.created_at,
    expiresAt: d.expires_at,
  }));

function reportParams(sel: Pick<ReportSelection, 'range' | 'from' | 'sections'>): Record<string, string> {
  const params: Record<string, string> = { range: sel.range, sections: sel.sections.join(',') };
  if (sel.range === 'custom' && sel.from) params.from = sel.from;
  return params;
}

/** GET /health-record/report — the share-audience report for the selection (not the question: it stays on the device). */
export function useHealthReport(sel: Pick<ReportSelection, 'range' | 'from' | 'sections'>, enabled = true) {
  const params = reportParams(sel);
  return useQuery({
    queryKey: healthRecordKeys.report(params),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/health-record/report', { params });
      return healthReportSchema.parse(data.data);
    },
    enabled: enabled && isAuthenticated() && sel.sections.length > 0 && (sel.range !== 'custom' || !!sel.from),
    placeholderData: keepPreviousData,
    staleTime: 30_000,
    retry: 1,
  });
}

/** GET /shared-reports/{token} — public, no session needed. 404 unknown, 410 expired / revoked. */
export async function fetchSharedReport(token: string): Promise<SharedReport> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/shared-reports/${encodeURIComponent(token)}`);
  return sharedReportSchema.parse(data.data);
}

export function useSharedReport(token: string) {
  return useQuery({
    queryKey: ['shared-report', token] as const,
    queryFn: () => fetchSharedReport(token),
    retry: false,
    staleTime: Infinity,
    gcTime: 0,
  });
}

/* ── Share links (owner) ─────────────────────────────────────────────────────────────────────────────────────────── */

export type ShareLinkStatus = 'active' | 'expired' | 'revoked';

export interface ShareLink {
  id: number;
  status: ShareLinkStatus;
  sections: string[];
  from: string;
  to: string;
  createdAt: string;
  expiresAt: string;
  revokedAt: string | null;
  views: number;
}

export interface ShareLinks {
  items: ShareLink[];
  activeCount: number;
  maxActive: number;
  ttlDays: number;
}

const shareLinkSchema = z
  .object({
    id: z.number(),
    status: z.enum(['active', 'expired', 'revoked']),
    sections: z.array(z.string()),
    range: z.object({ from: z.string(), to: z.string() }),
    created_at: z.string(),
    expires_at: z.string(),
    revoked_at: z.string().nullable(),
    views: z.number(),
  })
  .transform(
    (l): ShareLink => ({
      id: l.id,
      status: l.status,
      sections: l.sections,
      from: l.range.from,
      to: l.range.to,
      createdAt: l.created_at,
      expiresAt: l.expires_at,
      revokedAt: l.revoked_at,
      views: l.views,
    }),
  );

export const shareLinksSchema = z
  .object({ items: z.array(shareLinkSchema), active_count: z.number(), max_active: z.number(), ttl_days: z.number() })
  .transform((d): ShareLinks => ({ items: d.items, activeCount: d.active_count, maxActive: d.max_active, ttlDays: d.ttl_days }));

export const createdShareLinkSchema = z
  .object({ token: z.string().regex(/^[A-Za-z0-9_-]{43}$/) })
  .and(z.object({ id: z.number(), expires_at: z.string() }))
  .transform((d) => ({ id: d.id, token: d.token, expiresAt: d.expires_at }));

export type CreatedShareLink = z.output<typeof createdShareLinkSchema>;

/** GET /health-record/share-links — the owner's links of the last 30 days (metadata only). */
export function useShareLinks(enabled = true) {
  return useQuery({
    queryKey: healthRecordKeys.shareLinks(),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/health-record/share-links');
      return shareLinksSchema.parse(data.data);
    },
    enabled: enabled && isAuthenticated(),
    staleTime: 15_000,
    retry: 1,
  });
}

/** POST /health-record/share-links — Plus (402 → paywall); the token comes back once. */
export function useCreateShareLink() {
  const queryClient = useQueryClient();
  return useMutation<CreatedShareLink, unknown, ReportSelection>({
    mutationFn: async (sel) => {
      const body: Record<string, unknown> = { range: sel.range, sections: sel.sections };
      if (sel.range === 'custom') body.from = sel.from;
      const q = sel.question.trim();
      if (q) body.question = q;
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/health-record/share-links', body);
      return createdShareLinkSchema.parse(data.data);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: healthRecordKeys.shareLinks() }),
  });
}

/** DELETE /health-record/share-links/{id} — revoke now. */
export function useRevokeShareLink() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, number>({
    mutationFn: async (id) => {
      await apiClient.delete(`/health-record/share-links/${id}`);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: healthRecordKeys.shareLinks() }),
  });
}
