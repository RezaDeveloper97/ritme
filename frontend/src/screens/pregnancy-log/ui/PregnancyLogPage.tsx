'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { useRouter } from '@/shared/i18n';
import { ScreenHeader, type SegmentedTab, SegmentedTabs, SkyLayer } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

import { MovementForm } from './MovementForm';
import { SymptomsForm } from './SymptomsForm';
import { WeeklyForm } from './WeeklyForm';

type Tab = 'symptoms' | 'weekly' | 'movement';
const TABS: Tab[] = ['symptoms', 'weekly', 'movement'];

/** Daily pregnancy logging with three tabs (symptoms / weekly checkup / fetal
 *  movement). The initial tab can be deep-linked via `?tab=` from the tracker. */
export function PregnancyLogPage({ initialTab }: { initialTab?: string }) {
  const t = useTranslations('pregnancy');
  const router = useRouter();
  const [tab, setTab] = useState<Tab>(TABS.includes(initialTab as Tab) ? (initialTab as Tab) : 'symptoms');
  const tabs: SegmentedTab<Tab>[] = TABS.map((value) => ({ value, label: t(`log.tabs.${value}`) }));

  return (
    <div className="view plog-page pgn-screen">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader
          title={t('log.title')}
          subtitle={t('log.subtitle')}
          onBack={() => router.push('/pregnancy')}
          backLabel={t('back')}
        />
        <div className="pgn-body is-form">
          <SegmentedTabs tabs={tabs} value={tab} onChange={setTab} label={t('log.title')} panelId={(v) => `plog-panel-${v}`} />
          <div role="tabpanel" id={`plog-panel-${tab}`}>
            {tab === 'symptoms' && <SymptomsForm t={t} />}
            {tab === 'weekly' && <WeeklyForm t={t} />}
            {tab === 'movement' && <MovementForm t={t} />}
          </div>
        </div>
      </div>
      <BottomNav />
    </div>
  );
}
