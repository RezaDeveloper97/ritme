import { clsx } from 'clsx';

import { Icon } from '@/shared/ui';

/** The amber crown disc of the Plus screens (84px paywall, 120px success). Decorative. */
export function PlusCrown({ size = 'md', className }: { size?: 'md' | 'lg'; className?: string }) {
  return (
    <span className={clsx('plus-crown', size === 'lg' && 'is-lg', className)} aria-hidden>
      <Icon name="crown" size={size === 'lg' ? 56 : 40} strokeWidth={1.8} />
    </span>
  );
}
