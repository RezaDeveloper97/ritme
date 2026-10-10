'use client';

import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { adminRoleSchema, permissionsSchema } from '@/entities/admin';
import { api, ApiError, buildUrl, pagedSchema, type QueryValue } from '@/shared/api';
import { ADMIN_API_BASE_URL } from '@/shared/config';

/** Roles & permissions, audit log, temporary health-data access (admin-api.md §21, B-N9-02). */

export const HEALTH_ACCESS_MINUTES = 30;

const capabilitySchema = z.object({ key: z.string(), levels: z.array(z.enum(['view', 'manage'])) });
export type Capability = z.infer<typeof capabilitySchema>;

const roleSchema = z.object({
  key: adminRoleSchema,
  editable: z.boolean(),
  customized: z.boolean(),
  admins: z.number().optional(),
  permissions: permissionsSchema,
});
export type RoleRow = z.infer<typeof roleSchema>;

const rolesSchema = z.object({ capabilities: z.array(capabilitySchema), roles: z.array(roleSchema) });

const auditRowSchema = z.object({
  id: z.number(),
  admin: z.object({ id: z.number(), name: z.string().nullable(), role: z.string().nullable() }).nullable(),
  action: z.string(),
  target_type: z.string(),
  target_id: z.number(),
  ip: z.string(),
  detail: z.record(z.string(), z.unknown()).nullable(),
  created_at: z.string().nullable(),
});
export type AuditRow = z.infer<typeof auditRowSchema>;

export const ACCESS_STATUSES = ['pending', 'approved', 'rejected', 'revoked', 'expired'] as const;
const accessSchema = z.object({
  id: z.number(),
  admin: z.object({ id: z.number(), name: z.string().nullable() }),
  user_id: z.number(),
  reason: z.string(),
  ticket: z.string(),
  status: z.enum(ACCESS_STATUSES),
  decided_by: z.object({ id: z.number(), name: z.string().nullable() }).nullable(),
  decided_at: z.string().nullable(),
  expires_at: z.string().nullable(),
  created_at: z.string().nullable(),
});
export type AccessRequest = z.infer<typeof accessSchema>;

const healthSchema = z.object({
  user_id: z.number(),
  expires_at: z.string().nullable(),
  profile: z
    .object({
      birthday: z.string().nullable(),
      last_period_start: z.string().nullable(),
      cycle_duration: z.number().nullable(),
      period_duration: z.number().nullable(),
      pregnancy_intention: z.string().nullable(),
      user_goal: z.string().nullable(),
    })
    .nullable(),
  stats: z.object({ health_logs: z.number(), reminders: z.number() }),
});
export type HealthData = z.infer<typeof healthSchema>;

const keys = {
  roles: ['access', 'roles'] as const,
  audit: (q: Record<string, QueryValue>) => ['access', 'audit', q] as const,
  requests: (q: Record<string, QueryValue>) => ['access', 'requests', q] as const,
  health: (id: number) => ['access', 'health', id] as const,
};

export function useRoles() {
  return useQuery({ queryKey: keys.roles, queryFn: ({ signal }) => api.get('/roles', { schema: rolesSchema, signal }) });
}

export function useSaveRole() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ role, permissions }: { role: string; permissions: Record<string, string> }) =>
      api.put(`/roles/${role}`, { permissions }),
    onSuccess: () => client.invalidateQueries({ queryKey: keys.roles }),
  });
}

export function useResetRole() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (role: string) => api.post(`/roles/${role}/reset`),
    onSuccess: () => client.invalidateQueries({ queryKey: keys.roles }),
  });
}

export function useAuditLog(query: Record<string, QueryValue>) {
  return useQuery({
    queryKey: keys.audit(query),
    queryFn: ({ signal }) => api.get('/audit-logs', { query, schema: pagedSchema(auditRowSchema, {}), signal }),
    placeholderData: keepPreviousData,
  });
}

/** Downloads the filtered audit log as CSV (session cookie; the export itself is audited). */
export async function downloadAuditCsv(query: Record<string, QueryValue>): Promise<void> {
  const res = await fetch(buildUrl(ADMIN_API_BASE_URL, '/audit-logs/export', query), { credentials: 'include' });
  if (!res.ok) throw new ApiError({ status: res.status, code: res.status === 403 ? 'forbidden' : 'http_error' });
  const url = URL.createObjectURL(await res.blob());
  const a = document.createElement('a');
  a.href = url;
  a.download = 'audit-log.csv';
  a.click();
  URL.revokeObjectURL(url);
}

export function useAccessRequests(query: Record<string, QueryValue>) {
  return useQuery({
    queryKey: keys.requests(query),
    queryFn: ({ signal }) => api.get('/health-access', { query, schema: pagedSchema(accessSchema, {}), signal }),
    placeholderData: keepPreviousData,
  });
}

export interface AccessInput {
  user_id: number | string;
  reason: string;
  ticket: string;
}

function useAccessMutation<V>(fn: (v: V) => Promise<unknown>) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: () => client.invalidateQueries({ queryKey: ['access'] }),
  });
}

export const useRequestAccess = () => useAccessMutation((body: AccessInput) => api.post('/health-access', body));
export const useDecideAccess = () =>
  useAccessMutation(({ id, action }: { id: number; action: 'approve' | 'reject' | 'revoke' }) =>
    api.post(`/health-access/${id}/${action}`),
  );

/** GET /health-access/users/:id — only while the admin's grant is live (403 health_access_required otherwise). */
export function useHealthData(userId: number | null) {
  return useQuery({
    queryKey: keys.health(userId ?? 0),
    queryFn: ({ signal }) => api.get(`/health-access/users/${userId}`, { schema: healthSchema, signal }),
    enabled: userId !== null,
    retry: false,
    staleTime: 0,
    gcTime: 0, // never keep health data in the cache after the dialog closes
  });
}
