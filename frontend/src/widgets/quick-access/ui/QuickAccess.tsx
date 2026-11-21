'use client';

import clsx from 'clsx';
import { useTranslations } from 'next-intl';

import { Link } from '@/shared/i18n';
import { pilot } from '@/shared/pilot';
import { openSheet } from '@/shared/sheet';
import { Icon, type IconName, type Tone } from '@/shared/ui';

import { useQuickAccess } from '../model/use-quick-access';
import './qac.css';

/** Registry id of the edit sheet (`app/sheets/registry.tsx`). */
export const QUICK_ACCESS_SHEET = 'quick-access';

function ShortcutFace({ id, icon, tone }: { id: string; icon: IconName; tone: Tone }) {
  const t = useTranslations('nav.quickAccess');
  return (
    <>
      <span className={clsx('qac-ic', `nb-tone-${tone}`)} aria-hidden>
        <Icon name={icon} size={22} strokeWidth={1.8} />
      </span>
      <span className="qac-label">{t(`items.${id}` as 'items.record')}</span>
    </>
  );
}

/**
 * Today «دسترسی سریع» (CB-NAV-03, nbd_Nav_Today): her own shortcuts as round tiles plus «ویرایش», which opens
 * the edit sheet. Only screens that exist are offered (model/shortcuts); picks are stored on this device.
 */
export function QuickAccess({ className }: { className?: string }) {
  const t = useTranslations('nav.quickAccess');
  const qa = useQuickAccess();
  if (!qa.mode) return null;

  return (
    <section className={clsx('qac', className)} aria-labelledby="qac-title">
      <div className="qac-head">
        <h2 id="qac-title" className="nb-sect-title">
          {t('title')}
        </h2>
        <p className="qac-sub">{t('subtitle')}</p>
      </div>
      <ul className="qac-grid">
        {qa.visible.map((s) => (
          <li key={s.id}>
            {s.sheet ? (
              <button type="button" className="qac-tile" onClick={() => openSheet(s.sheet as string)} {...pilot(`home.quick:${s.id}`)}>
                <ShortcutFace id={s.id} icon={s.icon} tone={s.tone} />
              </button>
            ) : (
              <Link href={s.href as string} className="qac-tile" {...pilot(`home.quick:${s.id}`)}>
                <ShortcutFace id={s.id} icon={s.icon} tone={s.tone} />
              </Link>
            )}
          </li>
        ))}
        <li>
          <button type="button" className="qac-tile" onClick={() => openSheet(QUICK_ACCESS_SHEET)} {...pilot('home.quick-edit')}>
            <span className="qac-ic is-edit" aria-hidden>
              <Icon name="plus" size={22} strokeWidth={2} />
            </span>
            <span className="qac-label">{t('edit')}</span>
          </button>
        </li>
      </ul>
    </section>
  );
}
