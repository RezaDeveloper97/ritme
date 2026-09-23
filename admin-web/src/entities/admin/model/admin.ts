import { z } from 'zod';

/** The `Admin` object (admin-api.md §4). */
export const adminRoleSchema = z.enum(['super', 'editor']);
export type AdminRole = z.infer<typeof adminRoleSchema>;

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
