import { z } from 'zod';

/** InstructorJSON (backend-go internal/learning/json.go). */
export const instructorStatusSchema = z.enum(['pending', 'approved', 'revoked']);
export type InstructorStatus = z.infer<typeof instructorStatusSchema>;

export const instructorSchema = z.object({
  id: z.number().int(),
  display_name: z.string(),
  title: z.string().nullable(),
  bio: z.string().nullable(),
  status: instructorStatusSchema,
  approved_at: z.string().nullable(),
  created_at: z.string().nullable(),
});
export type Instructor = z.infer<typeof instructorSchema>;

/** GET /me and POST /apply: `{instructor}` (null when she never applied). */
export const meSchema = z.object({ instructor: instructorSchema.nullable() });
export type Me = z.infer<typeof meSchema>;

export interface ApplyInput {
  display_name: string;
  title?: string | null;
  bio?: string | null;
}

/** Server rules of POST /apply (ValidateApply), mirrored for instant feedback. */
export const APPLY_LIMITS = { displayNameMin: 2, displayNameMax: 80, titleMax: 60, bioMax: 500 } as const;
