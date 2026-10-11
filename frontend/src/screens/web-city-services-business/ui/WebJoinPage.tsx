'use client';

import { useQueryClient } from '@tanstack/react-query';
import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import {
  type JoinApplication,
  joinKeys,
  useJoinApplication,
  useJoinApplications,
  useOpenJoinDraft,
} from '@/entities/city-service';
import { getApiErrorCode } from '@/shared/api';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { isAuthenticated } from '@/shared/session';
import { Card, EmptyState, Icon, InfoNote, PrimaryButton, Skeleton, SkeletonGroup } from '@/shared/ui';
import {
  DonePage,
  type JoinStep,
  pickApplication,
  resumeStep,
  STEP_COUNT,
  StepDocs,
  StepIntro,
  StepPlace,
  StepServices,
} from '@/widgets/city-join-form';

import { BusinessShell } from './BusinessShell';

const STEP_KEYS = ['intro', 'place', 'services', 'docs'] as const;
const BUSINESS_HOME = '/business';

/** The venue card as far as the draft says (board: «پیش‌نمایش کارت مجموعه‌ات»); rating and distance come after publishing. */
function Preview({ app }: { app: JoinApplication }) {
  const t = useTranslations('webBusiness.join');
  return (
    <aside className="wbj-preview" aria-label={t('previewTitle')}>
      <h2 className="wbj-side-t">{t('previewTitle')}</h2>
      <div className="wbj-card">
        <strong className="wbj-card-name">{app.business_name || t('previewNameEmpty')}</strong>
        {app.district ? <span className="wbj-card-sub">{app.district}</span> : null}
        <span className="wbj-card-note">{t('previewNote')}</span>
      </div>
    </aside>
  );
}

function Wizard({ app }: { app: JoinApplication }) {
  const t = useTranslations('webBusiness.join');
  const tj = useTranslations('cityJoin');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const qc = useQueryClient();
  const [step, setStep] = useState<JoinStep>(() => resumeStep(app));
  const [locked, setLocked] = useState(false);
  const top = useRef<HTMLDivElement>(null);
  const num = (n: number) => formatNumber(n, locale);

  const go = (next: JoinStep) => {
    setStep(next);
    top.current?.scrollIntoView({ block: 'start' });
  };
  // A 409 `application_locked` means the team already has it: refetch, the page switches to the done screen.
  const onSaveError = (e: unknown) => {
    if (getApiErrorCode(e) === 'application_locked') {
      setLocked(true);
      void qc.invalidateQueries({ queryKey: joinKeys.all });
    }
  };
  // The step form saves what was typed when it unmounts, so leaving is just a navigation.
  const saveExit = () => router.push(BUSINESS_HOME);

  return (
    <div className="wbj-page">
      <header className="wbj-head">
        <h1 className="wbj-h1">{t('title')}</h1>
        <button type="button" className="wbj-exit" onClick={saveExit}>
          {t('saveExit')}
        </button>
      </header>

      <div className="wbj-layout">
        <aside className="wbj-side">
          <ol className="wbj-steps" aria-label={t('stepsLabel')}>
            {STEP_KEYS.map((k, i) => {
              const n = (i + 1) as JoinStep;
              const state = n < step ? 'done' : n === step ? 'current' : 'todo';
              return (
                <li key={k} className={clsx('wbj-step', `is-${state}`)} aria-current={n === step ? 'step' : undefined}>
                  <span className="wbj-dot" aria-hidden="true">
                    {state === 'done' ? <Icon name="check" size={14} /> : num(n)}
                  </span>
                  <span className="wbj-step-text">
                    <strong>{tj(`steps.${k}`)}</strong>
                    {state !== 'todo' ? <small>{state === 'done' ? t('stateDone') : t('stateCurrent')}</small> : null}
                  </span>
                </li>
              );
            })}
          </ol>
          <p className="wbj-hint">{t('autosave')}</p>
          <Preview app={app} />
        </aside>

        <section className="wbj-main" ref={top} aria-label={tj('stepLabel', { n: num(step), total: num(STEP_COUNT) })}>
          <p className="wbj-stepno">{tj('stepLabel', { n: num(step), total: num(STEP_COUNT) })}</p>
          {locked ? <InfoNote icon="lock">{t('locked')}</InfoNote> : null}
          {app.status === 'needs_info' ? (
            <InfoNote icon="info">
              <strong>{t('needsInfo')}</strong> {app.review_note ?? ''}
            </InfoNote>
          ) : null}
          {step === 1 ? <StepIntro key={1} app={app} showHero={false} leaveHref={BUSINESS_HOME} onNext={() => go(2)} onSaveError={onSaveError} /> : null}
          {step === 2 ? <StepPlace key={2} app={app} onNext={() => go(3)} onSaveError={onSaveError} /> : null}
          {step === 3 ? <StepServices key={3} app={app} onNext={() => go(4)} onSaveError={onSaveError} /> : null}
          {step === 4 ? <StepDocs key={4} app={app} onGoStep={go} onSaveError={onSaveError} /> : null}
          {step > 1 ? (
            <button type="button" className="wbj-prev" onClick={() => go((step - 1) as JoinStep)}>
              {t('back')}
            </button>
          ) : null}
        </section>
      </div>
    </div>
  );
}

/** The signed-in part: opens the account's draft (or creates one) and shows the wizard, or the done page once sent. */
function JoinFlow() {
  const t = useTranslations('webBusiness.join');
  const list = useJoinApplications();
  const open = useOpenJoinDraft();
  const picked = list.data ? pickApplication(list.data) : null;
  const asked = useRef(false);

  useEffect(() => {
    if (picked?.kind === 'none' && !asked.current) {
      asked.current = true;
      open.mutate();
    }
  }, [picked?.kind, open]);

  const id = picked?.id ?? open.data?.id ?? null;
  const app = useJoinApplication(id);

  if (list.isError || open.isError || app.isError) {
    return (
      <Card>
        <EmptyState
          icon="warning"
          title={t('loadError')}
          action={
            <PrimaryButton
              loading={list.isFetching || app.isFetching}
              onClick={() => {
                asked.current = false;
                void list.refetch();
                void app.refetch();
              }}
            >
              {t('retry')}
            </PrimaryButton>
          }
        />
      </Card>
    );
  }

  if (!app.data) {
    return (
      <SkeletonGroup label={t('loading')} className="wbj-skel">
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  }

  if (app.data.editable) return <Wizard key={app.data.id} app={app.data} />;

  return (
    <div className="wbj-done">
      <DonePage
        app={app.data}
        newBusy={open.isPending}
        homeHref="/places"
        onNew={() => {
          asked.current = true;
          open.mutate();
        }}
      />
    </div>
  );
}

/**
 * CB-DIR-12 — `W_Dir_Join` + `W_Dir_JoinDone`: the desktop join wizard and the done page, on the same draft API as the
 * app's join flow (CB-DIR-10) — a draft started here continues in the app and back. `business` is a public segment, so
 * the page itself asks a visitor without a session to log in. Under 1024px a signed-in visitor gets the app screens.
 */
export function WebJoinPage() {
  const t = useTranslations('webBusiness.join');
  const mounted = useMounted();
  const authed = mounted && isAuthenticated();

  let body = null;
  if (mounted && !authed) {
    body = (
      <div className="wbj-done">
        <Card>
          <EmptyState
            icon="lock"
            title={t('loginTitle')}
            body={t('loginBody')}
            action={
              <Link href="/signup" className="wbz-start">
                {t('login')}
              </Link>
            }
          />
        </Card>
      </div>
    );
  } else if (authed) {
    body = <JoinFlow />;
  }

  return <BusinessShell appHref="/city-services/join">{body}</BusinessShell>;
}
