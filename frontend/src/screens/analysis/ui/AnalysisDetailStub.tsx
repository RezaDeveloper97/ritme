'use client';

import { useTranslations } from 'next-intl';

import { useRouter } from '@/shared/i18n';
import { EmptyState, ScreenHeader, SecondaryButton, SkyLayer } from '@/shared/ui';

export type AnalysisDetail = 'cycle' | 'period' | 'symptoms' | 'correlations' | 'body' | 'labs' | 'monthly';

/**
 * Placeholder for an `/analysis/*` detail route until its screen lands
 * (B-N3-09: cycle, period, symptoms, correlations, body; B-N3-10: monthly,
 * labs). Those tasks swap the route's page body for their screen.
 */
export function AnalysisDetailStub({ detail }: { detail: AnalysisDetail }) {
  const t = useTranslations('analysis.detail');
  const router = useRouter();
  return (
    <div className="view an-page">
      <SkyLayer />
      <div className="scroll an-scroll">
        <ScreenHeader title={t(`titles.${detail}`)} onBack={() => router.push('/analysis')} backLabel={t('back')} />
        <EmptyState
          icon="chart"
          title={t('soonTitle')}
          body={t('soonBody')}
          action={
            <SecondaryButton block={false} onClick={() => router.push('/analysis')}>
              {t('toHub')}
            </SecondaryButton>
          }
        />
      </div>
    </div>
  );
}
