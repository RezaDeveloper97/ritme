/**
 * The pregnancy alert rules moved to the message engine (T-M10-08): the list lives in the rules screen, filtered to the
 * alerts_inbox slot and the `preg.` rule keys. The old list screen only forwards there.
 */
export const INBOX_RULES_HREF = '/messages/rules?placement=alerts_inbox&rule_key=preg.';
