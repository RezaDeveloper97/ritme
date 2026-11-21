'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import {
  type JoinApplication,
  useJoinApplication,
  useJoinApplications,
  useOpenJoinDraft,
  joinKeys,
} from '@/entities/city-service';
import { getApiErrorCode } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Card, EmptyState, InfoNote, PrimaryButton, ProgressSteps, ScreenHeader, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';
import { useQueryClient } from '@tanstack/react-query';

import { DonePage, type JoinStep, pickApplication, resumeStep, STEP_COUNT, StepDocs, StepIntro, StepPlace, StepServices } from '@/widgets/city-join-form';
import '@/shared/ui/styles/page.css';
import '@/shared/ui/styles/cjn.css';


const STEP_KEYS = ['intro', 'place', 'services', 'docs'] as const;

function Wizard({ app }: { app: JoinApplication }) {
  const t = useTranslations('cityJoin');
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
  const back = () => (step === 1 ? router.push('/city-services') : go((step - 1) as JoinStep));

  return (
    <div className="view cjn-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader
          title={t('title')}
          subtitle={t('stepLabel', { n: num(step), total: num(STEP_COUNT) })}
          onBack={back}
          backLabel={t('back')}
        />
        <div ref={top} className="cjn-body">
          <ProgressSteps total={STEP_COUNT} current={step} label={t('stepLabel', { n: num(step), total: num(STEP_COUNT) })} />
          <ol className="cjn-steps" aria-label={t('stepsLabel')}>
            {STEP_KEYS.map((k, i) => (
              <li key={k} className="cjn-step-name" aria-current={step === i + 1 ? 'step' : undefined} data-done={i + 1 < step}>
                {t(`steps.${k}`)}
              </li>
            ))}
          </ol>

          {locked ? <InfoNote icon="lock">{t('locked')}</InfoNote> : null}
          {app.status === 'needs_info' ? (
            <InfoNote icon="info">
              <strong>{t('needsInfo')}</strong> {app.review_note ?? ''}
            </InfoNote>
          ) : null}

          {step === 1 ? <StepIntro key={1} app={app} onNext={() => go(2)} onSaveError={onSaveError} /> : null}
          {step === 2 ? <StepPlace key={2} app={app} onNext={() => go(3)} onSaveError={onSaveError} /> : null}
          {step === 3 ? <StepServices key={3} app={app} onNext={() => go(4)} onSaveError={onSaveError} /> : null}
          {step === 4 ? <StepDocs key={4} app={app} onGoStep={go} onSaveError={onSaveError} /> : null}
        </div>
        <div className="page-tail" />
      </div>
    </div>
  );
}

function Frame({ title, children, onBack, backLabel }: { title: string; children: React.ReactNode; onBack: () => void; backLabel: string }) {
  return (
    <div className="view cjn-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={title} onBack={onBack} backLabel={backLabel} />
        <div className="cjn-body">{children}</div>
        <div className="page-tail" />
      </div>
    </div>
  );
}

/**
 * `/city-services/join` — the business join flow (CB-DIR-10, nbl_Dir_Join / JoinForm / JoinDocs / JoinDone). Opens the
 * account's draft (or creates one) and resumes it at the next step; once sent, the done screen shows the tracking code
 * and the review timeline. Bookings made through Ritme are requests that the Ritme team confirms. No bottom nav.
 */
export function JoinPage() {
  const t = useTranslations('cityJoin');
  const router = useRouter();
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
  const leave = () => router.push('/city-services');

  if (list.isError || open.isError || app.isError) {
    return (
      <Frame title={t('title')} onBack={leave} backLabel={t('back')}>
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
      </Frame>
    );
  }

  if (!app.data) {
    return (
      <Frame title={t('title')} onBack={leave} backLabel={t('back')}>
        <SkeletonGroup label={t('loading')} className="cjn-stack">
          <Skeleton shape="block" />
          <Skeleton shape="block" />
          <Skeleton shape="block" />
        </SkeletonGroup>
      </Frame>
    );
  }

  if (app.data.editable) return <Wizard key={app.data.id} app={app.data} />;

  return (
    <Frame title={t('done.title')} onBack={leave} backLabel={t('back')}>
      <DonePage
        app={app.data}
        newBusy={open.isPending}
        onNew={() => {
          asked.current = true;
          open.mutate();
        }}
      />
    </Frame>
  );
}
