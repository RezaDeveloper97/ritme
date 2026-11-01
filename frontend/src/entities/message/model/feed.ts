/** The slots of GET /messages/feed the home reads (SPEC §5.4). */
export const FEED_SLOTS = [
  'urgent_modal',
  'status_banner',
  'smart_tip',
  'daily_advice',
  'daily_challenge',
  'modal',
] as const;
export type FeedSlot = (typeof FEED_SLOTS)[number];

/** «What we saw / how sure we are / who to call» of the follow-up and urgent levels. */
export interface FeedEvidence {
  seen: string;
  confidence: string;
  /** Urgent level only: the call button (115 or the doctor's number). */
  contact: string;
}

/** One message of a slot, text already rendered and localized by the server. */
export interface FeedItem {
  slot: FeedSlot;
  /** The message rule's id (not the alerts-inbox row id). */
  id: number;
  /** The alerts-inbox row id for `POST /alerts/{id}/ack`; present only on items fanned out from the inbox. */
  alertId: number | null;
  ruleKey: string | null;
  version: number;
  variant: string;
  kind: string;
  level: string;
  title: string;
  body: string;
  ctaText: string | null;
  ctaDeeplink: string | null;
  dismissible: boolean;
  evidence: FeedEvidence | null;
}

/** `daily_challenge`: the server's pick for today (T-M10-10). */
export interface FeedChallenge {
  id: number;
  title: string;
  description: string | null;
  category: string | null;
  durationDays: number;
  /** 1-based day of an ongoing run. */
  day: number | null;
  reason: string;
}

export interface MessageFeed {
  slots: Record<FeedSlot, FeedItem[]>;
  alerts: { open: number; urgent: number };
  rulesRev: number;
  /** null = no pick (the home keeps the legacy card). */
  dailyChallenge: FeedChallenge | null;
}

/** The events a client may post. `shown` is logged by the server and never sent (M-19). */
export type MessageEventName = 'clicked' | 'dismissed' | 'goal';
