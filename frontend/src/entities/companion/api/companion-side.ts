'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient, getApiErrorStatus } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import {
  COMPANION_PHASES,
  type CompanionArticle,
  type CompanionHome,
  type CompanionPartner,
  type SharedAppointment,
  type SharedMedication,
  type ViewerLink,
} from '../model/companion-side';
import { ACCESS_LEVELS, COMPANION_SECTIONS, type CompanionGrants, type CompanionSection } from '../model/types';

/*
 * The companion's routes (B-N4-02 accept / links / leave, B-N4-03 home), Go
 * only. The invite code goes only into the accept body — never logged,
 * persisted or put in a URL. Section views are minimised server-side and
 * validated leniently here: a missing or malformed section reads as "not
 * shared" (null), the most private reading.
 */

export const companionSideKeys = {
  all: ['companion-side'] as const,
  home: () => [...companionSideKeys.all, 'home'] as const,
  links: () => [...companionSideKeys.all, 'links'] as const,
};

const level = z.enum(ACCESS_LEVELS).catch('none');
const grantsSchema = z.record(z.string(), z.unknown()).transform((raw): CompanionGrants => {
  const out = {} as CompanionGrants;
  for (const s of COMPANION_SECTIONS) out[s] = level.parse(raw[s]);
  return out;
});
const num = z.number().nullable().catch(null);
const str = z.string().nullable().catch(null);

export const viewerLinkSchema = z
  .object({
    id: z.number(),
    type: z.enum(['partner', 'spouse']).catch('partner'),
    owner: z.object({ id: z.number(), name: str }),
    accepted_at: str.optional(),
    grants: grantsSchema,
    can_record_for: z.array(z.string()).nullable().catch([]).optional(),
    family: z
      .object({ id: z.number(), shared_child_ids: z.array(z.number()).nullable().catch([]).optional() })
      .nullable()
      .catch(null)
      .optional(),
  })
  .transform(
    (l): ViewerLink => ({
      id: l.id,
      type: l.type,
      owner: { id: l.owner.id, name: l.owner.name?.trim() || null },
      acceptedAt: l.accepted_at ?? null,
      grants: l.grants,
      canRecordFor: (l.can_record_for ?? []).filter((s): s is CompanionSection =>
        (COMPANION_SECTIONS as readonly string[]).includes(s),
      ),
      family: l.family ? { id: l.family.id, sharedChildIds: l.family.shared_child_ids ?? [] } : null,
    }),
  );

const cycleSchema = z
  .object({
    date: z.string(),
    has_data: z.boolean().catch(false),
    cycle_day: num.optional(),
    cycle_length: num.optional(),
    main_phase: str.optional(),
    days_to_period: num.optional(),
    days_late: z.number().catch(0).optional(),
    predicted_next_period_start: str.optional(),
  })
  .transform((c) => ({
    date: c.date,
    hasData: c.has_data,
    cycleDay: c.cycle_day ?? null,
    cycleLength: c.cycle_length ?? null,
    mainPhase: c.main_phase ?? null,
    daysToPeriod: c.days_to_period ?? null,
    daysLate: c.days_late ?? 0,
    predictedNextPeriodStart: c.predicted_next_period_start ?? null,
  }));

const pregnancySchema = z
  .object({ is_active: z.boolean().catch(false), current_week: num.optional(), trimester: num.optional() })
  .transform((p) => ({ isActive: p.is_active, currentWeek: p.current_week ?? null, trimester: p.trimester ?? null }));

const medSchema = z
  .object({ id: z.number(), title: z.string().catch(''), times: z.array(z.string()).nullable().catch([]).optional() })
  .transform((m): SharedMedication => ({ id: m.id, title: m.title, times: m.times ?? [] }));

const appointmentSchema = z
  .object({ id: z.number(), title: z.string().catch(''), scheduled_at: str.optional(), days_until: num.optional() })
  .transform(
    (a): SharedAppointment => ({
      id: a.id,
      title: a.title,
      scheduledAt: a.scheduled_at ?? null,
      daysUntil: a.days_until ?? null,
    }),
  );

const symptomsSchema = z
  .object({ days: z.array(z.unknown()).catch([]) })
  .transform((s) => s.days.length);

/** A section view: null when not shared (or unreadable). */
function section<T extends z.ZodTypeAny>(schema: T) {
  return schema.nullable().catch(null).optional();
}

const partnerSchema = z
  .object({
    link: viewerLinkSchema,
    partner_name: str.optional(),
    cycle: section(cycleSchema),
    pregnancy: section(pregnancySchema),
    symptoms: section(symptomsSchema),
    meds: section(z.array(medSchema)),
    appointments: section(z.array(appointmentSchema)),
    phase: z.enum(COMPANION_PHASES).catch('general'),
    note: str.optional(),
    tips: z
      .array(z.object({ key: z.string(), title: z.string(), body: str.optional() }))
      .catch([]),
    child: z.unknown().optional(),
  })
  .transform(
    (p): CompanionPartner => ({
      link: p.link,
      partnerName: p.partner_name?.trim() || p.link.owner.name,
      cycle: p.cycle ?? null,
      pregnancy: p.pregnancy ?? null,
      symptomDays: p.symptoms ?? null,
      meds: p.meds ?? null,
      appointments: p.appointments ?? null,
      phase: p.phase,
      note: p.note ?? null,
      tips: p.tips.map((t) => ({ key: t.key, title: t.title, body: t.body ?? null })),
      child: p.child ?? null,
    }),
  );

const articleSchema = z
  .object({
    id: z.number(),
    slug: z.string(),
    title: str.optional(),
    read_time_minutes: num.optional(),
    image_url: str.optional(),
  })
  .transform(
    (a): CompanionArticle => ({
      id: a.id,
      slug: a.slug,
      title: a.title ?? null,
      readTimeMinutes: a.read_time_minutes ?? null,
      imageUrl: a.image_url ?? null,
    }),
  );

export const companionHomeSchema = z
  .object({
    viewer: z.object({ id: z.number(), name: str.optional() }),
    has_partners: z.boolean().catch(false),
    partners: z.array(partnerSchema).catch([]),
    empty_state: z
      .object({ title: z.string(), body: z.string(), action_label: z.string() })
      .nullable()
      .catch(null)
      .optional(),
    articles: z.array(articleSchema).catch([]),
  })
  .transform(
    (h): CompanionHome => ({
      viewer: { id: h.viewer.id, name: h.viewer.name?.trim() || null },
      hasPartners: h.has_partners && h.partners.length > 0,
      partners: h.partners,
      emptyState: h.empty_state
        ? { title: h.empty_state.title, body: h.empty_state.body, actionLabel: h.empty_state.action_label }
        : null,
      articles: h.articles,
    }),
  );

/** GET /companion/home — 403 `not_companion_account` for a woman's account. */
export async function fetchCompanionHome(): Promise<CompanionHome> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/companion/home');
  return companionHomeSchema.parse(data.data);
}

/** GET /companions/links — the owners who made this account their companion. */
export async function fetchCompanionLinks(): Promise<ViewerLink[]> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/companions/links');
  return z.array(viewerLinkSchema).parse(data.data ?? []);
}

export function useCompanionHome(enabled = true) {
  return useQuery({
    queryKey: companionSideKeys.home(),
    queryFn: fetchCompanionHome,
    enabled: enabled && isAuthenticated(),
    staleTime: 60_000,
    // 403 (not a companion account) is an answer, not a blip.
    retry: (count, error) => getApiErrorStatus(error) !== 403 && count < 2,
  });
}

export function useCompanionLinks(enabled = true) {
  return useQuery({
    queryKey: companionSideKeys.links(),
    queryFn: fetchCompanionLinks,
    enabled: enabled && isAuthenticated(),
    staleTime: 30_000,
  });
}

/**
 * POST /companions/accept {code} — one-time; a wrong, used or expired code is a
 * uniform 422 `invite_invalid`, throttled (429) after a few tries.
 */
export function useAcceptCompanion() {
  const queryClient = useQueryClient();
  return useMutation<ViewerLink, unknown, string>({
    mutationFn: async (code) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/companions/accept', { code });
      return viewerLinkSchema.parse(data.data);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: companionSideKeys.all });
    },
  });
}

/** DELETE /companions/links/{id} — the companion leaves; access ends at once. */
export function useLeaveCompanionLink() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, number>({
    mutationFn: async (id) => {
      await apiClient.delete(`/companions/links/${id}`);
    },
    onSuccess: (_, id) => {
      queryClient.setQueryData<ViewerLink[]>(companionSideKeys.links(), (rows) => rows?.filter((l) => l.id !== id));
      void queryClient.invalidateQueries({ queryKey: companionSideKeys.all });
    },
  });
}

export type AcceptErrorKind = 'invalid' | 'throttled' | 'generic';

/** Which message a failed accept shows: wrong/used/expired (422), too many tries (429), or anything else. */
export function acceptErrorKind(error: unknown): AcceptErrorKind {
  const status = getApiErrorStatus(error);
  if (status === 422 || status === 404) return 'invalid';
  if (status === 429) return 'throttled';
  return 'generic';
}
