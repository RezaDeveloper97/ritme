/** The four message levels, most important first (SPEC §5.4.1). */
export const ALERT_LEVELS = ['urgent', 'follow_up', 'suggestion', 'info'] as const;
export type AlertLevel = (typeof ALERT_LEVELS)[number];

export const ALERT_STATES = ['active', 'acknowledged', 'added_to_visit_note', 'resolved', 'archived'] as const;
export type AlertState = (typeof ALERT_STATES)[number];

/** `open` = active / acknowledged / added to visit note; `history` = resolved / archived. */
export type AlertListState = 'open' | 'history';

export interface AlertEvidence {
  /** «چی دیدیم». */
  seen: string;
  /** «چقدر مطمئنیم». */
  confidence: string;
  /** Only on urgent items: who to call (a phone number such as 115, or free text). */
  contact: string | null;
}

/** One alerts-inbox row (GET /alerts). Health copy: display only, never logged (§11). */
export interface AlertItem {
  id: number;
  level: AlertLevel;
  state: AlertState;
  title: string;
  body: string;
  ctaText: string | null;
  ctaDeeplink: string | null;
  evidence: AlertEvidence | null;
  hitCount: number;
  /** Server verdict: false for an urgent item until it is acknowledged. */
  dismissible: boolean;
  lastAt: string | null;
  ackedAt: string | null;
}

export interface AlertPage {
  items: AlertItem[];
  nextCursor: string | null;
}
