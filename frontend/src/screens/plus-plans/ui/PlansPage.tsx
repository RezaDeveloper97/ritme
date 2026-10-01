'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useSearchParams } from 'next/navigation';
import { useEffect, useRef, useState, type KeyboardEvent } from 'react';

import { formatToman, planFromParam, usePlusPlans, usePlusStatus, type PlusPlan } from '@/entities/plus';
import { useDirection, useRouter, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import {
  EmptyState,
  Icon,
  PrimaryButton,
  ProgressSteps,
  ScreenHeader,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';
import { useNavMode } from '@/widgets/bottom-nav';

/**
 * «مدت اشتراک را انتخاب کن» (`/plus/plans`, B-N2-07, `nbl_Prem_Plans`):
 * plan cards from `GET /plus/plans` (title, badge, price per month, savings —
 * nothing hard-coded), perks, and «ادامه · <plan>» to checkout. Step 1 of 3.
 */
export function PlansPage() {
  const t = useTranslations('plus');
  const router = useRouter();
  const loc = useLocale() as Locale;
  const params = useSearchParams();
  const navMode = useNavMode().mode;
  const query = usePlusPlans();
  const status = usePlusStatus();
  const [chosen, setChosen] = useState<number | null>(null);

  useEffect(() => {
    if (navMode === 'teen') router.replace('/profile');
  }, [navMode, router]);

  const plans = query.data?.plans ?? [];
  const selected = plans.find((p) => p.id === chosen) ?? planFromParam(plans, params.get('plan'));
  const trialDays = query.data?.trialDays ?? 0;
  const showTrial = trialDays > 0 && (status.data?.trialAvailable ?? false);

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="plus-plans">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError) {
    body = (
      <EmptyState
        icon="crown"
        title={t('error.title')}
        body={t('error.body')}
        action={
          <PrimaryButton icon="refresh" loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  } else if (!plans.length) {
    body = <EmptyState icon="crown" title={t('plans.empty')} />;
  } else {
    body = (
      <>
        <PlanCards plans={plans} value={selected?.id ?? null} onChange={setChosen} />
        <ul className="plus-perks">
          {showTrial ? <Perk text={t('plans.perks.trial', { days: formatNumber(trialDays, loc) })} /> : null}
          <Perk text={t('plans.perks.reminder')} />
          <Perk text={t('plans.perks.devices')} />
        </ul>
      </>
    );
  }

  return (
    <div className="view plus-page">
      <SkyLayer />
      <div className="scroll plus-scroll">
        <ScreenHeader
          title={t('plans.title')}
          onBack={() => router.push('/plus')}
          backLabel={t('back')}
          center={<ProgressSteps total={3} current={1} label={t('steps', { current: formatNumber(1, loc), total: formatNumber(3, loc) })} className="plus-steps" />}
        />
        <div className="plus-intro">
          <h1 className="plus-title">{t('plans.title')}</h1>
          <p className="plus-lead">{t('plans.lead')}</p>
        </div>
        {body}
      </div>
      {selected ? (
        <div className="plus-footer">
          <PrimaryButton onClick={() => router.push(`/plus/checkout?plan=${selected.id}`)}>
            {t('plans.cta', { plan: selected.title })}
          </PrimaryButton>
        </div>
      ) : null}
    </div>
  );
}

function Perk({ text }: { text: string }) {
  return (
    <li className="plus-perk">
      <Icon name="check" size={16} strokeWidth={2.4} />
      <span>{text}</span>
    </li>
  );
}

function PlanCards({ plans, value, onChange }: { plans: PlusPlan[]; value: number | null; onChange: (id: number) => void }) {
  const t = useTranslations('plus');
  const loc = useLocale() as Locale;
  const rtl = useDirection() === 'rtl';
  const refs = useRef<Array<HTMLButtonElement | null>>([]);
  const tabStop = Math.max(0, plans.findIndex((p) => p.id === value));
  // APG radiogroup: arrows move selection and focus together, wrapping.
  const onKeyDown = (event: KeyboardEvent<HTMLButtonElement>, index: number) => {
    const n = plans.length;
    const forward = rtl ? 'ArrowLeft' : 'ArrowRight';
    const backward = rtl ? 'ArrowRight' : 'ArrowLeft';
    let next: number | null = null;
    if (event.key === forward || event.key === 'ArrowDown') next = (index + 1) % n;
    else if (event.key === backward || event.key === 'ArrowUp') next = (index - 1 + n) % n;
    else if (event.key === 'Home') next = 0;
    else if (event.key === 'End') next = n - 1;
    if (next === null) return;
    event.preventDefault();
    onChange(plans[next].id);
    refs.current[next]?.focus();
  };
  return (
    <div role="radiogroup" aria-label={t('plans.groupLabel')} className="plus-plans">
      {plans.map((plan, index) => {
        const checked = plan.id === value;
        const price = formatToman(plan.price, loc);
        return (
          <button
            key={plan.id}
            ref={(el) => {
              refs.current[index] = el;
            }}
            type="button"
            role="radio"
            aria-checked={checked}
            tabIndex={index === tabStop ? 0 : -1}
            className="plus-plan"
            onClick={() => onChange(plan.id)}
            onKeyDown={(event) => onKeyDown(event, index)}
          >
            {plan.badge ? <span className="plus-plan-badge">{plan.badge}</span> : null}
            <span className="plus-plan-dot" aria-hidden />
            <span className="plus-plan-main">
              <span className="plus-plan-head">
                <b className="plus-plan-title">{plan.title}</b>
                {plan.savingsPercent > 0 ? (
                  <span className="plus-save">{t('plans.save', { percent: formatNumber(plan.savingsPercent, loc) })}</span>
                ) : null}
              </span>
              <span className="plus-plan-sub">
                {plan.durationMonths <= 1
                  ? t('plans.monthly', { price })
                  : t('plans.every', { price, months: formatNumber(plan.durationMonths, loc) })}
              </span>
            </span>
            <span className="plus-plan-price">
              <b className="plus-plan-num">{formatToman(plan.monthlyPrice, loc)}</b>
              <span className="plus-plan-unit">{t('plans.perMonth')}</span>
            </span>
          </button>
        );
      })}
    </div>
  );
}
