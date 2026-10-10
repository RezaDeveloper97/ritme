import { describe, expect, it } from 'vitest';

import type { CourseListItem } from '@/entities/course';

import { filterCourses } from './filter';

const c = (id: number, kinds: CourseListItem['lesson_kinds']): CourseListItem => ({
  id,
  kind: 'course',
  title: String(id),
  description: null,
  status: 'published',
  published_at: null,
  lessons_count: kinds.length,
  published_lessons: 0,
  duration_seconds: 0,
  created_at: null,
  lesson_kinds: kinds,
});

describe('filterCourses', () => {
  it('keeps courses with a lesson of the kind', () => {
    const list = [c(1, ['video', 'pdf']), c(2, ['audio']), c(3, [])];
    expect(filterCourses(list, 'all')).toHaveLength(3);
    expect(filterCourses(list, 'pdf').map((x) => x.id)).toEqual([1]);
    expect(filterCourses(list, 'audio').map((x) => x.id)).toEqual([2]);
  });
});
