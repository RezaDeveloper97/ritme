import type { BadgeTone } from '@/shared/ui';

/** Review states of a lesson (admin-api.md §20). `open` = pending + changed + flagged. */
export const REVIEW_TABS = ['open', 'pending', 'changed', 'flagged', 'approved', 'all'] as const;
export type ReviewTab = (typeof REVIEW_TABS)[number];

export const INSTRUCTOR_TABS = ['all', 'pending', 'approved', 'revoked'] as const;
export const COURSE_STATUSES = ['all', 'draft', 'published'] as const;

/** Note limits of flag / unpublish / instructor actions (the API validates the same). */
export const NOTE_MIN = 3;
export const NOTE_MAX = 300;

const REVIEW_TONE: Record<string, BadgeTone> = { pending: 'amber', changed: 'data', flagged: 'red', approved: 'green' };
const INSTRUCTOR_TONE: Record<string, BadgeTone> = { pending: 'amber', approved: 'green', revoked: 'neutral' };
const PUBLICATION_TONE: Record<string, BadgeTone> = { published: 'green', draft: 'neutral' };

export const reviewTone = (state: string): BadgeTone => REVIEW_TONE[state] ?? 'neutral';
export const instructorTone = (status: string): BadgeTone => INSTRUCTOR_TONE[status] ?? 'neutral';
export const publicationTone = (status: string): BadgeTone => PUBLICATION_TONE[status] ?? 'neutral';

/** `?state=` of a review-queue tab link (open is the default, so no parameter). */
export const reviewHref = (tab: ReviewTab): string => (tab === 'open' ? '/learning/reviews' : `/learning/reviews?state=${tab}`);

/** Which actions make sense for a lesson in the queue. */
export function lessonActions(lesson: { status: string; review: { state: string } }): {
  approve: boolean;
  flag: boolean;
  unpublish: boolean;
} {
  return {
    approve: lesson.review.state !== 'approved',
    flag: lesson.review.state !== 'flagged',
    unpublish: lesson.status === 'published',
  };
}

/** A note is valid when its trimmed length is within the limits (`required` = flag / unpublish). */
export function noteError(note: string, required: boolean): 'required' | 'min' | 'max' | null {
  const n = [...note.trim()].length;
  if (n === 0) return required ? 'required' : null;
  if (n < NOTE_MIN) return 'min';
  if (n > NOTE_MAX) return 'max';
  return null;
}

/** «m:ss» / «h:mm:ss» of a media length in seconds (null → null). */
export function formatDuration(seconds: number | null): string | null {
  if (seconds === null || seconds < 0) return null;
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  const pad = (v: number) => String(v).padStart(2, '0');
  return h > 0 ? `${h}:${pad(m)}:${pad(s)}` : `${m}:${pad(s)}`;
}
