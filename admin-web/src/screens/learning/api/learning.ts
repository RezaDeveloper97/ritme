'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api, createResource, pagedSchema } from '@/shared/api';

/** Instructors & courses moderation «مدرسین و دوره‌ها» (admin-api.md §20, B-N8-08). */

const list = <T extends z.ZodTypeAny>(item: T) => z.preprocess((v) => (Array.isArray(v) ? v : []), z.array(item));
const adminRefSchema = z.object({ id: z.number(), name: z.string().nullable() }).nullable().catch(null);
const countsSchema = z.record(z.string(), z.number());

export const instructorSchema = z.object({
  id: z.number(),
  display_name: z.string(),
  title: z.string().nullable(),
  bio: z.string().nullable(),
  status: z.string(),
  user: z.object({ id: z.number(), name: z.string().nullable(), mobile: z.string().nullable() }),
  courses_count: z.number(),
  published_courses: z.number(),
  students_count: z.number(),
  approved_at: z.string().nullable(),
  approved_by: adminRefSchema,
  revoked_at: z.string().nullable(),
  created_at: z.string().nullable(),
});
export type Instructor = z.infer<typeof instructorSchema>;

const historySchema = z.object({
  id: z.number(),
  action: z.string(),
  target_type: z.string(),
  target_id: z.number(),
  admin: adminRefSchema,
  note: z.string().nullable(),
  created_at: z.string().nullable(),
});
export type ModerationEntry = z.infer<typeof historySchema>;

export const instructorsApi = createResource({
  key: 'learning-instructors',
  path: '/learning/instructors',
  list: pagedSchema(instructorSchema, {
    filters: z.object({ status: z.string(), q: z.string() }).partial().optional(),
    counts: countsSchema.optional(),
  }),
  detail: z.object({ instructor: instructorSchema, history: list(historySchema) }),
  alsoInvalidate: [['learning-stats'], ['learning-courses']],
});

const courseRowSchema = z.object({
  id: z.number(),
  title: z.string(),
  kind: z.string(),
  status: z.string(),
  instructor: z.object({ id: z.number(), display_name: z.string(), status: z.string() }),
  chapters_count: z.number(),
  lessons_count: z.number(),
  published_lessons: z.number(),
  review_open: z.number(),
  flagged_count: z.number(),
  students_count: z.number(),
  completion_percent: z.number(),
  completed_count: z.number(),
  published_at: z.string().nullable(),
  created_at: z.string().nullable(),
});
export type CourseRow = z.infer<typeof courseRowSchema>;

export const reviewSchema = z.object({
  state: z.string(),
  status: z.string(),
  reviewed_at: z.string().nullable(),
  reviewed_by: adminRefSchema,
  note: z.string().nullable(),
});

export const lessonSchema = z.object({
  id: z.number(),
  chapter_id: z.number().nullable(),
  kind: z.string(),
  title: z.string(),
  description: z.string().nullable(),
  duration_seconds: z.number().nullable(),
  page_count: z.number().nullable(),
  size_bytes: z.number().nullable(),
  media_status: z.string(),
  status: z.string(),
  published_at: z.string().nullable(),
  review: reviewSchema,
  updated_at: z.string().nullable(),
});
export type Lesson = z.infer<typeof lessonSchema>;

const grantCountsSchema = z.object({
  all: z.number(),
  active: z.number(),
  pending: z.number(),
  expired: z.number(),
  revoked: z.number(),
});

const courseSchema = z.object({
  id: z.number(),
  title: z.string(),
  description: z.string().nullable(),
  kind: z.string(),
  status: z.string(),
  instructor: z.object({ id: z.number(), display_name: z.string(), title: z.string().nullable(), status: z.string() }),
  chapters: list(z.object({ id: z.number(), title: z.string(), sort_order: z.number(), unlock_at: z.string().nullable() })),
  lessons: list(lessonSchema.extend({ sort_order: z.number() })),
  grants: grantCountsSchema,
  usage: z.object({ students_count: z.number(), completion_percent: z.number(), completed_count: z.number() }),
  published_at: z.string().nullable(),
  created_at: z.string().nullable(),
});
export type Course = z.infer<typeof courseSchema>;

export const coursesApi = createResource({
  key: 'learning-courses',
  path: '/learning/courses',
  list: pagedSchema(courseRowSchema, {
    filters: z.object({ status: z.string(), q: z.string(), instructor_id: z.number().nullable() }).partial().optional(),
    counts: countsSchema.optional(),
  }),
  detail: z.object({ course: courseSchema }),
});

export const reviewItemSchema = lessonSchema.extend({
  course: z.object({ id: z.number(), title: z.string(), kind: z.string(), status: z.string() }),
  instructor: z.object({ id: z.number(), display_name: z.string(), status: z.string() }),
});
export type ReviewItem = z.infer<typeof reviewItemSchema>;

export const reviewsApi = createResource({
  key: 'learning-reviews',
  path: '/learning/reviews',
  list: pagedSchema(reviewItemSchema, {
    filters: z.object({ state: z.string() }).partial().optional(),
    counts: countsSchema.optional(),
  }),
  detail: z.unknown(),
});

const statsSchema = z.object({
  instructors: countsSchema,
  courses: countsSchema,
  lessons: z.object({ all: z.number(), published: z.number() }),
  review: countsSchema,
  grants: countsSchema,
  students: z.object({ with_access: z.number(), active_30d: z.number() }),
  completion: z.object({
    percent: z.number(),
    enrolments: z.number(),
    completed: z.number(),
    lessons_completed: z.number(),
  }),
  active_days: z.number(),
});
export type LearningStats = z.infer<typeof statsSchema>;

/** GET /learning/stats. */
export function useLearningStats() {
  return useQuery({
    queryKey: ['learning-stats'],
    queryFn: ({ signal }) => api.get('/learning/stats', { schema: statsSchema, signal }),
  });
}

/** Every moderation write makes the module's lists and the stats stale. */
function useInvalidateLearning() {
  const client = useQueryClient();
  return () => {
    for (const key of ['learning-stats', 'learning-instructors', 'learning-courses', 'learning-reviews']) {
      void client.invalidateQueries({ queryKey: [key] });
    }
  };
}

export type InstructorAction = 'approve' | 'revoke';

/** POST /learning/instructors/:id/approve|revoke {note?} — super admins only. */
export function useInstructorAction() {
  const invalidate = useInvalidateLearning();
  return useMutation({
    mutationFn: ({ id, action, note }: { id: number; action: InstructorAction; note?: string }) =>
      api.post(`/learning/instructors/${id}/${action}`, note ? { note } : {}, { schema: z.unknown() }),
    onSuccess: invalidate,
  });
}

export type LessonAction = 'approve' | 'flag' | 'unpublish';

/** POST /learning/lessons/:id/approve|flag|unpublish {note} — any admin; flag / unpublish need a note. */
export function useLessonAction() {
  const invalidate = useInvalidateLearning();
  return useMutation({
    mutationFn: ({ id, action, note }: { id: number; action: LessonAction; note?: string }) =>
      api.post(`/learning/lessons/${id}/${action}`, note ? { note } : {}, { schema: z.unknown() }),
    onSuccess: invalidate,
  });
}
