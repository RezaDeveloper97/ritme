import type { ReviewItem } from '../api/safety';

/** The source screen where an item is approved / rejected (the queue only claims). */
export function sourceHref(item: Pick<ReviewItem, 'source' | 'id' | 'ref'>): string {
  switch (item.source) {
    case 'message':
      return `/messages/rules/${item.id}`;
    case 'threshold':
      return '/clinical-config';
    case 'lesson':
      return '/learning/reviews';
    case 'assistant':
      return '/assistant/departments';
    case 'catalog':
      return `/catalog/${encodeURIComponent(item.ref)}`;
  }
}

/** «2 روز» style age split: whole days from 48 h, hours below. */
export function ageParts(hours: number): { unit: 'h' | 'd'; value: number } {
  return hours >= 48 ? { unit: 'd', value: Math.floor(hours / 24) } : { unit: 'h', value: Math.max(0, hours) };
}
