'use client';

import clsx from 'clsx';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { Link } from '@/shared/i18n';
import { Icon } from '@/shared/ui';
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
  const [tab, setTab] = useState<Tab>(
    TABS.includes(initialTab as Tab) ? (initialTab as Tab) : 'symptoms',
  );

  return (
    <div className="view plog-page">
      <div className="scroll">
        <div className="plog-hdr">
          <Link href="/pregnancy" className="iconbtn plog-back" aria-label={t('back')}>
            <Icon name="chevronRight" size={20} />
          </Link>
          <div className="titr plog-titr">{t('log.title')}</div>
          <p className="sub plog-sub">{t('log.subtitle')}</p>
        </div>

        {/* Tab switcher */}
        <div className="sec-tight">
          <div className="seg plog-tabs">
            {TABS.map((tb) => (
              <button
                key={tb}
                type="button"
                onClick={() => setTab(tb)}
                className={clsx('plog-tab', tab === tb && 'on')}
              >
                {t(`log.tabs.${tb}`)}
              </button>
            ))}
          </div>
        </div>

        <div className="sec-tight">
          {tab === 'symptoms' && <SymptomsForm t={t} />}
          {tab === 'weekly' && <WeeklyForm t={t} />}
          {tab === 'movement' && <MovementForm t={t} />}
        </div>

        <div className="plog-tail" />
      </div>

      <BottomNav />
    </div>
  );
}
