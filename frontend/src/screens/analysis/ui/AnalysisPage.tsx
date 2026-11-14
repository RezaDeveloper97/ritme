'use client';

import { useTranslations } from 'next-intl';
import { useState, type ReactNode } from 'react';

import { useRouter } from '@/shared/i18n';
import { useMounted } from '@/shared/lib/use-mounted';
import { EmptyState, PrimaryButton, SegmentedTabs, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';
import { BottomNav, useNavMode } from '@/widgets/bottom-nav';

import { hubVariant } from '../model/hub';
import type { AnalysisTab } from '../model/tab';
import { AnalysisHub } from './AnalysisHub';


/**
 * `/analysis` (B-N3-08): the «تحلیل» bottom-nav tab. Two tabs on top —
 * «تحلیل» (the hub) and «تاریخچه» (`historyPanel`, the cycle history from
 * screens/cycle, composed in by the route; `/cycle` redirects here). The life-stage
 * mode picks the hub — see {@link hubVariant}. `ttcHub` is the TTC hub
 * (screens/analysis-ttc, B-N3-11), composed in by the route so the two screen
 * slices never import each other. `pregnancyHub` is the pregnancy hub
 * (screens/analysis-pregnancy, B-N3-12), composed in the same way, and
 * `postpartumHub` the postpartum hub (screens/analysis-postpartum, B-N5-07).
 */
export function AnalysisPage({
  ttcHub,
  pregnancyHub,
  postpartumHub,
  historyPanel,
  initialTab = 'analysis',
}: {
  ttcHub?: ReactNode;
  pregnancyHub?: ReactNode;
  postpartumHub?: ReactNode;
  historyPanel?: ReactNode;
  initialTab?: AnalysisTab;
} = {}) {
  const t = useTranslations('analysis.hub');
  const tTabs = useTranslations('analysis.tabs');
  const [tab, setTab] = useState<AnalysisTab>(historyPanel ? initialTab : 'analysis');
  const changeTab = (next: AnalysisTab) => {
    setTab(next);
    // Keep the URL shareable / back-safe without a server round-trip.
    const url = new URL(window.location.href);
    if (next === 'history') url.searchParams.set('tab', 'history');
    else url.searchParams.delete('tab');
    window.history.replaceState(window.history.state, '', url);
  };
  const router = useRouter();
  const mounted = useMounted();
  const { mode, pending } = useNavMode();
  const variant = hubVariant(mode);

  let content;
  if (tab === 'history' && historyPanel) {
    content = historyPanel;
  } else if (!mounted || pending) {
    content = (
      <SkeletonGroup label={t('loading')} className="an-skel">
        <Skeleton shape="block" className="an-skel-finding" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (mode === 'ttc' && ttcHub) {
    content = ttcHub;
  } else if (mode === 'postpartum' && postpartumHub) {
    content = postpartumHub;
  } else if (variant === 'pregnancy' && pregnancyHub) {
    content = pregnancyHub;
  } else if (variant === 'pregnancy') {
    // Fallback when the route composes no pregnancy hub.
    content = (
      <div className="an-hub">
        <h1 className="an-title">{t('title')}</h1>
        <EmptyState
          icon="heart"
          title={t('pregnancy.title')}
          body={t('pregnancy.body')}
          action={
            <PrimaryButton block={false} onClick={() => router.push('/pregnancy')}>
              {t('pregnancy.cta')}
            </PrimaryButton>
          }
        />
      </div>
    );
  } else {
    content = <AnalysisHub variant={variant} />;
  }

  return (
    <div className="view an-page">
      <SkyLayer />
      <div className="scroll an-scroll">
        {historyPanel ? (
          <SegmentedTabs
            label={tTabs('label')}
            value={tab}
            tabs={[
              { value: 'analysis', label: tTabs('analysis') },
              { value: 'history', label: tTabs('history') },
            ]}
            onChange={changeTab}
            className="an-tabs"
          />
        ) : null}
        {content}
      </div>
      <BottomNav />
    </div>
  );
}
