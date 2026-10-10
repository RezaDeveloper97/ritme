import type { IconName, Tone } from '@/shared/ui';

import type { CourseFilter, LessonKind, MyCourses, StudentCourse } from './types';

/** Thumb look of a lesson kind (Learn_Hub: video = pink, audio = purple, PDF = amber). */
export const KIND_LOOK: Record<LessonKind, { icon: IconName; tone: Tone }> = {
  video: { icon: 'video', tone: 'period' },
  audio: { icon: 'music', tone: 'brand' },
  pdf: { icon: 'fileDoc', tone: 'warm' },
};

/** `mm:ss` (or `h:mm:ss`) of a player position, Latin digits like the artboards. */
export function formatClock(seconds: number): string {
  const s = Math.max(0, Math.floor(Number.isFinite(seconds) ? seconds : 0));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const pad = (n: number) => String(n).padStart(2, '0');
  return h ? `${h}:${pad(m)}:${pad(s % 60)}` : `${pad(m)}:${pad(s % 60)}`;
}

/** Whole minutes for copy («۷ دقیقه مانده»), at least 1 when anything is left. */
export function minutesOf(seconds: number | null | undefined): number {
  if (!seconds || seconds <= 0) return 0;
  return Math.max(1, Math.round(seconds / 60));
}

/** «در حال یادگیری» / «تمام‌شده» / «همه». */
export function filterCourses(courses: readonly StudentCourse[], filter: CourseFilter): StudentCourse[] {
  if (filter === 'all') return [...courses];
  if (filter === 'done') return courses.filter((c) => c.progressPercent >= 100);
  return courses.filter((c) => c.progressPercent < 100);
}

export interface InstructorGroup {
  instructor: StudentCourse['instructor'];
  /** The group the first course came through («گروه مهر ۱۴۰۵»), if any. */
  groupName: string | null;
  courses: StudentCourse[];
}

/** Courses grouped under «از <مدرس>», in first-seen order (the API sorts by last opened). */
export function groupByInstructor(courses: readonly StudentCourse[]): InstructorGroup[] {
  const out: InstructorGroup[] = [];
  for (const c of courses) {
    let g = out.find((x) => x.instructor.id === c.instructor.id);
    if (!g) {
      g = { instructor: c.instructor, groupName: c.access.group?.name ?? null, courses: [] };
      out.push(g);
    }
    g.courses.push(c);
  }
  return out;
}

const NEW_WINDOW_MS = 14 * 24 * 3600 * 1000;

/** Untouched open courses granted in the last 14 days, newest first (home card «جدید · از …»). */
export function newCourses(courses: readonly StudentCourse[], now: Date = new Date()): StudentCourse[] {
  return courses
    .filter((c) => !c.access.expired && c.progressPercent === 0 && c.completedLessons === 0)
    .filter((c) => !c.access.grantedAt || now.getTime() - new Date(c.access.grantedAt).getTime() < NEW_WINDOW_MS)
    .sort((a, b) => (b.access.grantedAt ?? '').localeCompare(a.access.grantedAt ?? ''));
}

/** Percent watched of a media lesson at `position` of `duration` seconds. */
export function percentAt(position: number, duration: number): number {
  if (!duration || !Number.isFinite(duration)) return 0;
  return Math.max(0, Math.min(100, Math.round((position / duration) * 100)));
}

/** B-N8-04: a download stays only while «دوره‌های من» still lists its course with live access. */
export function stillAccessible(mine: MyCourses, courseId: number): boolean {
  const c = mine.courses.find((x) => x.id === courseId);
  return !!c && !c.access.expired;
}
