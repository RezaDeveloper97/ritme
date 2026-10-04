'use client';

import { useTranslations } from 'next-intl';

import { useChild } from '@/entities/child';
import { useRouter } from '@/shared/i18n';
import { EmptyState, ScreenHeader, SecondaryButton, SkyLayer } from '@/shared/ui';

export type ChildSection = 'growth' | 'vaccines' | 'milestones' | 'learn';

/**
 * Interim page for `/children/[id]/{growth,vaccines,milestones,learn}` until
 * B-N5-06 builds them (v16_Growth / _Vaccines / _Milestones / _Learn), so the
 * child home's tiles never land on a 404. B-N5-06 swaps the route's import.
 */
export function ChildSectionSoon({ id, section }: { id: number; section: ChildSection }) {
  const t = useTranslations('children');
  const router = useRouter();
  const child = useChild(Number.isFinite(id) && id > 0 ? id : null);
  const name = child.data?.name ?? '';
  const title =
    section === 'growth'
      ? t('section.growth', { name })
      : section === 'vaccines'
        ? t('section.vaccines', { name })
        : section === 'milestones'
          ? t('section.milestones')
          : t('section.learn');
  const back = () => router.push(`/children/${id}`);
  return (
    <div className="view chd-screen">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={title} onBack={back} backLabel={t('common.back')} />
        <div className="chd-body">
          <EmptyState
            icon="sprout"
            title={t('section.soonTitle')}
            body={t('section.soonBody')}
            action={<SecondaryButton onClick={back}>{t('section.toChild')}</SecondaryButton>}
          />
        </div>
      </div>
    </div>
  );
}
