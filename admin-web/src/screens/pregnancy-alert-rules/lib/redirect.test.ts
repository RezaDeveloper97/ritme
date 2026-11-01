import { describe, expect, it } from 'vitest';

import { INBOX_RULES_HREF } from './redirect';

describe('old pregnancy-alert-rules screen', () => {
  it('forwards to the rules list filtered to the inbox slot and preg. keys', () => {
    const url = new URL(INBOX_RULES_HREF, 'http://x');
    expect(url.pathname).toBe('/messages/rules');
    expect(url.searchParams.get('placement')).toBe('alerts_inbox');
    expect(url.searchParams.get('rule_key')).toBe('preg.');
  });
});
