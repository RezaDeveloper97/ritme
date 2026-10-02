'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { type CompanionPartner, useCompanionHome } from '@/entities/companion';
import { getApiErrorStatus } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { clearOnboardingPending, getAuthToken, setAuthToken } from '@/shared/session';
import { Icon, Skeleton, SkeletonGroup, StatusPill } from '@/shared/ui';
import { OnbFrame } from '@/features/auth';

import { useFinishOnboarding } from '../api/onboarding';
import { FLOW_ROUTES, landingRoute } from '../model/flow';

/** The link just made: the newest one (link ids only grow). */
function newestPartner(partners: readonly CompanionPartner[]): CompanionPartner | null {
  return partners.reduce<CompanionPartner | null>((best, p) => (!best || p.link.id > best.link.id ? p : best), null);
}

function firstLetter(name: string): string {
  return Array.from(name.trim())[0] ?? '';
}

/**
 * nbl_Onb_PartnerLinked (B-N4-05): «به سارا وصل شدی» after the code was
 * accepted — her phase today when she shared her cycle — then «ورود به پنل
 * همدم» finishes onboarding and lands on `/companion`.
 */
export function PartnerLinkedStep() {
  const t = useTranslations('onboarding.flow.partnerLinked');
  const tf = useTranslations('onboarding.flow');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const home = useCompanionHome();
  const finish = useFinishOnboarding();
  const [leaving, setLeaving] = useState(false);

  const partner = home.data ? newestPartner(home.data.partners) : null;
  const name = partner?.partnerName ?? null;
  const cycle = partner?.cycle?.hasData ? partner.cycle : null;

  const enter = () => {
    if (finish.isPending || leaving) return;
    finish
      .mutateAsync()
      .then(() => {
        setLeaving(true);
        // Same hand-off as Ready: drop the resume marker, re-assert the session
        // cookie, full document load into the app.
        clearOnboardingPending();
        const token = getAuthToken();
        if (token) setAuthToken(token);
        window.location.replace(`/${locale}${landingRoute(null, false, 'male')}`);
      })
      .catch((error: unknown) => {
        if (getApiErrorStatus(error) === 422) router.replace(FLOW_ROUTES.name);
      });
  };

  const summary = [
    cycle?.daysToPeriod != null && cycle.daysToPeriod >= 0 ? t('nextPeriod', { days: cycle.daysToPeriod }) : null,
    partner?.note ?? null,
  ]
    .filter(Boolean)
    .join(' ');

  return (
    <OnbFrame
      className="cmh-linked"
      privacyNote={false}
      primary={{ label: t('enter'), onClick: enter, disabled: finish.isPending, loading: finish.isPending || leaving }}
    >
      <div className="cmh-linked-hero">
        <div className="cmh-linked-pair" aria-hidden>
          <span className="cmh-linked-av is-me">
            <Icon name="user" size={34} />
          </span>
          <span className="cmh-linked-line" />
          <span className="cmh-linked-av is-her">{name ? firstLetter(name) : <Icon name="heart" size={32} />}</span>
        </div>
        <h1 className="onb2-title is-center">{name ? t('title', { name }) : t('titleNoName')}</h1>
        <p className="onb2-sub is-center">{name ? t('body', { name }) : t('bodyNoName')}</p>
      </div>

      {home.isPending ? (
        <SkeletonGroup label={t('loading')} className="nb-card cmh-linked-card">
          <Skeleton width="medium" />
          <Skeleton />
          <Skeleton width="short" />
        </SkeletonGroup>
      ) : partner && (cycle || partner.phase === 'pregnancy' || summary) ? (
        <div className="nb-card cmh-linked-card">
          <div className="cmh-linked-card-head">
            <b className="cmh-linked-card-title">{name ? t('today', { name }) : t('todayNoName')}</b>
            {partner.phase !== 'general' ? <StatusPill tone="brand">{t(`phases.${partner.phase}`)}</StatusPill> : null}
          </div>
          {summary ? <p className="cmh-linked-card-body">{summary}</p> : null}
        </div>
      ) : null}
      {finish.isError && getApiErrorStatus(finish.error) !== 422 ? (
        <p className="onb2-error is-center" role="alert">
          {tf('saveError')}
        </p>
      ) : null}
    </OnbFrame>
  );
}
