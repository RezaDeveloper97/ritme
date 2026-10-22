'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api } from '@/shared/api';

/** Versioned clinical thresholds with two medical approvals (admin-api.md §23, B-N9-11). */

export const CONFIG_GROUPS = ['engine', 'vitals', 'pregnancy', 'lab', 'child', 'screening'] as const;
export type ConfigGroup = (typeof CONFIG_GROUPS)[number];

const personSchema = z.object({ id: z.number(), name: z.string().nullable() });

/** One cell of a rows key: a number, a string / enum value, or null (open bound, "any"). */
const cellSchema = z.union([z.number(), z.string(), z.null()]);
export type Cell = z.infer<typeof cellSchema>;
export type ConfigRow = Record<string, Cell>;

/** A rows key's value {rows:[…]} (T-M10-16), else named numbers. */
const rowsValueSchema = z.object({ rows: z.array(z.record(z.string(), cellSchema)) });
export type RowsValue = z.infer<typeof rowsValueSchema>;
const numbersValueSchema = z.record(z.string(), z.number());
export type NumbersValue = z.infer<typeof numbersValueSchema>;

const versionSchema = z.object({
  id: z.number().nullable(),
  version: z.number(),
  value: z.union([rowsValueSchema, numbersValueSchema]),
  source: z.string(),
  note: z.string(),
  effective_from: z.string().nullable(),
  status: z.enum(['pending', 'approved', 'rejected']),
  proposed_by: personSchema.nullable(),
  approvals: z.array(personSchema.extend({ at: z.string().nullable() })),
  required_approvals: z.number(),
  decided_at: z.string().nullable(),
  created_at: z.string().nullable(),
});
export type ConfigVersion = z.infer<typeof versionSchema>;

const fieldSchema = z.object({ name: z.string(), default: z.number(), min: z.number(), max: z.number() });
export type ConfigField = z.infer<typeof fieldSchema>;

const columnSchema = z.object({
  name: z.string(),
  type: z.enum(['number', 'string', 'enum']),
  nullable: z.boolean(),
  min: z.number().optional(),
  max: z.number().optional(),
  integer: z.boolean().optional(),
  enum: z.array(z.string()).optional(),
  max_length: z.number().optional(),
});
export type ConfigColumn = z.infer<typeof columnSchema>;

const tableSchema = z.object({
  columns: z.array(columnSchema),
  min_rows: z.number(),
  max_rows: z.number(),
  sort_by: z.object({ column: z.string(), strict: z.boolean() }).nullable(),
  range: z
    .object({
      from: z.string(),
      to: z.string(),
      to_inclusive: z.boolean(),
      contiguous: z.boolean(),
      group_by: z.array(z.string()),
    })
    .nullable(),
  unique: z.array(z.string()),
  row_order: z.array(z.tuple([z.string(), z.string()])),
});
export type ConfigTable = z.infer<typeof tableSchema>;

const keySchema = z.object({
  key: z.string(),
  group: z.enum(CONFIG_GROUPS),
  kind: z.enum(['numbers', 'rows']),
  unit: z.string(),
  fields: z.array(fieldSchema),
  schema: tableSchema.nullable(),
  current: versionSchema,
  upcoming: versionSchema.nullable(),
  pending: versionSchema.nullable(),
});
export type ConfigKey = z.infer<typeof keySchema>;

const listSchema = z.object({
  groups: z.array(z.string()),
  keys: z.array(keySchema),
  can_approve: z.boolean(),
  required_approvals: z.number(),
  today: z.string(),
});
export type ConfigList = z.infer<typeof listSchema>;

const LIST_KEY = ['clinical-config'] as const;

export function useClinicalConfig() {
  return useQuery({
    queryKey: LIST_KEY,
    queryFn: ({ signal }) => api.get('/clinical-config', { schema: listSchema, signal }),
  });
}

/** The rows of a version of a rows key ([] for a numbers value). */
export function rowsOf(value: ConfigVersion['value']): ConfigRow[] {
  return 'rows' in value && Array.isArray(value.rows) ? value.rows : [];
}

/** The named numbers of a version of a numbers key ({} for a rows value). */
export function numbersOf(value: ConfigVersion['value']): NumbersValue {
  return 'rows' in value && Array.isArray(value.rows) ? {} : (value as NumbersValue);
}

export interface ProposalInput {
  key: string;
  value: NumbersValue | RowsValue;
  effective_from: string;
  source: string;
  note: string;
}

export function usePropose() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ key, ...body }: ProposalInput) => api.post(`/clinical-config/${encodeURIComponent(key)}/proposals`, body),
    onSuccess: () => client.invalidateQueries({ queryKey: LIST_KEY }),
  });
}

export function useDecide() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ id, action }: { id: number; action: 'approve' | 'reject' }) =>
      api.post(`/clinical-config/proposals/${id}/${action}`),
    onSuccess: () => client.invalidateQueries({ queryKey: LIST_KEY }),
  });
}
