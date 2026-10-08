'use client';

import { useTranslations } from 'next-intl';

import { useChildren } from '@/entities/child';
import { useDirection, useRouter } from '@/shared/i18n';
import { openSheet } from '@/shared/sheet';
import { Icon } from '@/shared/ui';

/** M3 appointment form (`/reminders/appointment/new`) — «نوبت یا ویزیت» plans one for later. */
const APPOINTMENT_NEW = '/reminders/appointment/new';
/** B-N6-08 to-do add sheet (`app/sheets/registry.tsx`). */
const TODO_ADD_SHEET = 'todo-add';

/**
 * Bottom of the + sheet (CB-NAV-03, nbd_Nav_Plus): «برای بعد» — an appointment (→ the M3 form) or a new task
 * (→ the B-N6-08 to-do sheet, stacked over this one) — and «ثبت برای <child>» (feeding / sleep / diapers,
 * B-N5-07) for her first own child when she has children. The postpartum preset already lists the baby rows,
 * so `childRow={false}` there.
 */
export function LogLater({ childRow = true }: { childRow?: boolean }) {
  const t = useTranslations('nav.later');
  const router = useRouter();
  const rtl = useDirection() === 'rtl';
  const kids = useChildren({ enabled: childRow });
  const child = childRow ? kids.data?.children.find((c) => c.role === 'owner') : undefined;

  // Navigating closes every open sheet (shared/sheet closeSheetOnRouteChange) — no closeSheet() race with history.back().
  const go = (href: string) => router.push(href);

  return (
    <section className="llt" aria-labelledby="llt-title">
      <h3 id="llt-title" className="lday-sec-title">
        {t('title')}
      </h3>
      <div className="llt-pair">
        <button type="button" className="llt-btn" onClick={() => go(APPOINTMENT_NEW)}>
          <Icon name="calendar" size={18} strokeWidth={1.8} className="llt-ic" />
          <span>{t('appointment')}</span>
        </button>
        <button type="button" className="llt-btn" onClick={() => openSheet(TODO_ADD_SHEET)}>
          <Icon name="todo" size={18} strokeWidth={1.8} className="llt-ic" />
          <span>{t('task')}</span>
        </button>
      </div>
      {child ? (
        <button type="button" className="llt-row" onClick={() => go(`/children/${child.id}/feeding`)}>
          <span className="nb-avatar is-md llt-avatar" aria-hidden>
            <span>{Array.from(child.name.trim())[0] ?? ''}</span>
          </span>
          <span className="llt-row-text">
            <span className="llt-row-title">{t('child', { name: child.name })}</span>
            <span className="llt-row-sub">{t('childSub')}</span>
          </span>
          <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} className="llt-chev" />
        </button>
      ) : null}
    </section>
  );
}
