'use client';

import { useTranslations } from 'next-intl';

import { useModuleEnabled } from '@/entities/app-module';
import { useChildren } from '@/entities/child';
import { useDirection, useRouter } from '@/shared/i18n';
import { pilot } from '@/shared/pilot';
import { openSheet } from '@/shared/sheet';
import { Icon, IconCircle } from '@/shared/ui';
import '@/shared/ui/styles/lday.css';
import './llt.css';

/** M3 appointment form (`/reminders/appointment/new`) — «نوبت یا ویزیت» plans one for later. */
const APPOINTMENT_NEW = '/reminders/appointment/new';
/** B-N6-08 to-do add sheet (`app/sheets/registry.tsx`). */
const TODO_ADD_SHEET = 'todo-add';
/** Vitals chooser sheet (`screens/vitals-chooser`): BP / heart rate / glucose add forms + the hub. */
const VITALS_SHEET = 'vitals-add';

/**
 * Bottom of the + sheet (CB-NAV-03, nbd_Nav_Plus): «علائم حیاتی» (→ the vitals chooser sheet), «برای بعد» — an appointment (→ the M3 form) or a new task
 * (→ the B-N6-08 to-do sheet, stacked over this one) — and «ثبت برای <child>» (feeding / sleep / diapers,
 * B-N5-07) for her first own child when she has children. The postpartum preset already lists the baby rows,
 * so `childRow={false}` there.
 */
export function LogLater({ childRow = true }: { childRow?: boolean }) {
  const t = useTranslations('nav.later');
  const router = useRouter();
  const rtl = useDirection() === 'rtl';
  const childrenOn = useModuleEnabled('children'); // admin switch «کودک»
  const vitalsOn = useModuleEnabled('vitals'); // admin switch «علائم حیاتی»
  const todoOn = useModuleEnabled('todo'); // admin switch «لیست کارها»
  const withChild = childRow && childrenOn;
  const kids = useChildren({ enabled: withChild });
  const child = withChild ? kids.data?.children.find((c) => c.role === 'owner') : undefined;

  // Navigating closes every open sheet (shared/sheet closeSheetOnRouteChange) — no closeSheet() racing the router with a history step.
  const go = (href: string) => router.push(href);

  return (
    <>
    {vitalsOn ? (
    <section className="llt" aria-labelledby="llt-vitals-title">
      <h3 id="llt-vitals-title" className="lday-sec-title">
        {t('vitalsTitle')}
      </h3>
      <button type="button" className="llt-row" onClick={() => openSheet(VITALS_SHEET)} {...pilot('home.later-vitals')}>
        <IconCircle icon="heartLine" tone="bloom" size="md" />
        <span className="llt-row-text">
          <span className="llt-row-title">{t('vitals')}</span>
          <span className="llt-row-sub">{t('vitalsSub')}</span>
        </span>
        <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} className="llt-chev" />
      </button>
    </section>
    ) : null}
    <section className="llt" aria-labelledby="llt-title">
      <h3 id="llt-title" className="lday-sec-title">
        {t('title')}
      </h3>
      <div className="llt-pair">
        <button type="button" className="llt-btn" onClick={() => go(APPOINTMENT_NEW)} {...pilot('home.later-appointment')}>
          <Icon name="calendar" size={18} strokeWidth={1.8} className="llt-ic" />
          <span>{t('appointment')}</span>
        </button>
        {todoOn ? (
          <button type="button" className="llt-btn" onClick={() => openSheet(TODO_ADD_SHEET)} {...pilot('home.later-task')}>
            <Icon name="todo" size={18} strokeWidth={1.8} className="llt-ic" />
            <span>{t('task')}</span>
          </button>
        ) : null}
      </div>
      {child ? (
        <button type="button" className="llt-row" onClick={() => go(`/children/${child.id}/feeding`)} {...pilot('home.later-feeding')}>
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
    </>
  );
}
