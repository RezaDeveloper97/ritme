import { describe, expect, it } from 'vitest';

import { groupByChapter, hoursOf, lessonKindOfFile, minutesOf, toTehranWallClock } from './format';
import type { Chapter, Lesson } from './schema';

const lesson = (id: number, chapter_id: number | null, sort_order: number): Lesson => ({
  id,
  chapter_id,
  kind: 'video',
  title: `L${id}`,
  description: null,
  duration_seconds: 600,
  page_count: null,
  size_bytes: null,
  status: 'draft',
  published_at: null,
  sort_order,
  media: null,
  media_status: 'none',
});
const chapter = (id: number, sort_order: number): Chapter => ({
  id,
  title: `C${id}`,
  sort_order,
  unlock_at: null,
  locked: false,
});

describe('course format helpers', () => {
  it('maps files to lesson kinds', () => {
    expect(lessonKindOfFile({ type: 'video/mp4', name: 'a.mp4' })).toBe('video');
    expect(lessonKindOfFile({ type: '', name: 'a.MP3' })).toBe('audio');
    expect(lessonKindOfFile({ type: 'application/pdf', name: 'x' })).toBe('pdf');
    expect(lessonKindOfFile({ type: 'image/png', name: 'a.png' })).toBeNull();
  });

  it('rounds durations', () => {
    expect(minutesOf(null)).toBeNull();
    expect(minutesOf(20)).toBe(1);
    expect(minutesOf(1080)).toBe(18);
    expect(hoursOf(5400)).toBe(1.5);
    expect(hoursOf(3600 * 320)).toBe(320);
  });

  it('groups lessons by chapter order, loose lessons last', () => {
    const groups = groupByChapter([chapter(2, 2), chapter(1, 1)], [lesson(3, 2, 1), lesson(1, 1, 2), lesson(2, 1, 1), lesson(4, null, 3)]);
    expect(groups.map((g) => g.chapter?.id ?? null)).toEqual([1, 2, null]);
    expect(groups[0]?.lessons.map((l) => l.id)).toEqual([2, 1]);
  });

  it('formats Tehran wall clock', () => {
    expect(toTehranWallClock('2026-10-07T06:30:00Z')).toBe('2026-10-07 10:00');
    expect(toTehranWallClock(null)).toBe('');
  });
});
