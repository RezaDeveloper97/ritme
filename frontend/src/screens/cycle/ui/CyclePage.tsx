'use client';

import { useTranslations } from 'next-intl';

import { BottomNav } from '@/widgets/bottom-nav';

import { BmiCard } from './BmiCard';
import { CycleSummaryCard } from './CycleSummaryCard';
import { MyCyclesCard } from './MyCyclesCard';

/**
 * Analysis screen — the user's own cycle history and numbers, moved here from
 * the home screen so the home feed stays about today.
 */
export function CyclePage() {
  const t = useTranslations('cycle');

  return (
    <div className="view cyc-page">
      <div className="home-grad cyc-grad" />

      <div className="scroll page-scroll cyc-scroll">
        <div className="cyc-hdr is-solo">
          <div className="cyc-brand">
            <div className="cyc-title">{t('title')}</div>
            <div className="cyc-tagline">{t('tagline')}</div>
          </div>
        </div>

        <MyCyclesCard />
        <CycleSummaryCard />
        <BmiCard />
        <div className="page-tail" />
      </div>

      <BottomNav />
    </div>
  );
}
