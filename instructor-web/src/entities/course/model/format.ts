import type { Chapter, Lesson, LessonKind } from './schema';

const num = new Intl.NumberFormat('fa-IR');

/** Persian digits. */
export function faNum(n: number): string {
  return num.format(n);
}

/** Whole minutes (at least 1 when there is any duration), or null. */
export function minutesOf(seconds: number | null | undefined): number | null {
  if (!seconds || seconds <= 0) return null;
  return Math.max(1, Math.round(seconds / 60));
}

/** Watch hours, rounded down to one decimal under 10 h, whole above. */
export function hoursOf(seconds: number): number {
  const h = seconds / 3600;
  return h >= 10 ? Math.round(h) : Math.floor(h * 10) / 10;
}

/** Megabytes with one decimal under 10. */
export function megabytesOf(bytes: number): number {
  const mb = bytes / (1024 * 1024);
  return mb >= 10 ? Math.round(mb) : Math.round(mb * 10) / 10;
}

/** The lesson kind a picked file becomes (null = not accepted). */
export function lessonKindOfFile(file: { type: string; name: string }): LessonKind | null {
  const type = file.type.toLowerCase();
  if (type.startsWith('video/')) return 'video';
  if (type.startsWith('audio/')) return 'audio';
  if (type === 'application/pdf') return 'pdf';
  const ext = file.name.toLowerCase().split('.').pop() ?? '';
  if (['mp4', 'mov', 'webm'].includes(ext)) return 'video';
  if (['mp3', 'm4a', 'aac', 'ogg', 'wav', 'flac'].includes(ext)) return 'audio';
  if (ext === 'pdf') return 'pdf';
  return null;
}

/** The lessons under each chapter (chapter order), then the ones without a chapter. */
export function groupByChapter(
  chapters: Chapter[],
  lessons: Lesson[],
): { chapter: Chapter | null; lessons: Lesson[] }[] {
  const byOrder = (a: Lesson, b: Lesson) => a.sort_order - b.sort_order || a.id - b.id;
  const out: { chapter: Chapter | null; lessons: Lesson[] }[] = [...chapters]
    .sort((a, b) => a.sort_order - b.sort_order || a.id - b.id)
    .map((chapter) => ({ chapter, lessons: lessons.filter((l) => l.chapter_id === chapter.id).sort(byOrder) }));
  const known = new Set(chapters.map((c) => c.id));
  const loose = lessons.filter((l) => l.chapter_id === null || !known.has(l.chapter_id)).sort(byOrder);
  if (loose.length > 0 || out.length === 0) out.push({ chapter: null, lessons: loose });
  return out;
}

/** ISO → the "Y-m-d H:i" Tehran wall clock the chapter API takes (and the datetime-local value with a T). */
export function toTehranWallClock(iso: string | null): string {
  if (!iso) return '';
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Tehran',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23',
  }).formatToParts(new Date(iso));
  const get = (type: string) => parts.find((p) => p.type === type)?.value ?? '';
  return `${get('year')}-${get('month')}-${get('day')} ${get('hour')}:${get('minute')}`;
}
