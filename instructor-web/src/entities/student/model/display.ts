import { formatMobile, toPersianDigits } from '@/shared/lib';

import type { Grant } from './schema';

/** Phone as the instructor typed it, grouped and in Persian digits (`۰۹۱۲ ۳۴۵ ۶۷۸۹`). */
export const phoneLabel = (phone: string): string => toPersianDigits(formatMobile(phone));

/** Avatar letter: first letter of the name, «؟» while the name is still hidden (H1: not opened a course yet). */
export function initialOf(g: Pick<Grant, 'student_name'>): string {
  const name = g.student_name?.trim();
  return name ? [...name][0]! : '؟';
}

/** Where a grant points: group name, course title, or null. */
export function targetOf(g: Grant): { kind: 'group' | 'course'; name: string } | null {
  if (g.group) return { kind: 'group', name: g.group.name ?? '' };
  if (g.course) return { kind: 'course', name: g.course.title ?? '' };
  return null;
}
