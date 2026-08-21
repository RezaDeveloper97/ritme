'use client';

import { useDirection } from '@/shared/i18n';

import { Icon } from './Icon';

interface NavBackProps {
  onClick: () => void;
  label?: string;
}

/**
 * Direction-aware back button.
 * RTL: chevronRight → points right, sits on the right side of the header.
 * LTR: chevronLeft  ← points left,  sits on the left side of the header.
 *
 * Keyed off the active language's direction, not off "is it Persian" — any RTL
 * language an admin adds (CLAUDE.md §6) must point the same way.
 */
export function NavBack({ onClick, label }: NavBackProps) {
  const iconName = useDirection() === 'rtl' ? 'chevronRight' : 'chevronLeft';
  return (
    <button className="iconbtn" onClick={onClick} aria-label={label ?? 'برگرد'}>
      <Icon name={iconName} size={24} />
    </button>
  );
}
