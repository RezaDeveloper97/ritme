import { describe, expect, it } from 'vitest';

import { INBOX_SLOT, inboxRulesHref, matchesPrefix } from './inbox';

describe('inbox rules filter', () => {
  it('builds the filtered list url', () => {
    expect(inboxRulesHref()).toBe(`/messages/rules?placement=${INBOX_SLOT}&rule_key=preg.`);
    expect(inboxRulesHref('')).toBe('/messages/rules?placement=alerts_inbox');
  });
  it('filters rows by rule_key prefix', () => {
    const rows = [{ rule_key: 'preg.bp' }, { rule_key: 'meno.hot' }, { rule_key: null }];
    expect(matchesPrefix(rows, 'preg.')).toHaveLength(1);
    expect(matchesPrefix(rows, '')).toHaveLength(3);
  });
});
