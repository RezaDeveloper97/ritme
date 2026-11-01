/** Inbox rules (T-M10-08): the alerts_inbox slot and the rule_key families that live in it. */
export const INBOX_SLOT = 'alerts_inbox';
export const RULE_KEY_PREFIXES = ['preg.', 'meno.', 'cond.'] as const;

/** The rules-list URL the old pregnancy-alert-rules screen redirects to. */
export function inboxRulesHref(prefix: string = RULE_KEY_PREFIXES[0]): string {
  const q = new URLSearchParams({ placement: INBOX_SLOT });
  if (prefix) q.set('rule_key', prefix);
  return `/messages/rules?${q.toString()}`;
}

/** Client-side `rule_key` prefix filter (the list API has none); an empty prefix keeps everything. */
export function matchesPrefix<T extends { rule_key: string | null }>(rows: readonly T[], prefix: string): T[] {
  return prefix === '' ? [...rows] : rows.filter((r) => r.rule_key?.startsWith(prefix));
}
