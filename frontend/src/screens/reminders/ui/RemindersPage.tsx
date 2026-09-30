'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { useRouter } from '@/shared/i18n';
import { openSheet } from '@/shared/sheet';
import { HeaderButton, PrimaryButton, ScreenHeader, SegmentedTabs, SkyLayer } from '@/shared/ui';

import { REMINDER_TABS, type RemindersTab, parseTab, tabHref, visibleSections } from '../model/view';
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
  const [tab, setTab] = useState<RemindersTab>(() => parseTab(initialTab));
  const show = visibleSections(tab);

  const select = (next: RemindersTab) => {
    if (next === tab) return;
    setTab(next);
    router.replace(tabHref(next), { scroll: false });
  };

  return (
    <div className="view rmd-page">
      <div className="scroll rmd-screen">
        <SkyLayer />
        <ScreenHeader
          title={t('title')}
          subtitle={t('subtitle')}
          onBack={() => router.push('/home')}
          backLabel={t('back')}
          action={
            <HeaderButton
              icon="bell"
              label={t('notificationSettings')}
              onClick={() => openSheet(NOTIFICATIONS_SHEET)}
            />
          }
        />

        <div className="rmd-body">
          <SegmentedTabs
            tabs={REMINDER_TABS.map((key) => ({ value: key, label: t(`tabs.${key}`) }))}
            value={tab}
            onChange={select}
            label={t('title')}
            panelId={() => PANEL_ID}
          />

          <div id={PANEL_ID} role="tabpanel" aria-label={t(`tabs.${tab}`)} className="rmd-panel">
            {show.today && <TodayCard />}
            {show.medications && <MedicationSection />}
            {show.appointments && <AppointmentSection />}
          </div>

          <PrimaryButton icon="plus" onClick={() => openSheet(ADD_SHEET)}>
            {t('addNewReminder')}
          </PrimaryButton>
        </div>
      </div>
    </div>
  );
}
