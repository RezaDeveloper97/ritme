'use client';

import { useLocale } from 'next-intl';

import { Icon } from './Icon';

interface NavBackProps {
  onClick: () => void;
  label?: string;
}

/**
 * Direction-aware back button.
 * RTL (fa): chevronRight → points right, sits on the right side of the header.
 * LTR (en): chevronLeft  ← points left,  sits on the left side of the header.
 */
export function NavBack({ onClick, label }: NavBackProps) {
  const locale = useLocale();
  const iconName = locale === 'fa' ? 'chevronRight' : 'chevronLeft';
  return (
    <button className="iconbtn" onClick={onClick} aria-label={label ?? 'برگرد'}>
      <Icon name={iconName} size={24} />
    </button>
  );
}
