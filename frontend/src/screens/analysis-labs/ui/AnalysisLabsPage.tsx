'use client';

import { useTranslations } from 'next-intl';

import { DEFAULT_RANGE, useAnalysisSummary } from '@/entities/analysis';
import { useRouter } from '@/shared/i18n';
import { EmptyState, Icon, InfoNote, ScreenHeader, SkyLayer, StatusPill } from '@/shared/ui';
import { PlusBadge } from '@/shared/ui/plus-gate';
import { BottomNav } from '@/widgets/bottom-nav';

/**
 * `/analysis/labs` (An_Labs, B-N3-10). Lab results arrive with the lab
 * analysis (B-N6-06); until then the screen is an honest empty state with the
 * «آنالیز آزمایش» entry marked «به‌زودی». The hub's `labs` section (cached
 * from the hub) says whether this is a Plus lock for the reader.
 */
export function AnalysisLabsPage() {
  const t = useTranslations('analysis.reports');
  const tl = useTranslations('analysis.reports.labs');
  const tPlus = useTranslations('plus.gate');
  const router = useRouter();
  const summary = useAnalysisSummary(DEFAULT_RANGE);
  const locked = summary.data?.sections.labs.locked === true;

  return (
    <div className="view axc-page">
      <SkyLayer />
      <div className="scroll axc-scroll">
        <ScreenHeader
          title={tl('title')}
          subtitle={tl('sub')}
          onBack={() => router.push('/analysis')}
          backLabel={t('back')}
          className="axc-hdr"
        />
        <div className="axc-content">
          <section className="nb-card alb-card">
            <EmptyState
              icon="flask"
              title={tl('emptyTitle')}
              body={tl('emptyBody')}
              action={
                <div className="alb-action">
                  <button type="button" className="alb-cta" aria-disabled="true" aria-describedby="alb-cta-note">
                    <Icon name="flaskLh" size={18} strokeWidth={2} />
                    <span className="alb-cta-label">{tl('cta')}</span>
                    <StatusPill tone="brand">{tl('soon')}</StatusPill>
                  </button>
                  {locked ? (
                    <p id="alb-cta-note" className="alb-plus">
                      <PlusBadge label={tPlus('label')} />
                      <span>{tl('locked')}</span>
                    </p>
                  ) : null}
                </div>
              }
            />
          </section>
          <InfoNote>{tl('note')}</InfoNote>
        </div>
      </div>
      <BottomNav />
    </div>
  );
}
