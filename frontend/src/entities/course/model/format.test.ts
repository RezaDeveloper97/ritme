import { describe, expect, it } from 'vitest';

import { myCoursesSchema } from '../api/schema';
import { filterCourses, formatClock, groupByInstructor, minutesOf, newCourses, percentAt } from './format';
import type { StudentCourse } from './types';

function course(id: number, over: Partial<StudentCourse> = {}): StudentCourse {
  return {
    id,
    kind: 'course',
    title: `c${id}`,
    description: null,
    instructor: { id: 1, name: 'Leila', title: null },
    lessonsCount: 3,
    completedLessons: 0,
    primaryKind: 'video',
    durationSeconds: 600,
    pageCount: 0,
    progressPercent: 0,
    access: { grantId: id, unlimited: true, expiresAt: null, daysLeft: null, expired: false, grantedAt: null, group: null },
    ...over,
  };
}

describe('course format', () => {
  it('formats clocks and minutes', () => {
    expect(formatClock(302)).toBe('05:02');
    expect(formatClock(3725)).toBe('1:02:05');
    expect(formatClock(Number.NaN)).toBe('00:00');
    expect(minutesOf(0)).toBe(0);
    expect(minutesOf(20)).toBe(1);
    expect(minutesOf(420)).toBe(7);
    expect(percentAt(30, 120)).toBe(25);
    expect(percentAt(30, 0)).toBe(0);
  });

  it('filters and groups courses', () => {
    const list = [course(1, { progressPercent: 35 }), course(2, { progressPercent: 100, instructor: { id: 2, name: 'Sara', title: null } }), course(3)];
    expect(filterCourses(list, 'learning').map((c) => c.id)).toEqual([1, 3]);
    expect(filterCourses(list, 'done').map((c) => c.id)).toEqual([2]);
    expect(filterCourses(list, 'all')).toHaveLength(3);
    expect(groupByInstructor(list).map((g) => g.courses.map((c) => c.id))).toEqual([[1, 3], [2]]);
  });

  it('picks untouched recent grants as new', () => {
    const now = new Date('2026-10-08T10:00:00+03:30');
    const recent = course(1, { access: { ...course(1).access, grantedAt: '2026-10-07T10:00:00+03:30' } });
    const old = course(2, { access: { ...course(2).access, grantedAt: '2026-08-01T10:00:00+03:30' } });
    const started = course(3, { progressPercent: 10 });
    expect(newCourses([old, recent, started], now).map((c) => c.id)).toEqual([1]);
  });

  it('parses the my-courses payload and drops broken rows', () => {
    const parsed = myCoursesSchema.parse({
      continue: null,
      courses: [
        {
          id: 5,
          kind: 'course',
          title: 'Birth prep',
          description: null,
          instructor: { id: 1, name: 'Leila', title: 'Midwife' },
          lessons_count: 2,
          completed_lessons: 1,
          primary_kind: 'video',
          duration_seconds: 900,
          page_count: 0,
          progress_percent: 50,
          access: { grant_id: 9, unlimited: false, expires_at: '2026-12-20T00:00:00+03:30', days_left: 73, expired: false, granted_at: null, group: null },
        },
        { id: 'broken' },
      ],
    });
    expect(parsed.courses).toHaveLength(1);
    expect(parsed.courses[0]?.access.daysLeft).toBe(73);
  });
});
