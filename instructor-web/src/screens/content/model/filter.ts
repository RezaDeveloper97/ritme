import type { CourseListItem, LessonKind } from '@/entities/course';

export type ContentFilter = 'all' | LessonKind;
export const CONTENT_FILTERS: readonly ContentFilter[] = ['all', 'video', 'audio', 'pdf'];

/** Courses holding at least one lesson of the kind (all → every course). */
export function filterCourses(courses: CourseListItem[], filter: ContentFilter): CourseListItem[] {
  if (filter === 'all') return courses;
  return courses.filter((c) => c.lesson_kinds.includes(filter));
}
