/*
 * Student side of the courses domain (B-N8-01 `/api/v1/learning/*`, B-N8-02
 * playback). Camel-cased from backend-go/internal/learning/json.go.
 */

export const LESSON_KINDS = ['video', 'audio', 'pdf'] as const;
export type LessonKind = (typeof LESSON_KINDS)[number];

export interface InstructorRef {
  id: number;
  name: string;
  title: string | null;
}

export interface CourseAccess {
  grantId: number;
  unlimited: boolean;
  expiresAt: string | null;
  daysLeft: number | null;
  expired: boolean;
  grantedAt: string | null;
  group: { id: number; name: string | null } | null;
}

/** One card of «دوره‌های من». */
export interface StudentCourse {
  id: number;
  kind: string;
  title: string;
  description: string | null;
  instructor: InstructorRef;
  lessonsCount: number;
  completedLessons: number;
  primaryKind: LessonKind | null;
  durationSeconds: number;
  pageCount: number;
  progressPercent: number;
  access: CourseAccess;
}

export interface LessonProgress {
  positionSeconds: number;
  percent: number;
  completed: boolean;
  lastSeenAt: string | null;
}

export interface StudentLesson {
  id: number;
  number: number;
  chapterId: number | null;
  kind: LessonKind;
  title: string;
  description: string | null;
  durationSeconds: number | null;
  pageCount: number | null;
  sizeBytes: number | null;
  isNew: boolean;
  locked: boolean;
  unlockAt: string | null;
  progress: LessonProgress | null;
  media: { id: number; status: string } | null;
}

export interface CourseChapter {
  id: number;
  title: string;
  locked: boolean;
  unlockAt: string | null;
  lessons: StudentLesson[];
}

/** Learn_Course. */
export interface CourseOutline {
  course: StudentCourse;
  nextLesson: { id: number; number: number; title: string } | null;
  chapters: CourseChapter[];
  /** Lessons outside any chapter. */
  lessons: StudentLesson[];
}

/** The home card / hub hero «ادامه از جایی که ماندی». */
export interface ContinueCard {
  course: { id: number; title: string; kind: string; instructor: InstructorRef };
  lesson: { id: number; number: number; kind: LessonKind; title: string; durationSeconds: number | null };
  positionSeconds: number;
  remainingSeconds: number | null;
  percent: number;
  coursePercent: number;
}

export interface MyCourses {
  continue: ContinueCard | null;
  courses: StudentCourse[];
}

/** GET /learning/lessons/{id}. */
export interface LessonDetail {
  lesson: StudentLesson;
  course: { id: number; title: string; kind: string; lessonsCount: number; instructor: InstructorRef };
  nextLessonId: number | null;
}

export interface Playback {
  mediaId: number;
  kind: string;
  mime: string;
  url: string;
  expiresAt: string | null;
}

/** Learn_Unlocked. */
export interface UnlockedGrant {
  grantId: number;
  instructor: InstructorRef;
  group: { id: number; name: string | null } | null;
  courses: {
    id: number;
    kind: string;
    title: string;
    lessonsCount: number;
    primaryKind: LessonKind | null;
    durationSeconds: number;
    pageCount: number;
  }[];
  unlimited: boolean;
  expiresAt: string | null;
  daysLeft: number | null;
  expired: boolean;
}

export type CourseFilter = 'learning' | 'done' | 'all';

/** Why a lesson can't be opened (403 codes of B-N8-01). */
export type LessonBlock = 'expired' | 'locked' | 'notFound';
