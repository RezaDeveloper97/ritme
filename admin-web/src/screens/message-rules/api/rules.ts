'use client';

import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api, pagedSchema, type QueryValue } from '@/shared/api';

/** Smart messages v2 — rule-based messages (admin-api.md §22, B-N9-05, D-86). */

export const LEVELS = ['info', 'suggestion', 'worth_followup', 'follow_soon'] as const;
export const KINDS = ['safety', 'engine', 'clinical', 'content', 'product', 'marketing'] as const;
export const STATUSES = ['draft', 'pending_approval', 'approved'] as const;

const variantSchema = z.object({
  title: z.string().default(''),
  body: z.string().default(''),
  cta_label: z.string().optional().default(''),
  cta_target: z.string().optional().default(''),
});
export type VariantContent = z.infer<typeof variantSchema>;

const conditionSchema = z.object({ fact: z.string(), op: z.string(), value: z.unknown() });
export type Condition = z.infer<typeof conditionSchema>;
const groupSchema = z.object({ op: z.enum(['and', 'or']), items: z.array(conditionSchema).default([]) });
export type Group = z.infer<typeof groupSchema>;

const nullable = <T extends z.ZodTypeAny>(s: T) => s.nullish().transform((v) => v ?? null);
const capSchema = z.object({ count: z.number(), period: z.string() });
export type Cap = z.infer<typeof capSchema>;
const variantWeightSchema = z.object({ key: z.string(), weight: z.number() });
export type VariantWeight = z.infer<typeof variantWeightSchema>;

const statSchema = z.object({
  impressions: z.number(),
  clicks: z.number(),
  dismissals: z.number(),
  goals: z.number(),
  ctr: z.number(),
});
export type Stat = z.infer<typeof statSchema>;

export const ruleSchema = z.object({
  id: z.number(),
  title: z.string(),
  level: z.enum(LEVELS),
  kind: z.enum(KINDS),
  priority: z.number(),
  placement: z.string(),
  content: z.object({
    a: variantSchema,
    b: variantSchema.nullish(),
    variants: z.record(z.string(), variantSchema).nullish().transform((v) => v ?? {}),
    seen: z.string().optional().default(''),
    confidence: z.string().optional().default(''),
    contact: z.string().optional().default(''),
  }),
  variants_enabled: z.boolean(),
  ab_split: z.number(),
  conditions: z.array(groupSchema).nullable().transform((v) => v ?? []),
  exceptions: z.array(conditionSchema).nullable().transform((v) => v ?? []),
  cap_count: z.number().nullable(),
  cap_period: z.string().nullable(),
  cooldown_days: z.number(),
  hours_from: z.number().nullable(),
  hours_to: z.number().nullable(),
  starts_on: z.string().nullable(),
  ends_on: z.string().nullable(),
  audience_percent: z.number(),
  goal: z.string().nullable(),
  // Message model v6 (T-M10-02)
  rule_key: nullable(z.string()),
  variants: nullable(z.array(variantWeightSchema)),
  topics: nullable(z.array(z.string())),
  caps: nullable(z.array(capSchema)),
  hour_windows: nullable(z.array(z.tuple([z.number(), z.number()]))),
  starts_at: nullable(z.string()),
  ends_at: nullable(z.string()),
  goal_event: nullable(z.string()),
  locked: z.boolean().default(false),
  approved_version: nullable(z.number()),
  served_version: nullable(z.number()),
  type: z.string().optional(),
  slot: z.string().optional(),
  name: z.string().optional(),
  level_name: z.string().optional(),
  status: z.enum(STATUSES),
  is_active: z.boolean(),
  version: z.number(),
  needs_medical_approval: z.boolean(),
  approved_by: z.number().nullable(),
  approved_at: z.string().nullable(),
  updated_at: z.string().nullable(),
});
export type Rule = z.infer<typeof ruleSchema>;

const listRuleSchema = ruleSchema.extend({ stats_7d: statSchema });
export type ListRule = z.infer<typeof listRuleSchema>;

const enumsSchema = z.object({
  placements: z.array(z.string()),
  levels: z.array(z.string()),
  kinds: z.array(z.string()),
  statuses: z.array(z.string()),
  variables: z.array(z.string()),
  ops: z.array(z.string()),
  cap_periods: z.array(z.string()),
  bands: z.record(z.string(), z.object({ min: z.number(), max: z.number() })),
  goal_events: z.array(z.string()).default([]),
  topics: z.array(z.string()).default([]),
  level_kinds: z.record(z.string(), z.array(z.string())).default({}),
  slot_kinds: z.record(z.string(), z.array(z.string())).default({}),
  approval_kinds: z.array(z.string()).default(['safety', 'clinical', 'content']),
  max_variants: z.number().default(6),
});
export type RuleEnums = z.infer<typeof enumsSchema>;

const conflictSchema = z.object({
  rule_id: z.number(),
  title: z.string(),
  priority: z.number(),
  kind: z.string(),
  winner: z.enum(['this', 'other', 'tie']),
  safety: z.boolean(),
});
export type Conflict = z.infer<typeof conflictSchema>;

const detailSchema = z.object({
  rule: ruleSchema,
  /** Snapshot of the approved version (for the diff); not served by the API yet — the banner works without it. */
  approved: ruleSchema.partial().nullish(),
  versions: z.array(z.object({ version: z.number(), changed_by: z.number().nullable(), created_at: z.string().nullable() })),
  variant_stats_7d: z.object({ a: statSchema, b: statSchema }),
  conflicts: z.array(conflictSchema),
});

const writeSchema = z.object({ rule: ruleSchema });

const previewSchema = z.object({
  eligible: z.boolean(),
  reasons: z.array(z.string()),
  variant: z.string(),
  title: z.string(),
  body: z.string(),
  cta_label: z.string(),
  level: z.string(),
});
export type Preview = z.infer<typeof previewSchema>;

const keys = {
  all: ['message-rules'] as const,
  list: (q: Record<string, QueryValue>) => ['message-rules', 'list', q] as const,
  one: (id: number) => ['message-rules', 'one', id] as const,
};

export function useRuleList(query: Record<string, QueryValue>) {
  return useQuery({
    queryKey: keys.list(query),
    queryFn: ({ signal }) =>
      api.get('/messages/rules', { query, schema: pagedSchema(listRuleSchema, { enums: enumsSchema }), signal }),
    placeholderData: keepPreviousData,
  });
}

export function useRule(id: number | null) {
  return useQuery({
    queryKey: keys.one(id ?? 0),
    queryFn: ({ signal }) => api.get(`/messages/rules/${id}`, { schema: detailSchema, signal }),
    enabled: id !== null,
  });
}

/** The enum lists (placements, ops, bands…) ride on the list answer; the editor reads them from a 1-row page. */
export function useRuleEnums() {
  return useQuery({
    queryKey: ['message-rules', 'enums'],
    queryFn: ({ signal }) =>
      api.get('/messages/rules', { query: { per_page: 1 }, schema: pagedSchema(listRuleSchema, { enums: enumsSchema }), signal }),
    select: (d) => d.enums,
    staleTime: 5 * 60_000,
  });
}

function useWrite<V>(fn: (v: V) => Promise<unknown>) {
  const client = useQueryClient();
  return useMutation({ mutationFn: fn, onSuccess: () => client.invalidateQueries({ queryKey: keys.all }) });
}

export const useCreateRule = () =>
  useWrite((body: unknown) => api.post('/messages/rules', body, { schema: writeSchema }));
export const useUpdateRule = () =>
  useWrite(({ id, body }: { id: number; body: unknown }) => api.put(`/messages/rules/${id}`, body, { schema: writeSchema }));
export const useDeleteRule = () => useWrite((id: number) => api.delete(`/messages/rules/${id}`));
export const useRuleAction = () =>
  useWrite(({ id, action }: { id: number; action: 'submit' | 'approve' | 'reject' | 'toggle' }) =>
    api.post(`/messages/rules/${id}/${action}`, undefined, { schema: writeSchema }),
  );

export function useConflictCheck() {
  return useMutation({
    mutationFn: (body: { rule: unknown; rule_id?: number }) =>
      api.post('/messages/rules/conflicts', body, { schema: z.object({ conflicts: z.array(conflictSchema) }) }),
  });
}

export function usePreview() {
  return useMutation({
    mutationFn: (body: { rule: unknown; rule_id?: number; user_id?: number; facts?: Record<string, unknown> }) =>
      api.post('/messages/rules/preview', body, { schema: previewSchema }),
  });
}

// ---- B-N9-06: engine view + simulator (pure, no writes) ----

const engineSchema = z.object({
  placements: z.array(
    z.object({
      placement: z.string(),
      urgent: z.boolean(),
      count: z.number(),
      rules: z.array(
        z.object({
          rule_id: z.number(),
          title: z.string(),
          kind: z.string(),
          level: z.string(),
          priority: z.number(),
          served: z.boolean(),
          unlimited: z.boolean(),
          cap: z.object({ count: z.number(), period: z.string() }).nullable(),
          cooldown_days: z.number(),
          audience_percent: z.number(),
          variants: z.boolean(),
          conflicts: z.number(),
        }),
      ),
    }),
  ),
  bands: z.record(z.string(), z.object({ min: z.number(), max: z.number() })),
});
export type EngineView = z.infer<typeof engineSchema>;

const simulationSchema = z.object({
  at: z.string(),
  placements: z.array(
    z.object({
      placement: z.string(),
      winner: z
        .object({
          rule_id: z.number(),
          title: z.string(),
          variant: z.string(),
          level: z.string(),
          kind: z.string(),
          priority: z.number(),
          message_title: z.string(),
          message_body: z.string(),
        })
        .nullable(),
      trace: z.array(
        z.object({
          rule_id: z.number(),
          title: z.string(),
          kind: z.string(),
          level: z.string(),
          priority: z.number(),
          eligible: z.boolean(),
          reason: z.string(),
          failed_groups: z.array(z.number()),
          variant: z.string(),
          winner: z.boolean(),
        }),
      ),
    }),
  ),
});
export type Simulation = z.infer<typeof simulationSchema>;

export function useEngineView() {
  return useQuery({
    queryKey: ['message-rules', 'engine'],
    queryFn: ({ signal }) => api.get('/messages/rules/engine', { schema: engineSchema, signal }),
  });
}

export function useSimulate() {
  return useMutation({
    mutationFn: (body: { facts: Record<string, unknown>; at?: string; user_id?: number }) =>
      api.post('/messages/rules/simulate', body, { schema: simulationSchema }),
  });
}
