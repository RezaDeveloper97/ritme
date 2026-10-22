import { describe, expect, it } from 'vitest';

import { consentHistorySchema } from './history';

describe('consent history (T-M10-23)', () => {
  it('maps the ledger events, newest first as sent', () => {
    const events = consentHistorySchema.parse({
      events: [
        {
          id: 3,
          consent: 'ai_assistant',
          title: 'دستیار',
          version: 1,
          action: 'revoke',
          source: 'settings',
          scope: null,
          created_at: '2026-10-22T10:00:00+03:30',
        },
        {
          id: 1,
          consent: 'telemed_share',
          title: null,
          version: 1,
          action: 'grant',
          source: 'telemed',
          scope: { booking_id: 7, items: ['cycle'] },
          created_at: null,
        },
      ],
    });
    expect(events).toEqual([
      {
        id: 3,
        consent: 'ai_assistant',
        title: 'دستیار',
        action: 'revoke',
        source: 'settings',
        createdAt: '2026-10-22T10:00:00+03:30',
      },
      { id: 1, consent: 'telemed_share', title: null, action: 'grant', source: 'telemed', createdAt: null },
    ]);
  });

  it('reads an unknown source as other and keeps the scope out of the client model', () => {
    const [e] = consentHistorySchema.parse({
      events: [
        { id: 1, consent: 'x', title: 'X', version: 0, action: 'grant', source: 'future', scope: {}, created_at: null },
      ],
    });
    expect(e?.source).toBe('other');
    expect(e && 'scope' in e).toBe(false);
  });

  it('rejects an unknown action', () => {
    expect(() =>
      consentHistorySchema.parse({
        events: [{ id: 1, consent: 'x', title: null, action: 'update', source: 'settings', created_at: null }],
      }),
    ).toThrow();
  });
});
