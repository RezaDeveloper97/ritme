import { z } from 'zod';

/** The `Admin` object (admin-api.md §4; roles of B-N9-02 §21). */
export const ADMIN_ROLES = [
  'super',
  'editor',
  'product_manager',
  'medical_advisor',
  'support',
  'clinical_support',
  'analyst',
  'marketing',
  'engineering',
  'doctor',
  'instructor_moderator',
] as const;
export const adminRoleSchema = z.enum(ADMIN_ROLES);
export type AdminRole = z.infer<typeof adminRoleSchema>;

/** A capability level (admin-api.md §21): `view` = GET, `manage` = every method. */
export type PermissionLevel = 'view' | 'manage';
/** `{capability: level}` — what the admin's role grants (login / `/auth/me`). */
export const permissionsSchema = z.preprocess(
  (value) => (value && typeof value === 'object' && !Array.isArray(value) ? value : {}),
  z.record(z.string(), z.enum(['view', 'manage'])),
);
export type Permissions = Record<string, PermissionLevel>;

/** Whether `role` + `perms` grant `capability` at `level` (manage implies view; super has everything). */
export function can(role: AdminRole, perms: Permissions, capability: string, level: PermissionLevel = 'view'): boolean {
  if (role === 'super') return true;
  const got = perms[capability];
  return got === 'manage' || (got === 'view' && level === 'view');
}

export const adminSchema = z.object({
  id: z.number(),
  name: z.string(),
  email: z.string(),
  role: adminRoleSchema,
  is_super: z.boolean(),
  is_active: z.boolean(),
  last_login_at: z.string().nullable(),
  created_at: z.string().nullable(),
  updated_at: z.string().nullable(),
});
export type Admin = z.infer<typeof adminSchema>;

/** First visible character for the avatar (grapheme-safe enough for names). */
export function adminInitial(name: string): string {
  return Array.from(name.trim())[0] ?? '?';
}
