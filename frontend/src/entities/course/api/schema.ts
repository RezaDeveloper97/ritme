import { z } from 'zod';

import {
  LESSON_KINDS,
  type ContinueCard,
  type CourseAccess,
  type CourseOutline,
  type InstructorRef,
  type LessonDetail,
  type MyCourses,
  type Playback,
  type StudentCourse,
  type StudentLesson,
  type UnlockedGrant,
} from '../model/types';

/*
 * Parsers of `/api/v1/learning/*` (B-N8-01/02; backend-go/internal/learning/json.go,
 * internal/media/handlers.go). Snake case → camel case.
 */

const str = z.string().nullable().catch(null);
const int = z.number().nullable().catch(null);
const kind = z.enum(LESSON_KINDS);

const instructorRaw = z.object({ id: z.number(), name: z.string(), title: str });
const toInstructor = (r: z.infer<typeof instructorRaw>): InstructorRef => ({ id: r.id, name: r.name, title: r.title });

const groupRaw = z.object({ id: z.number(), name: str }).nullable().catch(null);

const accessRaw = z.object({
  grant_id: z.number(),
  unlimited: z.boolean().catch(false),
  expires_at: str,
  days_left: int,
  expired: z.boolean().catch(false),
  granted_at: str,
  group: groupRaw,
});
const toAccess = (r: z.infer<typeof accessRaw>): CourseAccess => ({
  grantId: r.grant_id,
  unlimited: r.unlimited,
  expiresAt: r.expires_at,
  daysLeft: r.days_left,
  expired: r.expired,
  grantedAt: r.granted_at,
  group: r.group,
});

const courseRaw = z.object({
  id: z.number(),
  kind: z.string(),
  title: z.string(),
  description: str,
  instructor: instructorRaw,
  lessons_count: z.number().catch(0),
  completed_lessons: z.number().catch(0),
  primary_kind: kind.nullable().catch(null),
  duration_seconds: z.number().catch(0),
  page_count: z.number().catch(0),
  progress_percent: z.number().catch(0),
  access: accessRaw,
});
const toCourse = (r: z.infer<typeof courseRaw>): StudentCourse => ({
  id: r.id,
  kind: r.kind,
  title: r.title,
  description: r.description,
  instructor: toInstructor(r.instructor),
  lessonsCount: r.lessons_count,
  completedLessons: r.completed_lessons,
  primaryKind: r.primary_kind,
  durationSeconds: r.duration_seconds,
  pageCount: r.page_count,
  progressPercent: r.progress_percent,
  access: toAccess(r.access),
});

const progressRaw = z
  .object({ position_seconds: z.number(), percent: z.number(), completed: z.boolean(), last_seen_at: str })
  .nullable()
  .catch(null);

const lessonRaw = z.object({
  id: z.number(),
  number: z.number().catch(0),
  chapter_id: int,
  kind,
  title: z.string(),
  description: str,
  duration_seconds: int,
  page_count: int,
  size_bytes: int,
  is_new: z.boolean().catch(false),
  locked: z.boolean().catch(false),
  unlock_at: str,
  progress: progressRaw,
  media: z.object({ id: z.number(), status: z.string() }).nullable().catch(null),
});
const toLesson = (r: z.infer<typeof lessonRaw>): StudentLesson => ({
  id: r.id,
  number: r.number,
  chapterId: r.chapter_id,
  kind: r.kind,
  title: r.title,
  description: r.description,
  durationSeconds: r.duration_seconds,
  pageCount: r.page_count,
  sizeBytes: r.size_bytes,
  isNew: r.is_new,
  locked: r.locked,
  unlockAt: r.unlock_at,
  progress: r.progress
    ? {
        positionSeconds: r.progress.position_seconds,
        percent: r.progress.percent,
        completed: r.progress.completed,
        lastSeenAt: r.progress.last_seen_at,
      }
    : null,
  media: r.media,
});

/** Lessons of an unknown kind are dropped rather than failing the whole outline. */
function lessonList(raw: unknown[]): StudentLesson[] {
  const out: StudentLesson[] = [];
  for (const item of raw) {
    const p = lessonRaw.safeParse(item);
    if (p.success) out.push(toLesson(p.data));
  }
  return out;
}

const continueRaw = z
  .object({
    course: z.object({ id: z.number(), title: z.string(), kind: z.string(), instructor: instructorRaw }),
    lesson: z.object({ id: z.number(), number: z.number(), kind, title: z.string(), duration_seconds: int }),
    position_seconds: z.number().catch(0),
    remaining_seconds: int,
    percent: z.number().catch(0),
    course_percent: z.number().catch(0),
  })
  .nullable()
  .catch(null);
const toContinue = (r: NonNullable<z.infer<typeof continueRaw>>): ContinueCard => ({
  course: { id: r.course.id, title: r.course.title, kind: r.course.kind, instructor: toInstructor(r.course.instructor) },
  lesson: { id: r.lesson.id, number: r.lesson.number, kind: r.lesson.kind, title: r.lesson.title, durationSeconds: r.lesson.duration_seconds },
  positionSeconds: r.position_seconds,
  remainingSeconds: r.remaining_seconds,
  percent: r.percent,
  coursePercent: r.course_percent,
});

export const myCoursesSchema = z
  .object({ continue: continueRaw, courses: z.array(z.unknown()) })
  .transform(
    (r): MyCourses => ({
      continue: r.continue ? toContinue(r.continue) : null,
      courses: r.courses.flatMap((c) => {
        const p = courseRaw.safeParse(c);
        return p.success ? [toCourse(p.data)] : [];
      }),
    }),
  );

export const outlineSchema = z
  .object({
    course: courseRaw,
    next_lesson: z.object({ id: z.number(), number: z.number(), title: z.string() }).nullable().catch(null),
    chapters: z.array(
      z.object({ id: z.number(), title: z.string(), locked: z.boolean().catch(false), unlock_at: str, lessons: z.array(z.unknown()) }),
    ),
    lessons: z.array(z.unknown()),
  })
  .transform(
    (r): CourseOutline => ({
      course: toCourse(r.course),
      nextLesson: r.next_lesson,
      chapters: r.chapters.map((c) => ({ id: c.id, title: c.title, locked: c.locked, unlockAt: c.unlock_at, lessons: lessonList(c.lessons) })),
      lessons: lessonList(r.lessons),
    }),
  );

export const lessonDetailSchema = z
  .object({
    lesson: lessonRaw,
    course: z.object({ id: z.number(), title: z.string(), kind: z.string(), lessons_count: z.number().catch(0), instructor: instructorRaw }),
    next_lesson_id: int,
  })
  .transform(
    (r): LessonDetail => ({
      lesson: toLesson(r.lesson),
      course: {
        id: r.course.id,
        title: r.course.title,
        kind: r.course.kind,
        lessonsCount: r.course.lessons_count,
        instructor: toInstructor(r.course.instructor),
      },
      nextLessonId: r.next_lesson_id,
    }),
  );

export const playbackSchema = z
  .object({
    playback: z.object({ media_id: z.number(), kind: z.string(), mime: z.string().catch(''), url: z.string(), expires_at: str }),
  })
  .transform((r): Playback => ({ mediaId: r.playback.media_id, kind: r.playback.kind, mime: r.playback.mime, url: r.playback.url, expiresAt: r.playback.expires_at }));

export const progressSavedSchema = z.object({
  progress: z.object({ lesson_id: z.number(), position_seconds: z.number(), percent: z.number(), completed: z.boolean() }),
  course_percent: z.number().catch(0),
});

export const unlockedSchema = z
  .object({
    grant_id: z.number(),
    instructor: instructorRaw,
    group: groupRaw,
    courses: z.array(
      z.object({
        id: z.number(),
        kind: z.string(),
        title: z.string(),
        lessons_count: z.number().catch(0),
        primary_kind: kind.nullable().catch(null),
        duration_seconds: z.number().catch(0),
        page_count: z.number().catch(0),
      }),
    ),
    unlimited: z.boolean().catch(false),
    expires_at: str,
    days_left: int,
    expired: z.boolean().catch(false),
  })
  .transform(
    (r): UnlockedGrant => ({
      grantId: r.grant_id,
      instructor: toInstructor(r.instructor),
      group: r.group,
      courses: r.courses.map((c) => ({
        id: c.id,
        kind: c.kind,
        title: c.title,
        lessonsCount: c.lessons_count,
        primaryKind: c.primary_kind,
        durationSeconds: c.duration_seconds,
        pageCount: c.page_count,
      })),
      unlimited: r.unlimited,
      expiresAt: r.expires_at,
      daysLeft: r.days_left,
      expired: r.expired,
    }),
  );
