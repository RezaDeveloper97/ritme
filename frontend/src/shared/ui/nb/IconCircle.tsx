import { clsx } from 'clsx';

import { Icon, type IconName } from '../Icon';
import { toneClass, type Tone } from './tone';

const PX = { sm: 34, md: 40, lg: 44 } as const;

interface IconCircleProps {
  icon: IconName;
  tone?: Tone;
  /** 34 (list rows) · 40 (settings rows, tiles) · 44 (headers). */
  size?: keyof typeof PX;
  /** Adds the 1px tone ring used on settings rows. */
  outlined?: boolean;
  /** When the icon alone carries meaning; otherwise it is decorative. */
  label?: string;
  className?: string;
}

/** Tone-tinted icon disc (13% fill, optional 33% ring). */
export function IconCircle({ icon, tone = 'brand', size = 'md', outlined, label, className }: IconCircleProps) {
  return (
    <span
      className={clsx('nb-icircle', `is-${size}`, toneClass(tone), outlined && 'is-outlined', className)}
      role={label ? 'img' : undefined}
      aria-label={label}
      aria-hidden={label ? undefined : true}
    >
      <Icon name={icon} size={Math.round(PX[size] * 0.5)} strokeWidth={1.8} />
    </span>
  );
}

interface AvatarProps {
  /** Person's display name — the accessible name and the source of the initial. */
  name: string;
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

/** Circle with the first letter on `--avatar-grad`. */
export function Avatar({ name, size = 'md', className }: AvatarProps) {
  const initial = Array.from(name.trim())[0] ?? '';
  return (
    <span role="img" aria-label={name} className={clsx('nb-avatar', `is-${size}`, className)}>
      <span aria-hidden>{initial}</span>
    </span>
  );
}
