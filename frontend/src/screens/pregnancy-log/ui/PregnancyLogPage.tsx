'use client';

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
    <div className="view" style={{ background: 'var(--page)' }}>
      <div className="scroll">
        <div style={{ padding: '10px 20px 0', textAlign: 'start' }}>
          <Link href="/pregnancy" className="iconbtn" aria-label={t('back')} style={{ display: 'inline-flex' }}>
            <Icon name="chevronRight" size={20} />
          </Link>
          <div className="titr" style={{ marginTop: 8 }}>{t('log.title')}</div>
          <p className="sub" style={{ margin: '6px 0 0' }}>{t('log.subtitle')}</p>
        </div>

        {/* Tab switcher */}
        <div style={{ padding: '14px 16px 0' }}>
          <div className="seg" style={{ display: 'flex', gap: 4 }}>
            {TABS.map((tb) => (
              <button
                key={tb}
                type="button"
                onClick={() => setTab(tb)}
                className={tab === tb ? 'on' : ''}
                style={{
                  flex: 1,
                  padding: '9px 6px',
                  borderRadius: 12,
                  border: 0,
                  cursor: 'pointer',
                  fontFamily: 'inherit',
                  fontSize: 12.5,
                  fontWeight: 700,
                  background: tab === tb ? 'var(--brand)' : 'transparent',
                  color: tab === tb ? 'var(--on-accent)' : 'var(--muted)',
                }}
              >
                {t(`log.tabs.${tb}`)}
              </button>
            ))}
          </div>
        </div>

        <div style={{ padding: '14px 16px 0' }}>
          {tab === 'symptoms' && <SymptomsForm t={t} />}
          {tab === 'weekly' && <WeeklyForm t={t} />}
          {tab === 'movement' && <MovementForm t={t} />}
        </div>

        <div style={{ height: 24 }} />
      </div>

      <BottomNav />
    </div>
  );
}
