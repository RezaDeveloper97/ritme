/**
 * Ritme Plus domain shapes (B-N2-04 API, camelCased at the boundary).
 *
 * Every amount is an integer number of **rials** (`currency: IRR`); the UI
 * shows toman (÷10) through `formatToman`. Dates are ISO-8601 strings from the
 * API, formatted only through `shared/lib/date`.
 */

export type PlusTier = 'free' | 'trial' | 'plus';

export interface PlusPlan {
  id: number;
  code: string;
  /** Localized by the API («۳ ماهه»). */
  title: string;
  /** «محبوب‌ترین» — localized, null when the plan has none. */
  badge: string | null;
  durationMonths: number;
  /** Full price of the plan, rials. */
  price: number;
  /** Price per month, rials. */
  monthlyPrice: number;
  savingsPercent: number;
  isHighlighted: boolean;
}

export interface PlusCatalog {
  currency: string;
  vatRateBps: number;
  trialDays: number;
  plans: PlusPlan[];
}

/** The short plan object on subscriptions, quotes and invoices (null when the plan was deleted). */
export interface PlusPlanRef {
  id: number;
  code: string;
  title: string;
  durationMonths: number;
}

export interface PlusEntitlement {
  key: string;
  allowed: boolean;
  unlimited: boolean;
  limit: number | null;
  used: number;
  remaining: number | null;
}

export interface PlusSubscription {
  id: number;
  plan: PlusPlanRef | null;
  status: string;
  source: string;
  startsAt: string;
  endsAt: string;
  daysLeft: number;
  autoRenew: boolean;
  canceledAt: string | null;
}

export interface PlusTrial {
  startedAt: string;
  endsAt: string;
  isActive: boolean;
  daysLeft: number;
}

export interface PlusStatus {
  tier: PlusTier;
  isPlus: boolean;
  subscription: PlusSubscription | null;
  trial: PlusTrial | null;
  trialAvailable: boolean;
  periodStart: string;
  resetsAt: string;
  entitlements: PlusEntitlement[];
}

/** Where a discount came from: a typed code or the trial offer (B-N2-06); null = none. */
export type PlusDiscountSource = 'code' | 'trial_offer' | null;

export interface PlusQuote {
  plan: PlusPlanRef | null;
  currency: string;
  subtotal: number;
  discount: number;
  discountCode: string | null;
  discountSource: PlusDiscountSource;
  vatRateBps: number;
  vat: number;
  total: number;
}

export interface PlusReceipt {
  refId: string;
  cardPan: string | null;
  amount: number;
  paidAt: string;
}

export interface PlusInvoice {
  reference: string;
  /** `pending` | `paid` | `failed` | `expired` … (server-owned). */
  status: string;
  plan: PlusPlanRef | null;
  durationMonths: number;
  currency: string;
  subtotal: number;
  discount: number;
  discountCode: string | null;
  vatRateBps: number;
  vat: number;
  total: number;
  gateway: string | null;
  createdAt: string | null;
  expiresAt: string | null;
  paidAt: string | null;
  receipt: PlusReceipt | null;
}

export interface PlusPayment {
  gateway: string;
  authority: string;
  /** The gateway page to send the browser to. */
  redirectUrl: string;
}

export interface PlusCheckout {
  invoice: PlusInvoice;
  /** Null when nothing is charged (100% discount): the invoice is already paid. */
  payment: PlusPayment | null;
}

export interface PlusVerification {
  invoice: PlusInvoice;
  alreadyVerified: boolean;
  status: PlusStatus;
}

export interface PlusRestore {
  restored: number;
  status: PlusStatus;
}
