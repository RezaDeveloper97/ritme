import { z } from 'zod';

/** Shapes of backend-go internal/learning (json.go GrantJSON, GroupJSON) — the instructor's view of access. */
export const grantStatusSchema = z.enum(['active', 'pending', 'expired']);
export type GrantStatus = z.infer<typeof grantStatusSchema>;

export const grantSchema = z.object({
  id: z.number().int(),
  phone: z.string(),
  status: grantStatusSchema,
  registered: z.boolean().optional(),
  student_name: z.string().nullable().optional(),
  group: z.object({ id: z.number().int(), name: z.string().nullable().optional() }).nullable().optional(),
  course: z.object({ id: z.number().int(), title: z.string().nullable().optional() }).nullable().optional(),
  duration: z.object({
    type: z.enum(['unlimited', 'days', 'until']),
    days: z.number().nullable().optional(),
    until: z.string().nullable().optional(),
  }),
  expires_at: z.string().nullable().optional(),
  days_left: z.number().nullable().optional(),
  progress_percent: z.number().nullable().optional(),
  created_at: z.string().nullable().optional(),
});
export type Grant = z.infer<typeof grantSchema>;

export const studentsSchema = z.object({
  counts: z.object({ active: z.number().int(), pending: z.number().int(), expired: z.number().int() }),
  students: z.array(grantSchema),
});

const groupCourseSchema = z.object({ id: z.number().int(), kind: z.string(), title: z.string(), status: z.string() });
export const groupSchema = z.object({
  id: z.number().int(),
  name: z.string(),
  courses: z.array(groupCourseSchema),
  members: z.object({ active: z.number().int(), pending: z.number().int() }),
});
export type Group = z.infer<typeof groupSchema>;
export const groupsSchema = z.object({ groups: z.array(groupSchema) });

export const groupDetailSchema = z.object({
  group: z.object({ id: z.number().int(), name: z.string(), courses: z.array(groupCourseSchema) }),
  members: z.array(grantSchema),
  pending_signups: z.number().int(),
});
export type GroupDetail = z.infer<typeof groupDetailSchema>;

export const GROUP_NAME_MAX = 100;
export interface GroupInput {
  name?: string;
  course_ids?: number[];
}
