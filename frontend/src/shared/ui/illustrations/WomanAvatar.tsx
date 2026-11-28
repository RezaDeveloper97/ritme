import clsx from 'clsx';

import { type IllustrationProps, svgA11y } from './svg-a11y';
import '@/shared/ui/styles/wav.css';

/**
 * The default profile picture: a flat woman bust, drawn to fill a round frame
 * (the caller's container clips it). Shown instead of the name's first letter
 * while profile photos are not supported. Fills are `.wav-*` classes on the
 * `--avatar-*` tokens, so it flips with the theme.
 */
export function WomanAvatar({ size = 96, label, className }: Omit<IllustrationProps, 'onHero'>) {
  return (
    <svg width={size} height={size} viewBox="0 0 100 100" className={clsx('wav', className)} {...svgA11y(label)}>
      <path
        className="wav-hair"
        d="M50 14C30 14 22 30 23 48C24 62 20 74 17 84L83 84C80 74 76 62 77 48C78 30 70 14 50 14Z"
      />
      <rect className="wav-skin" x="43" y="56" width="14" height="20" rx="6" />
      <path className="wav-top" d="M14 104C14 84 30 74 50 74C70 74 86 84 86 104Z" />
      <ellipse className="wav-skin" cx="50" cy="45" rx="17" ry="19" />
      <path className="wav-hair" d="M32 46C31 29 40 21 51 21C62 21 70 29 68 44C60 41 53 35 50 28C47 37 40 43 32 46Z" />
      <circle className="wav-cheek" cx="40" cy="52" r="3.2" />
      <circle className="wav-cheek" cx="60" cy="52" r="3.2" />
      <circle className="wav-eye" cx="43" cy="46.5" r="1.7" />
      <circle className="wav-eye" cx="57" cy="46.5" r="1.7" />
      <path className="wav-smile" d="M46 55.5Q50 58.5 54 55.5" />
    </svg>
  );
}
