'use client';

import clsx from 'clsx';
import { useTranslations } from 'next-intl';
import { type KeyboardEvent, useRef, useState } from 'react';

import { Link, useDirection, useRouter } from '@/shared/i18n';
import { openSheet } from '@/shared/sheet';
import { Icon } from '@/shared/ui';

import {
  REMINDER_TABS,
  type RemindersTab,
  nextTab,
  parseTab,
  tabHref,
  visibleSections,
} from '../model/view';
import { AppointmentSection } from './AppointmentSection';
import { MedicationSection } from './MedicationSection';
import { TodayCard } from './TodayCard';

/** The AddChooser sheet (`app/sheets/registry.tsx`). */
const ADD_SHEET = 'reminders-add';
/** The existing notification-settings sheet (profile → «اعلان‌ها»). */
const NOTIFICATIONS_SHEET = 'notifications';

const PANEL_ID = 'rmd-panel';

/**
 * Reminders (`v13_Reminders`, light `nbl_` / dark `nbd_`): the hub the home
 * card's «همه» and the profile row lead to — today's dose strip, the
 * medication list with switches, upcoming appointments, and the add CTA.
 *
 * The tab lives in `?tab=` (read by the route, written back with `replace`) so
 * a reload or a return from a form lands on the same filter.
 */
export function RemindersPage({ initialTab }: { initialTab?: string }) {
  const t = useTranslations('care');
  const router = useRouter();
  const dir = useDirection();
  const [tab, setTab] = useState<RemindersTab>(() => parseTab(initialTab));
  const tabRefs = useRef<Partial<Record<RemindersTab, HTMLButtonElement | null>>>({});
  const show = visibleSections(tab);

  const select = (next: RemindersTab, focus = false) => {
    if (focus) tabRefs.current[next]?.focus();
    if (next === tab) return;
    setTab(next);
    router.replace(tabHref(next), { scroll: false });
  };

  const onTabKey = (event: KeyboardEvent<HTMLDivElement>) => {
    const next = nextTab(tab, event.key, dir === 'rtl' ? 'rtl' : 'ltr');
    if (!next) return;
    event.preventDefault();
    select(next, true);
  };

  return (
    <div className="view rmd-page">
      <div className="scroll">
        <header className="rmd-hdr">
          <Link href="/home" className="rmd-hdr-btn" aria-label={t('back')}>
            <Icon name={dir === 'rtl' ? 'chevronRight' : 'chevronLeft'} size={20} strokeWidth={1.8} />
          </Link>
          <div className="rmd-hdr-text">
            <h1 className="rmd-hdr-title">{t('title')}</h1>
            <p className="rmd-hdr-sub">{t('subtitle')}</p>
          </div>
          <button
            type="button"
            className="rmd-hdr-btn"
            aria-label={t('notificationSettings')}
            onClick={() => openSheet(NOTIFICATIONS_SHEET)}
          >
            <Icon name="bell" size={20} strokeWidth={1.8} />
          </button>
        </header>

        <div className="rmd-body">
          <div className="rmd-tabs" role="tablist" aria-label={t('title')} onKeyDown={onTabKey}>
            {REMINDER_TABS.map((key) => (
              <button
                key={key}
                ref={(el) => {
                  tabRefs.current[key] = el;
                }}
                type="button"
                role="tab"
                id={`rmd-tab-${key}`}
                aria-selected={tab === key}
                aria-controls={PANEL_ID}
                tabIndex={tab === key ? 0 : -1}
                className={clsx('rmd-tab', tab === key && 'on')}
                onClick={() => select(key)}
              >
                {t(`tabs.${key}`)}
              </button>
            ))}
          </div>

          <div id={PANEL_ID} role="tabpanel" aria-labelledby={`rmd-tab-${tab}`} className="rmd-panel">
            {show.today && <TodayCard />}
            {show.medications && <MedicationSection />}
            {show.appointments && <AppointmentSection />}
          </div>

          <button type="button" className="rmd-cta" onClick={() => openSheet(ADD_SHEET)}>
            <Icon name="plus" size={18} strokeWidth={2.2} />
            {t('addNewReminder')}
          </button>
        </div>
      </div>
    </div>
  );
}
