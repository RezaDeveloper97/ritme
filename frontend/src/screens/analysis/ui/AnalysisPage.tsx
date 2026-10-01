'use client';

import { useTranslations } from 'next-intl';

import { useRouter } from '@/shared/i18n';
import { useMounted } from '@/shared/lib/use-mounted';
import { EmptyState, PrimaryButton, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';
import { BottomNav, useNavMode } from '@/widgets/bottom-nav';

import { hubVariant } from '../model/hub';
import { AnalysisHub } from './AnalysisHub';

/**
 * `/analysis` (B-N3-08). No nav tab of its own: it lives under the mode tab,
 * which the bottom nav lights here (docs/night-bloom/nav.md). The life-stage
 * mode picks the hub — see {@link hubVariant}.
 */
export function AnalysisPage() {
  const t = useTranslations('analysis.hub');
  const router = useRouter();
  const mounted = useMounted();
  const { mode, pending } = useNavMode();
  const variant = hubVariant(mode);

  let content;
  if (!mounted || pending) {
    content = (
      <SkeletonGroup label={t('loading')} className="an-skel">
        <Skeleton shape="block" className="an-skel-finding" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (variant === 'pregnancy') {
    // B-N3-12 builds the pregnancy hub (An_Hub_Preg).
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
      <div className="scroll an-scroll">{content}</div>
      <BottomNav />
    </div>
  );
}
