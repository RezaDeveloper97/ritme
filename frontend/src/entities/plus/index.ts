// Public API of the `plus` entity (Ritme Plus, B-N2-07). Import only from here (CLAUDE.md §3.3).
export {
  fetchPlusHistory,
  fetchPlusPlans,
  fetchPlusQuote,
  fetchPlusStatus,
  fetchPlusTrial,
  plusKeys,
  usePlusHistory,
  usePlusPlans,
  usePlusQuote,
  usePlusStatus,
  usePlusTrial,
} from './api/queries';
export {
  checkoutSchema,
  restoreSchema,
  statusSchema,
  storeIntentSchema,
  storeSyncSchema,
  storeVerificationSchema,
  verificationSchema,
} from './api/schema';
export { planDuration, plansForChannel, storeAvailable, type PlanDuration } from './lib/store';
export { splitSeconds, secondsRemaining, useServerCountdown, type CountdownParts } from './lib/countdown';
export { formatToman, formatTomanThousands, rialsToToman } from './lib/money';
export {
  defaultPlan,
  membershipOf,
  parseGatewayReturn,
  planFromParam,
  remainingShare,
  upgradePlan,
  type GatewayReturn,
  type PlusMembership,
} from './lib/plans';
export {
  detailLocked,
  featureLocked,
  findEntitlement,
  isLongRange,
  isRangeDenial,
  PLUS_ANALYSIS_LONG_RANGE,
  PLUS_COMPANION,
  PLUS_DEEP_ANALYSIS,
  PLUS_SMS_REMINDERS,
  rangeLocked,
} from './lib/gates';
export { PlusDetailLock, PlusRangePaywall, usePlusDetailLocked, usePlusRangeLocked } from './ui/PlusLocks';
export { PlusCrown } from './ui/PlusCrown';
export { entitlementOf, PlusFeatureGate, usePlusLocked } from './ui/PlusFeatureGate';
export type {
  PlusCatalog,
  PlusCheckout,
  PlusDiscountSource,
  PlusEntitlement,
  PlusInvoice,
  PlusOfferPlan,
  PlusPayment,
  PlusPlan,
  PlusPlanRef,
  PlusQuote,
  PlusReceipt,
  PlusRestore,
  PlusStoreIntent,
  PlusStoreSync,
  PlusStoreVerification,
  PlusStatus,
  PlusSubscription,
  PlusTier,
  PlusTrial,
  PlusTrialOffer,
  PlusTrialSheet,
  PlusTrialUsage,
  PlusVerification,
} from './model/types';
