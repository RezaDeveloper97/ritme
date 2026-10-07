import { describe, expect, it } from 'vitest';

import { discreetKeys, discreetSchema } from './discreet';

describe('discreet notifications (CB-PRIV-01)', () => {
  it('reads neutral_copy from the notification-settings body', () => {
    expect(discreetSchema.parse({ neutral_copy: false, groups: [], quiet_hours: {} })).toBe(false);
    expect(discreetSchema.parse({ neutral_copy: true })).toBe(true);
  });

  it('defaults to on, like the server', () => {
    expect(discreetSchema.parse({})).toBe(true);
  });

  it('lives under the notification-settings key so both screens stay in sync', () => {
    expect(discreetKeys.flag()[0]).toBe('notification-settings');
  });
});
