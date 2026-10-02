'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { Icon } from '@/shared/ui';

export type PersonTone = 'self' | 'companion' | 'child';

/** First grapheme of a name for the round initial (falls back to a person glyph). */
function initialOf(name: string | null): string {
  return Array.from((name ?? '').trim())[0] ?? '';
}

interface PersonBubbleProps {
  name: string | null;
  tone: PersonTone;
  size?: 'md' | 'lg';
  className?: string;
}

/**
 * The tinted initial circle of the Hamdam artboards: rose for «تو», violet for
 * the companion, bloom for a child (44px; 54px on Hamdam_Done). Decorative —
 * the name is always printed next to it.
 */
export function PersonBubble({ name, tone, size = 'md', className }: PersonBubbleProps) {
  const initial = initialOf(name);
  return (
    <span className={clsx('cmp-bubble', `is-${tone}`, size === 'lg' && 'is-lg', className)} aria-hidden>
      {initial || <Icon name="user" size={size === 'lg' ? 24 : 20} />}
    </span>
  );
}

interface FamilyStripProps {
  /** The owner's own name, for her initial; the label is always «تو». */
  selfName: string | null;
  companionName: string;
  /** The line under the strip (children note / pending note). */
  note?: ReactNode;
  size?: 'md' | 'lg';
  className?: string;
}

/** «خانواده شما»: you + spouse (+ shared children once B-N5 lands) as initials with names. */
export function FamilyStrip({ selfName, companionName, note, size = 'md', className }: FamilyStripProps) {
  const t = useTranslations('companions.family');
  return (
    <section className={clsx('nb-card cmp-family', size === 'lg' && 'is-lg', className)} aria-labelledby="cmp-family-title">
      <h2 id="cmp-family-title" className="cmp-family-title">
        <Icon name="home" size={18} className="cmp-family-icon" />
        {t('title')}
      </h2>
      <ul className="cmp-family-row">
        <li className="cmp-family-member">
          <PersonBubble name={selfName} tone="self" size={size} />
          <span className="cmp-family-name">{t('you')}</span>
        </li>
        <li className="cmp-family-member">
          <PersonBubble name={companionName} tone="companion" size={size} />
          <span className="cmp-family-name">{companionName}</span>
        </li>
      </ul>
      {note ? <p className="cmp-family-note">{note}</p> : null}
    </section>
  );
}
