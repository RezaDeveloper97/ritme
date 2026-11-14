'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState, type ReactNode } from 'react';

import { defaultPlan, formatTomanThousands, usePlusPlans, usePlusStatus, type PlusPlan } from '@/entities/plus';
import { useStartPlusTrial } from '@/features/purchase-plus';
import { Link, useRouter, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import { Card, Icon, PrimaryButton, Skeleton, SkeletonGroup, StatusPill } from '@/shared/ui';

/** The Plus-only actions that open this sheet; each has its own title + lead under `plus.requiredSheet`. */
export type PlusRequiredFeature = 'companion';

type T = ReturnType<typeof useTranslations<'plus'>>;

function PlanOffer({ plan, t, loc }: { plan: PlusPlan; t: T; loc: Locale }) {
  const router = useRouter();
  return (
    <>
      <Card as="section" className="plus-tsheet-plan" aria-labelledby="plus-rsheet-plan">
        <div className="plus-tsheet-plan-head">
          <h3 id="plus-rsheet-plan" className="plus-tsheet-plan-title">
            {plan.badge ? t('trialSheet.planTitle', { plan: plan.title, badge: plan.badge }) : plan.title}
          </h3>
          {plan.savingsPercent > 0 ? (
            <StatusPill tone="warm" className="plus-tsheet-pill">
              {t('requiredSheet.savings', { percent: formatNumber(plan.savingsPercent, loc) })}
            </StatusPill>
          ) : null}
        </div>
        <div className="plus-tsheet-prices">
          <span className="plus-tsheet-now">{t('trialSheet.now', { amount: formatTomanThousands(plan.price, loc) })}</span>
        </div>
        <p className="plus-tsheet-note">
          {t('requiredSheet.perMonth', { amount: formatTomanThousands(plan.monthlyPrice, loc) })}
        </p>
      </Card>
      <PrimaryButton onClick={() => router.push(`/plus/checkout?plan=${plan.id}`)}>
        {t('requiredSheet.buy', { plan: plan.title })}
      </PrimaryButton>
      <Link href="/plus/plans" className="plus-tsheet-compare">
        {t('requiredSheet.allPlans')}
      </Link>
    </>
  );
}

interface PlusRequiredSheetProps {
  feature: PlusRequiredFeature;
  open: boolean;
  onClose: () => void;
}

/**
 * «این قابلیت مخصوص ریتمی پلاس است»: opened where a free user reaches a
 * Plus-only action. Says the feature is not free, then makes the offer on the
 * spot — the free trial when she never used it (started here; the sheet closes
 * and she carries on), otherwise the featured plan with a checkout CTA. Needs
 * the `plus` message namespace on the route.
 */
export function PlusRequiredSheet({ feature, open, onClose }: PlusRequiredSheetProps) {
  const t = useTranslations('plus');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const status = usePlusStatus();
  const plans = usePlusPlans();
  const trial = useStartPlusTrial(plans.data);
  const [failed, setFailed] = useState(false);

  const trialDays = plans.data?.trialDays ?? 0;
  const canTrial = (status.data?.trialAvailable ?? false) && trialDays > 0;
  const plan = plans.data ? defaultPlan(plans.data.plans) : null;
  const days = formatNumber(trialDays, loc);

  const onTrial = () => {
    setFailed(false);
    trial.start({ onSuccess: onClose, onError: () => setFailed(true) });
  };

  let offer: ReactNode;
  if (status.isPending || plans.isPending) {
    offer = (
      <SkeletonGroup label={t('loading')}>
        <Skeleton shape="card" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (canTrial) {
    offer = (
      <>
        <Card as="section" className="plus-tsheet-plan" aria-labelledby="plus-rsheet-trial">
          <div className="plus-tsheet-plan-head">
            <h3 id="plus-rsheet-trial" className="plus-tsheet-plan-title">
              {t('requiredSheet.trialTitle', { days })}
            </h3>
            <StatusPill tone="warm" className="plus-tsheet-pill">
              {t('requiredSheet.trialPill')}
            </StatusPill>
          </div>
          <p className="plus-tsheet-note">{t('requiredSheet.trialBody', { days })}</p>
        </Card>
        {failed ? (
          <p className="plus-notice" role="alert">
            {t('paywall.trialFailed')}
          </p>
        ) : null}
        <PrimaryButton loading={trial.isPending} onClick={onTrial}>
          {t('requiredSheet.trialCta', { days })}
        </PrimaryButton>
        <Link href="/plus/plans" className="plus-tsheet-compare">
          {t('requiredSheet.allPlans')}
        </Link>
      </>
    );
  } else if (plan) {
    offer = <PlanOffer plan={plan} t={t} loc={loc} />;
  } else {
    // The catalogue did not load: the paywall still explains Plus and retries there.
    offer = <PrimaryButton onClick={() => router.push('/plus')}>{t('requiredSheet.see')}</PrimaryButton>;
  }

  return (
    <AppSheet
      open={open}
      onClose={onClose}
      size="full"
      className="plus-tsheet-panel"
      title={<span className="sr-only">{t('trialSheet.label')}</span>}
    >
      <div className="plus-tsheet">
        <Card as="section" className="plus-tsheet-hero" aria-labelledby="plus-rsheet-title">
          <Icon name="crown" size={30} strokeWidth={1.8} className="plus-tsheet-crown" />
          <h2 id="plus-rsheet-title" className="plus-tsheet-title">
            {t(`requiredSheet.${feature}.title`)}
          </h2>
          <p className="plus-tsheet-lead">{t(`requiredSheet.${feature}.lead`)}</p>
        </Card>
        {offer}
        <Link href="/plus" className="plus-tsheet-compare">
          {t('trialSheet.compare')}
        </Link>
      </div>
    </AppSheet>
  );
}
