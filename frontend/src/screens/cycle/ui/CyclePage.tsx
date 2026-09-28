'use client';

import { useTranslations } from 'next-intl';

import { useMounted } from '@/shared/lib/use-mounted';
import { BottomNav } from '@/widgets/bottom-nav';
import { SmartTipCard } from '@/widgets/smart-tip';
import { WeekSummaryCard } from '@/widgets/week-summary';

import { BmiCard } from './BmiCard';
import { CycleSummaryCard } from './CycleSummaryCard';
import { MyCyclesCard } from './MyCyclesCard';

/**
 * Analysis screen — the user's own cycle history and numbers, moved here from
 * the home screen so the home feed stays about today.
 */
export function CyclePage() {
  const t = useTranslations('cycle');
  // Every card below reads a query gated on the token in localStorage: off
  // during SSR, on in the browser, so the first client render would show
  // skeletons where the server rendered nothing (hydration mismatch). The cards
  // mount only after hydration; their data is client-fetched anyway.
  const mounted = useMounted();

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

        {/* Order per product: my cycles → smart tip → cycle summary →
            week summary → BMI. */}
        {mounted && (
          <>
            <MyCyclesCard />
            <SmartTipCard />
            <CycleSummaryCard />
            <WeekSummaryCard />
            <BmiCard />
          </>
        )}
        <div className="page-tail" />
      </div>

      <BottomNav />
    </div>
  );
}
