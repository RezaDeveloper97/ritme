import type { ReactNode } from 'react';
import { clsx } from 'clsx';

import { Icon, type IconName } from '../Icon';
import { toneClass, type Tone } from './tone';

interface StatusPillProps {
  tone?: Tone;
  icon?: IconName;
  /** `solid` = the accent fill with `--on-brand` (only for `brand`, e.g. «محبوب»). */
  solid?: boolean;
  className?: string;
  children: ReactNode;
}

/** Radius-999 verdict pill (منظم / طبیعی / پلاس), 11/800, tinted bg + deep text. */
export function StatusPill({ tone = 'brand', icon, solid, className, children }: StatusPillProps) {
  return (
    <span className={clsx('nb-pill', toneClass(tone), solid && 'is-solid', className)}>
      {icon ? <Icon name={icon} size={12} strokeWidth={2.4} /> : null}
      {children}
    </span>
  );
}

interface PlusLockProps {
  /** Text of the pill, «پلاس». */
  label: ReactNode;
  /** Screen-reader explanation of what is hidden, e.g. «این بخش در ریتمی پلاس باز می‌شود». */
  lockedText: string;
  /** Opens the paywall; renders the whole card as one button when given. */
  onUnlock?: () => void;
  /** The teaser content — blurred and hidden from assistive tech. */
  children: ReactNode;
  className?: string;
}

/**
 * A card body behind the «پلاس» lock: blurred teaser (hidden from assistive
 * tech) + warm lock pill. With `onUnlock` a full-card button opens the paywall.
 */
export function PlusLock({ label, lockedText, onUnlock, children, className }: PlusLockProps) {
  return (
    <div className={clsx('nb-lock', className)}>
      <div className="nb-lock-body" aria-hidden inert>
        {children}
      </div>
      <span className="nb-lock-pill">
        <StatusPill tone="warm" icon="lock">
          {label}
        </StatusPill>
      </span>
      {onUnlock ? (
        <button type="button" className="nb-lock-hit" aria-label={lockedText} onClick={onUnlock} />
      ) : (
        <span className="sr-only">{lockedText}</span>
      )}
    </div>
  );
}
