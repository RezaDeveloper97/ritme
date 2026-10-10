import { describe, expect, it } from 'vitest';

import type { Conversation } from '@/entities/assistant';
import type { Booking } from '@/entities/doctor';

import { buildFeed } from './feed';

const conv = (id: number, at: string) => ({ id, last_message_at: at, created_at: at }) as Conversation;
const book = (id: number, at: string, status: Booking['status'] = 'completed') => ({ id, starts_at: at, status }) as Booking;

describe('buildFeed', () => {
  const c = [conv(1, '2026-10-01T10:00:00Z'), conv(2, '2026-08-01T10:00:00Z')];
  const b = [book(5, '2026-09-01T10:00:00Z'), book(6, '2026-09-05T10:00:00Z', 'expired')];
  it('merges newest first and drops dead bookings', () => {
    expect(buildFeed(c, b, 'all').map((i) => i.key)).toEqual(['a1', 'v5', 'a2']);
  });
  it('filters by tab', () => {
    expect(buildFeed(c, b, 'assistant').map((i) => i.key)).toEqual(['a1', 'a2']);
    expect(buildFeed(c, b, 'doctor').map((i) => i.key)).toEqual(['v5']);
  });
});
