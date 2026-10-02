import { describe, expect, it } from 'vitest';

import { withKitTick } from './queries';
import { teenKitSchema, teenProfileStateSchema, teenTodaySchema } from './schema';

// Trimmed from backend-go/contract/golden/teen/onboarding_kit_note.en.json and today_no_profile.en.json.
const kit = {
  items: [
    { code: 'pads', title: '2 sanitary pads', body: null, needs_review: true, checked: true },
    { code: 'wipes', title: 'Wet wipes', body: null, needs_review: true, checked: false },
  ],
  checked_count: 1,
  total: 2,
  ready: false,
};
const allowsOff = { shop: false, banners: false, ads: false, plus_upsell: false, commercial_recommendations: false };

describe('teen schemas', () => {
  it('parses the teen home read model', () => {
    const today = teenTodaySchema.parse({
      profile: { age_band: '13_15', menarche: 'not_yet', parent_note: null, updated_at: null },
      needs_onboarding: false,
      is_teen_mode: true,
      allows: allowsOff,
      readiness: {
        code: 'estimate_talk',
        title: 'Talk',
        body: 'b',
        meta: { kind: 'estimate', severity: 'caution' },
        needs_review: true,
      },
      signs: [{ code: 'approaching_signs', title: 'Signs', body: 'b', meta: { kind: 'sign' }, needs_review: true }, 7],
      talk_note: { code: 'when_to_talk', title: 'When', body: 'b', meta: { kind: 'talk' }, needs_review: true },
      kit,
      faq: [{ code: 'irregular', title: 'Q', body: 'A', meta: null, needs_review: true }],
      parent_preview: { next_period_week: 'unknown', kit_ready: false, note: null },
      parent_links: [
        {
          id: 3,
          status: 'invited',
          display_name: null,
          accepted_at: null,
          grants: { teen_period_week: 'view', teen_kit: 'none', teen_notes: 'none' },
        },
      ],
    });
    expect(today.profile).toEqual({ ageBand: '13_15', menarche: 'not_yet', parentNote: null });
    expect(today.readiness?.severity).toBe('caution');
    expect(today.signs).toHaveLength(1);
    expect(today.kit.checkedCount).toBe(1);
    expect(today.parentLinks[0]).toMatchObject({ id: 3, status: 'invited', grants: { teenPeriodWeek: 'view' } });
  });

  it('reads a missing profile as needing onboarding and never opens commerce on bad flags', () => {
    const state = teenProfileStateSchema.parse({
      profile: null,
      needs_onboarding: true,
      is_teen_mode: false,
      allows: 'nonsense',
    });
    expect(state.needsOnboarding).toBe(true);
    expect(state.allows).toEqual({ shop: false, banners: false, ads: false, plusUpsell: false });
  });

  it('applies an optimistic tick', () => {
    const parsed = teenKitSchema.parse(kit);
    const next = withKitTick(parsed, { code: 'wipes', checked: true });
    expect(next).toMatchObject({ checkedCount: 2, ready: true });
    expect(withKitTick(next, { code: 'pads', checked: false }).ready).toBe(false);
  });
});
