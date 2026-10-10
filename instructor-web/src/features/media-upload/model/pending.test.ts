import { describe, expect, it } from 'vitest';

import { isSameFile, nextChunk, percentOf, readPending, removePending, savePending, type PendingUpload } from './pending';

function memory() {
  const m = new Map<string, string>();
  return { getItem: (k: string) => m.get(k) ?? null, setItem: (k: string, v: string) => void m.set(k, v) };
}

const p = (mediaId: number, lessonId: number, startedAt = Date.now()): PendingUpload => ({
  mediaId,
  lessonId,
  courseId: 1,
  lessonTitle: 'x',
  name: 'a.mp4',
  size: 100,
  lastModified: 5,
  publish: false,
  standalone: false,
  startedAt,
});

describe('pending uploads', () => {
  it('saves, replaces per lesson and removes', () => {
    const kv = memory();
    savePending(p(1, 10), kv);
    savePending(p(2, 10), kv);
    savePending(p(3, 11), kv);
    expect(readPending(Date.now(), kv).map((x) => x.mediaId)).toEqual([2, 3]);
    removePending(2, kv);
    expect(readPending(Date.now(), kv).map((x) => x.mediaId)).toEqual([3]);
  });

  it('drops expired and malformed entries', () => {
    const kv = memory();
    kv.setItem('ritme_instructor_uploads', JSON.stringify([p(1, 1, 0), { bad: true }, p(2, 2)]));
    expect(readPending(Date.now(), kv).map((x) => x.mediaId)).toEqual([2]);
    kv.setItem('ritme_instructor_uploads', '{nope');
    expect(readPending(Date.now(), kv)).toEqual([]);
  });

  it('matches the same file and plans chunks', () => {
    expect(isSameFile(p(1, 1), { name: 'a.mp4', size: 100, lastModified: 5 })).toBe(true);
    expect(isSameFile(p(1, 1), { name: 'a.mp4', size: 101, lastModified: 5 })).toBe(false);
    expect(nextChunk(0, 100, 30)).toEqual([0, 30]);
    expect(nextChunk(90, 100, 30)).toEqual([90, 100]);
    expect(percentOf(68, 100)).toBe(68);
    expect(percentOf(999, 1000)).toBe(99);
    expect(percentOf(1000, 1000)).toBe(100);
  });
});
