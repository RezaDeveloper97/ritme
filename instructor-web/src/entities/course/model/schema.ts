import { z } from 'zod';

/** Shapes of backend-go internal/learning (json.go, dashboard.go) — instructor view. */
export const courseKindSchema = z.enum(['course', 'standalone']);
export const publishStatusSchema = z.enum(['draft', 'published']);
export const lessonKindSchema = z.enum(['video', 'audio', 'pdf']);
export type CourseKind = z.infer<typeof courseKindSchema>;
export type PublishStatus = z.infer<typeof publishStatusSchema>;
export type LessonKind = z.infer<typeof lessonKindSchema>;

export const courseListItemSchema = z.object({
  id: z.number().int(),
  kind: courseKindSchema,
  title: z.string(),
  description: z.string().nullable(),
  status: publishStatusSchema,
  published_at: z.string().nullable(),
  lessons_count: z.number().int(),
  published_lessons: z.number().int(),
  duration_seconds: z.number().int(),
  created_at: z.string().nullable(),
  lesson_kinds: z.array(lessonKindSchema).default([]),
});
export type CourseListItem = z.infer<typeof courseListItemSchema>;

export const coursesSchema = z.object({ courses: z.array(courseListItemSchema) });

export const activitySchema = z.object({
  type: z.enum(['lesson_published', 'students_added']),
  at: z.string().nullable(),
  lesson: z.object({ id: z.number(), title: z.string() }).nullable(),
  course: z.object({ id: z.number(), title: z.string().nullable() }).nullable(),
  group: z.object({ id: z.number(), name: z.string().nullable() }).nullable(),
  count: z.number().int().nullable(),
});
export type Activity = z.infer<typeof activitySchema>;

export const dashboardSchema = z.object({
  stats: z.object({
    active_students: z.number().int(),
    new_students_week: z.number().int(),
    courses: z.number().int(),
    drafts: z.number().int(),
    watch_seconds_week: z.number().int(),
    groups: z.number().int(),
    group_members: z.number().int(),
  }),
  courses: z.array(courseListItemSchema),
  activity: z.array(activitySchema),
});
export type Dashboard = z.infer<typeof dashboardSchema>;

export const chapterSchema = z.object({
  id: z.number().int(),
  title: z.string(),
  sort_order: z.number().int(),
  unlock_at: z.string().nullable(),
  locked: z.boolean(),
});
export type Chapter = z.infer<typeof chapterSchema>;

export const lessonSchema = z.object({
  id: z.number().int(),
  chapter_id: z.number().int().nullable(),
  kind: lessonKindSchema,
  title: z.string(),
  description: z.string().nullable(),
  duration_seconds: z.number().int().nullable(),
  page_count: z.number().int().nullable(),
  size_bytes: z.number().int().nullable(),
  status: publishStatusSchema,
  published_at: z.string().nullable(),
  sort_order: z.number().int(),
  media: z.object({ id: z.number().int(), status: z.string() }).nullable(),
  media_status: z.string(),
});
export type Lesson = z.infer<typeof lessonSchema>;

export const outlineSchema = z.object({
  course: z.object({
    id: z.number().int(),
    kind: courseKindSchema,
    title: z.string(),
    description: z.string().nullable(),
    status: publishStatusSchema,
    published_at: z.string().nullable(),
    created_at: z.string().nullable(),
  }),
  chapters: z.array(chapterSchema),
  lessons: z.array(lessonSchema),
  duration_seconds: z.number().int(),
  access: z.object({
    students: z.number().int(),
    direct: z.number().int(),
    groups: z.array(z.object({ id: z.number().int(), name: z.string(), members: z.number().int() })),
  }),
});
export type Outline = z.infer<typeof outlineSchema>;

export const chapterEnvelopeSchema = z.object({ chapter: chapterSchema });
export const lessonEnvelopeSchema = z.object({ lesson: lessonSchema });

export interface CourseInput {
  kind?: CourseKind;
  title?: string;
  description?: string | null;
  status?: PublishStatus;
}

export interface ChapterInput {
  title?: string;
  /** Tehran wall clock "Y-m-d H:i", null opens the chapter. */
  unlock_at?: string | null;
  sort_order?: number;
}

export interface LessonInput {
  kind?: LessonKind;
  title?: string;
  chapter_id?: number | null;
  description?: string | null;
  size_bytes?: number | null;
  status?: PublishStatus;
  sort_order?: number;
}

/** Server limits (internal/learning MaxTitleLen / MaxDescriptionLen). */
export const COURSE_LIMITS = { titleMax: 150, descriptionMax: 5000 } as const;
