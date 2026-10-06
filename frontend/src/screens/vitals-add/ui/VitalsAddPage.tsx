'use client';

import { useTranslations } from 'next-intl';

import { AddVitalForm } from '@/features/add-vital';
import { useVitalsHub, type VitalType } from '@/entities/vital';
import { useRouter } from '@/shared/i18n';
import { ScreenHeader, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';

/**
 * `/vitals/bp/new`, `/vitals/glucose/new`, `/vitals/heart-rate/new`
 * (nbl_Vitals_AddBP / AddGlucose / AddHR). A form: close header, no bottom nav.
 * The form starts from the newest reading of the type (hub cache) and returns
 * to the hub after saving (after the urgent modal, when one opens).
 */
export function VitalsAddPage({ type }: { type: VitalType }) {
  const t = useTranslations('vitals');
  const router = useRouter();
  const hub = useVitalsHub();
  const header = (
    <ScreenHeader
      title={t(`add.titles.${type}`)}
      onBack={() => router.push('/vitals')}
      backLabel={t('common.close')}
      backIcon="close"
    />
  );
  if (hub.isPending) {
    return (
      <div className="view vt-screen vt-add-screen">
        <SkyLayer />
        <div className="scroll vt-scroll">
          {header}
          <SkeletonGroup label={t('common.loading')} className="vt-form">
            <Skeleton shape="card" className="vt-skel-hero" />
            <Skeleton shape="card" />
            <Skeleton shape="card" />
          </SkeletonGroup>
        </div>
      </div>
    );
  }
  return (
    <div className="view vt-screen vt-add-screen">
      <SkyLayer />
      <AddVitalForm type={type} latest={hub.data?.latest[type] ?? null} header={header} onDone={() => router.push('/vitals')} />
    </div>
  );
}
