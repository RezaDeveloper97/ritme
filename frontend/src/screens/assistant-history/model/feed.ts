import type { Conversation } from '@/entities/assistant';
import type { Booking } from '@/entities/doctor';

export const HISTORY_TABS = ['all', 'assistant', 'doctor'] as const;
export type HistoryTab = (typeof HISTORY_TABS)[number];

export type FeedItem =
  | { kind: 'assistant'; key: string; at: number; conversation: Conversation }
  | { kind: 'visit'; key: string; at: number; booking: Booking };

const ts = (iso: string | null): number => {
  const v = iso ? Date.parse(iso) : NaN;
  return Number.isNaN(v) ? 0 : v;
};

/** Visits that never happened (a hold that expired, a failed payment) are not history. */
const isRealVisit = (b: Booking) => b.status === 'confirmed' || b.status === 'completed' || b.status === 'cancelled';

/** One chronological (newest first) timeline of assistant chats and telemed visits, composed on the client. */
export function buildFeed(conversations: readonly Conversation[], bookings: readonly Booking[], tab: HistoryTab): FeedItem[] {
  const items: FeedItem[] = [];
  if (tab !== 'doctor')
    for (const c of conversations) items.push({ kind: 'assistant', key: `a${c.id}`, at: ts(c.last_message_at ?? c.created_at), conversation: c });
  if (tab !== 'assistant')
    for (const b of bookings) if (isRealVisit(b)) items.push({ kind: 'visit', key: `v${b.id}`, at: ts(b.starts_at), booking: b });
  return items.sort((x, y) => y.at - x.at);
}
