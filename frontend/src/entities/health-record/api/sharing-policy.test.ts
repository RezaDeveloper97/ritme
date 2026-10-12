import { describe, expect, it } from 'vitest';

import { shareLinksSchema, wantsLosses } from './report';
import { recordSharingSchema } from './sharing';
import { OPEN_SHARING_POLICY } from './sharing-policy';

// User decisions 2026-10-12: the per-share loss opt-in and the teen sharing policy.

const overview = {
  doctor: {
    available: true,
    ttl_hours: 24,
    active_count: 1,
    max_active: 5,
    items: [
      {
        id: 1,
        label: null,
        status: 'active',
        sections: ['basics', 'pregnancies'],
        include_losses: true,
        range: { from: '2026-04-12', to: '2026-10-12' },
        created_at: '2026-10-12T09:00:00+03:30',
        expires_at: '2026-10-13T09:00:00+03:30',
        revoked_at: null,
        views: 0,
        last_viewed_at: null,
      },
    ],
  },
  family: [],
  access_log: [],
};

describe('sharing policy', () => {
  it('reads the teen policy and the per-code loss flag', () => {
    const d = recordSharingSchema.parse({
      ...overview,
      policy: { teen: true, parent_linked: false, doctor_sharing: false, emergency_card: true },
    });
    expect(d.policy).toEqual({ teen: true, parentLinked: false, doctorSharing: false });
    expect(d.doctor.items[0]?.includeLosses).toBe(true);
  });

  it('falls back to an open policy when the server sends none', () => {
    expect(recordSharingSchema.parse(overview).policy).toEqual(OPEN_SHARING_POLICY);
    const links = shareLinksSchema.parse({ items: [], active_count: 0, max_active: 10, ttl_days: 7 });
    expect(links.policy.doctorSharing).toBe(true);
  });

  it('sends the loss opt-in only with the pregnancies section', () => {
    expect(wantsLosses({ sections: ['basics'], includeLosses: true })).toBe(false);
    expect(wantsLosses({ sections: ['pregnancies'] })).toBe(false);
    expect(wantsLosses({ sections: ['pregnancies'], includeLosses: true })).toBe(true);
  });
});
