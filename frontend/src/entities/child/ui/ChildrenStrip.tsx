'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';

import { Link } from '@/shared/i18n';
import { Icon } from '@/shared/ui';

import type { Child } from '../model/types';
import { ChildAvatar } from './ChildAvatar';

interface ChildrenStripProps {
  items: readonly Pick<Child, 'id' | 'name' | 'initial' | 'sex'>[];
  /** Show the «+» add link (owners under the limit). */
  canAdd: boolean;
  className?: string;
}

/**
 * The «فرزندان» row of the postpartum hero (nbl_v15_Main): one avatar + name
 * per child linking to its home, and «+» to add one. Renders nothing for no
 * children and no add.
 */
export function ChildrenStrip({ items, canAdd, className }: ChildrenStripProps) {
  const t = useTranslations('children');
  if (items.length === 0 && !canAdd) return null;
  return (
    <nav className={clsx('chd-strip', className)} aria-label={t('strip.title')}>
      <span className="chd-strip-title">{t('strip.title')}</span>
      <ul className="chd-strip-list">
        {items.map((c) => (
          <li key={c.id}>
            <Link href={`/children/${c.id}`} className="chd-strip-item">
              <ChildAvatar id={c.id} name={c.name} initial={c.initial} sex={c.sex} size={36} />
              <span className="chd-strip-name">{c.name}</span>
            </Link>
          </li>
        ))}
      </ul>
      {canAdd ? (
        <Link href="/children/new" className="chd-strip-add" aria-label={t('strip.add')}>
          <Icon name="plus" size={18} strokeWidth={2.2} />
        </Link>
      ) : null}
    </nav>
  );
}
