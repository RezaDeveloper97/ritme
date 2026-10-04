import { describe, expect, it } from 'vitest';

import { applyConsent, consentChangeBody, consentsSchema } from './consents';

describe('consentsSchema', () => {
  it('keeps known codes in server order and drops unknown ones', () => {
    const parsed = consentsSchema.parse({
      consents: [
        { code: 'anonymous_stats', granted: true, granted_at: '2026-09-23T10:00:00+03:30', revoked_at: null },
        { code: 'future_thing', granted: true, granted_at: null, revoked_at: null },
        { code: 'ai_lab_analysis', granted: false, granted_at: null, revoked_at: null },
      ],
    });
    expect(parsed.map((c) => c.code)).toEqual(['anonymous_stats', 'ai_lab_analysis']);
    expect(parsed[0]?.grantedAt).toBe('2026-09-23T10:00:00+03:30');
  });

  it('applyConsent flips one code and keeps the last grant time on withdrawal', () => {
    const list = consentsSchema.parse({
      consents: [{ code: 'ai_lab_analysis', granted: true, granted_at: '2026-09-01T09:00:00+03:30' }],
    });
    const off = applyConsent(list, 'ai_lab_analysis', false, '2026-10-01T09:00:00+03:30');
    expect(off[0]).toEqual({
      code: 'ai_lab_analysis',
      granted: false,
      grantedAt: '2026-09-01T09:00:00+03:30',
      version: 1,
    });
  });

  it('reads the version in force and sends it back with a grant (B-N6-05b)', () => {
    const [c] = consentsSchema.parse({
      consents: [{ code: 'assistant_profile', granted: false, granted_at: null, version: 2, accepted_version: 1 }],
    });
    expect(c?.version).toBe(2);
    expect(consentChangeBody('assistant_profile', true, 2)).toEqual({
      consents: { assistant_profile: true },
      versions: { assistant_profile: 2 },
    });
    expect(consentChangeBody('assistant_profile', false, 2)).toEqual({ consents: { assistant_profile: false } });
  });
});
