// Plus gating primitives (B-N2-08). Pure UI — no entity imports; data comes in via props.
// The wired wrapper (status entitlements + paywall link + copy) is `PlusFeatureGate` in `@/entities/plus`.
export {
  PLUS_QUOTA_EXCEEDED,
  PLUS_REQUIRED,
  plusDenialOf,
  upgradeHelps,
  type PlusDenial,
  type PlusDenialKind,
} from './denial';
export { PlusBadge, PlusGate } from './PlusGate';
