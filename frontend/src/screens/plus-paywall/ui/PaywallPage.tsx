'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import { PlusCrown, usePlusPlans, usePlusStatus } from '@/entities/plus';
import { useRestorePlus, useStartTrial } from '@/features/purchase-plus';
import { useRouter, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { HeaderButton, Icon, PrimaryButton, SecondaryButton, SkyLayer } from '@/shared/ui';
import { useNavMode } from '@/widgets/bottom-nav';

type FeatureKey = 'analysis' | 'prediction' | 'assistant' | 'pdf' | 'tracking' | 'todo';

/** Free vs Plus rows of `nbl_Prem_Paywall` (marketing copy; prices come from the API). */
const FEATURES: readonly { key: FeatureKey; free: boolean }[] = [
  { key: 'analysis', free: false },
  { key: 'prediction', free: false },
  { key: 'assistant', free: false },
  { key: 'pdf', free: false },
  { key: 'tracking', free: true },
  { key: 'todo', free: true },
];

/**
 * «ریتمی پلاس» paywall (`/plus`, B-N2-07, `nbl_Prem_Paywall` / `nbd_Prem_Paywall`):
 * close + «بازیابی خرید», crown hero, free-vs-plus table, testimonial, and the
 * «انتخاب اشتراک» CTA with the first-subscription trial line. Teen mode never
 * sees an upsell (QUESTIONS #70) — it is sent back to «من».
 */
export function PaywallPage() {
  const t = useTranslations('plus');
  const router = useRouter();
  const loc = useLocale() as Locale;
  const navMode = useNavMode().mode;
  const status = usePlusStatus();
  const plans = usePlusPlans();
  const restore = useRestorePlus();
  const trial = useStartTrial();
  const [notice, setNotice] = useState<string | null>(null);

  useEffect(() => {
    if (navMode === 'teen') router.replace('/profile');
  }, [navMode, router]);

  const isPlus = status.data?.isPlus ?? false;
  const trialDays = plans.data?.trialDays ?? 0;
  const canTrial = !isPlus && (status.data?.trialAvailable ?? false) && trialDays > 0;

  const onRestore = () => {
    setNotice(null);
    restore.mutate(undefined, {
      onSuccess: (r) => {
        if (r.status.isPlus) router.push('/plus/manage');
        else setNotice(r.restored > 0 ? t('paywall.restored') : t('paywall.nothingToRestore'));
      },
      onError: () => setNotice(t('paywall.restoreFailed')),
    });
  };

  const onTrial = () => {
    setNotice(null);
    trial.mutate(undefined, {
      onSuccess: () => router.push('/plus/success?trial=1'),
      onError: () => setNotice(t('paywall.trialFailed')),
    });
  };

  if (navMode === 'teen') return <div className="view plus-page" />;

  return (
    <div className="view plus-page">
      <SkyLayer />
      <div className="scroll plus-scroll">
        <div className="plus-topbar">
          <HeaderButton label={t('close')} icon="x" onClick={() => router.push('/profile')} />
          <button type="button" className="plus-link" onClick={onRestore} disabled={restore.isPending} aria-busy={restore.isPending || undefined}>
            {t('paywall.restore')}
          </button>
        </div>

        <div className="plus-hero">
          <PlusCrown />
          <h1 className="plus-display">{t('paywall.title')}</h1>
          <p className="plus-lead">{isPlus ? t('paywall.active') : t('paywall.lead')}</p>
        </div>

        {notice ? (
          <p className="plus-notice" role="status">
            {notice}
          </p>
        ) : null}

        <table className="nb-card plus-compare" aria-label={t('paywall.tableLabel')}>
          <thead>
            <tr>
              <th scope="col" className="plus-compare-feature">
                <span className="sr-only">{t('paywall.feature')}</span>
              </th>
              <th scope="col">{t('paywall.free')}</th>
              <th scope="col" className="is-plus">
                {t('paywall.plus')}
              </th>
            </tr>
          </thead>
          <tbody>
            {FEATURES.map((f) => (
              <tr key={f.key}>
                <th scope="row" className="plus-compare-feature">
                  {t(`paywall.features.${f.key}`)}
                </th>
                <td>
                  <Mark on={f.free} label={f.free ? t('paywall.included') : t('paywall.excluded')} />
                </td>
                <td className="is-plus">
                  <Mark on label={t('paywall.included')} plus />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="plus-footer">
        {isPlus ? (
          <PrimaryButton onClick={() => router.push('/plus/manage')}>{t('paywall.manage')}</PrimaryButton>
        ) : (
          <PrimaryButton onClick={() => router.push('/plus/plans')}>{t('paywall.cta')}</PrimaryButton>
        )}
        {canTrial ? (
          <SecondaryButton variant="text" loading={trial.isPending} onClick={onTrial}>
            {t('paywall.trial', { days: formatNumber(trialDays, loc) })}
          </SecondaryButton>
        ) : null}
      </div>
    </div>
  );
}

function Mark({ on, label, plus }: { on: boolean; label: string; plus?: boolean }) {
  return (
    <span className={plus ? 'plus-mark is-plus' : 'plus-mark'} role="img" aria-label={label}>
      <Icon name={on ? 'check' : 'x'} size={on ? 17 : 16} strokeWidth={on ? 2.6 : 2.2} />
    </span>
  );
}
