import { clsx } from 'clsx';
import type { ReactNode } from 'react';

import { Icon } from '../Icon';
import { PlusLock, StatusPill } from '../nb';
import type { PlusDenial } from './denial';

/** The amber «پلاس» crown pill that marks a Plus feature (list rows, card headers, tiles). */
export function PlusBadge({ label, className }: { label: ReactNode; className?: string }) {
  return (
    <StatusPill tone="warm" icon="crown" className={clsx('plus-badge', className)}>
      {label}
    </StatusPill>
  );
}

interface PlusGateProps {
  /** The feature is not in the user's tier (e.g. its `/plus/status` entitlement has `allowed: false`). */
  locked?: boolean;
  /** What the API answered last time (`plusDenialOf(error)`); wins over `locked`. */
  denial?: PlusDenial | null;
  /** Text of the lock pill, «پلاس». */
  label: ReactNode;
  /** Screen-reader explanation of the lock, «این بخش در ریتمی پلاس باز می‌شود». */
  lockedText: string;
  /** Opens the paywall; the locked card becomes one button. Not offered for `exhausted` (upgrading can't help). */
  onUnlock?: () => void;
  /** The note shown when a Plus user's monthly quota ran out (429), e.g. «سهمیهٔ این ماه تمام شد…». */
  exhaustedText?: ReactNode;
  /** The feature itself — rendered as-is when open, as a blurred teaser when locked. */
  children: ReactNode;
  className?: string;
}

/**
 * Plus gating as UI (B-N2-08). A pure primitive: it knows nothing about the
 * plus entity or routing — callers pass the state (`locked` from the status
 * entitlements, or a `denial` mapped from a 402/429 by `plusDenialOf`), the
 * copy and the unlock action. `entities/plus` has the wired wrapper
 * (`PlusFeatureGate`).
 *
 * - open → `children` unchanged;
 * - locked / free quota spent (402) → the blurred teaser behind the «پلاس» lock;
 * - Plus quota spent (429) → `children` with a renewal note above (no paywall).
 */
export function PlusGate({ locked, denial, label, lockedText, onUnlock, exhaustedText, children, className }: PlusGateProps) {
  if (denial?.kind === 'exhausted') {
    return (
      <div className={clsx('plus-gate', className)}>
        {exhaustedText ? (
          <p className="plus-gate-note" role="status">
            <Icon name="clock" size={16} strokeWidth={2} />
            <span>{exhaustedText}</span>
          </p>
        ) : null}
        {children}
      </div>
    );
  }
  if (denial || locked) {
    return (
      <PlusLock label={label} lockedText={lockedText} onUnlock={onUnlock} className={clsx('plus-gate', className)}>
        {children}
      </PlusLock>
    );
  }
  return <>{children}</>;
}
