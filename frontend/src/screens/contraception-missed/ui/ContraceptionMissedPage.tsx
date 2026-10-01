'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  type ContraceptionMethodCode,
  isPillMethod,
  missedGuide,
  type MissedPillRule,
  useContraception,
  useMissedPillRules,
} from '@/entities/contraception';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import {
  Card,
  ChipGroup,
  EmptyState,
  InfoNote,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SectionTitle,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  UrgentCard,
} from '@/shared/ui';

/**
 * «قرص جا افتاده» (`/contraception/missed`, CB-CONTRA-03, nbl_Contra_Missed):
 * count chips → numbered steps → danger card → general-guidance note. Every
 * line of guidance is the admin-editable catalog group `missed_pill_rules`
 * (decision #7) picked for the user's pill; nothing clinical lives in code.
 * Back-header flow screen: no bottom nav.
 */
export function ContraceptionMissedPage() {
  const t = useTranslations('contraception');
  const router = useRouter();
  const locale = useLocale();
  const overview = useContraception();
  const rules = useMissedPillRules(locale);
  const method = overview.data?.method ?? null;

  let body;
  if (overview.isPending || rules.isPending) {
    body = (
      <SkeletonGroup label={t('missed.loading')} className="ctr-skel">
        <Skeleton shape="card" />
        <Skeleton width="short" />
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (overview.isError || rules.isError) {
    const refetching = overview.isFetching || rules.isFetching;
    body = (
      <EmptyState
        icon="pill"
        title={t('error.title')}
        body={t('error.body')}
        action={
          <PrimaryButton
            icon="refresh"
            loading={refetching}
            onClick={() => {
              if (overview.isError) void overview.refetch();
              if (rules.isError) void rules.refetch();
            }}
          >
            {t('error.retry')}
          </PrimaryButton>
        }
      />
    );
  } else if (!method || !isPillMethod(method.method)) {
    body = (
      <EmptyState
        icon="pill"
        title={t('missed.notPill.title')}
        body={t('missed.notPill.body')}
        action={<PrimaryButton onClick={() => router.push('/contraception')}>{t('missed.notPill.cta')}</PrimaryButton>}
      />
    );
  } else {
    body = <Guide rules={rules.data} method={method.method} />;
  }

  return (
    <div className="view ctr-page">
      <SkyLayer />
      <div className="scroll ctr-scroll">
        <ScreenHeader
          title={t('missed.title')}
          subtitle={method && isPillMethod(method.method) ? t(`setup.methods.${method.method}`) : undefined}
          onBack={() => router.push('/contraception')}
          backLabel={t('missed.back')}
        />
        {body}
      </div>
    </div>
  );
}

function Guide({ rules, method }: { rules: MissedPillRule[]; method: ContraceptionMethodCode }) {
  const t = useTranslations('contraception.missed');
  const locale = useLocale() as Locale;
  const guide = missedGuide(rules, method);
  const [picked, setPicked] = useState(0);
  const current = guide.countRules[picked] ?? guide.countRules[0] ?? guide.baseRule;

  if (!current && guide.urgent.length === 0) {
    return <EmptyState icon="info" title={t('unavailable.title')} body={t('unavailable.body')} />;
  }

  return (
    <>
      {guide.countRules.length > 0 ? (
        <Card className="ctrm-count">
          <h2 className="ctrm-count-title">
            {t('countTitle')}
          </h2>
          <ChipGroup label={t('countTitle')} className="ctrm-chips">
            {guide.countRules.map((rule, index) => (
              <PillChip
                key={rule.code}
                pressed={current === rule}
                onPressedChange={() => setPicked(index)}
                className="ctr-chip"
              >
                {rule.title ?? formatNumber(rule.missed ?? index + 1, locale)}
              </PillChip>
            ))}
          </ChipGroup>
        </Card>
      ) : null}

      {current && current.steps.length > 0 ? (
        <section className="ctrm-steps-wrap" aria-labelledby="ctrm-steps-title">
          <SectionTitle id="ctrm-steps-title" title={t('stepsTitle')} />
          <Card className="ctrm-steps-card">
            <ol className="ctrm-steps" aria-label={t('stepsLabel')} aria-live="polite">
              {current.steps.map((step, index) => (
                <li key={`${current.code}-${index}`} className="ctrm-step">
                  <span className="ctrm-step-num" aria-hidden>
                    {formatNumber(index + 1, locale)}
                  </span>
                  <span className="ctrm-step-text">{step}</span>
                </li>
              ))}
            </ol>
          </Card>
        </section>
      ) : null}

      {guide.urgent.map((rule) => (
        <UrgentCard key={rule.code} title={rule.title ?? rule.body} urgent={false} className="ctrm-urgent">
          {rule.title && rule.body ? rule.body : null}
        </UrgentCard>
      ))}

      {current?.body ? <InfoNote className="ctrm-note">{current.body}</InfoNote> : null}
    </>
  );
}
