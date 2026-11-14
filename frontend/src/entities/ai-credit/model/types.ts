/**
 * AI credits («اعتبار») domain shapes (docs/ai-credits.md), camelCased at the boundary.
 * The user only ever sees credits — never tokens.
 */

export type CreditPeriod = 'plus' | 'trial' | null;

export interface CreditFeature {
  key: string;
  title: string;
  creditCost: number;
  available: boolean;
  affordable: boolean;
}

export interface AiCredits {
  /** A running subscription (Plus or trial) exists, so AI works. */
  canUse: boolean;
  period: CreditPeriod;
  /** When the period's credits are gone; null without a subscription. */
  periodEndsAt: string | null;
  plan: { total: number; used: number; remaining: number };
  /** Extra credits never expire; `locked` = kept, but there is no subscription to spend them under. */
  extra: { remaining: number; locked: boolean };
  spendable: number;
  features: CreditFeature[];
}

export interface CreditUsageByFeature {
  feature: string;
  title: string;
  uses: number;
  credits: number;
}

export interface CreditUsageByDay {
  /** API date `YYYY-MM-DD`. */
  date: string;
  credits: number;
}

export interface CreditHistoryItem {
  id: number;
  feature: string;
  title: string;
  credits: number;
  at: string;
}

export interface AiUsagePage {
  days: number;
  since: string | null;
  total: number;
  byFeature: CreditUsageByFeature[];
  byDay: CreditUsageByDay[];
  history: { items: CreditHistoryItem[]; next: number | null };
}

export interface CreditPack {
  id: number;
  code: string;
  title: string;
  badge: string | null;
  credits: number;
  /** Rials. */
  price: number;
  isHighlighted: boolean;
}

export interface CreditPackCatalog {
  currency: string;
  vatRateBps: number;
  packs: CreditPack[];
}

export interface CreditPackQuote {
  pack: CreditPack;
  currency: string;
  subtotal: number;
  vatRateBps: number;
  vat: number;
  total: number;
}

/**
 * The slice of `useAiGate` a surface FEATURE needs. A feature cannot import the `ai-gate` feature, so the screen
 * that composes both passes `useAiGate(key)` down as this prop.
 */
export interface AiGateHooks {
  /** Before starting an AI action: false = the gate sheet opened, do not start. */
  guard: () => boolean;
  /** In every catch / onError of an AI request: true = the gate sheet took it, show nothing else. */
  handleError: (error: unknown) => boolean;
  /** After every successful AI request. */
  refresh: () => void;
}
