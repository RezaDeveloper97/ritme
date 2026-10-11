'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  buildTimeline,
  type ClaimStatus,
  costTypeKey,
  type InsuranceQuestionnaire,
  type QuestionnaireStatus,
  type QuestionnaireStep,
  type TimelineRow,
  useAdvanceQuestionnaire,
  useQuestionnaires,
  useRecentClaims,
} from '@/entities/insurance';
import { type Locale, useRouter } from '@/shared/i18n';
import {
  type DateParts,
  formatDayMonth,
  fromApiDate,
  partsToDate,
  toApiDate,
  toParts,
  today,
} from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import {
  CalendarPicker,
  ChipGroup,
  EmptyState,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
  StepTimeline,
  type Tone,
} from '@/shared/ui';

const STATE_TONE: Record<QuestionnaireStatus, Tone> = {
  draft: 'brand',
  confirmed: 'brand',
  submitted: 'data',
  reviewing: 'data',
  doctor_review: 'warm',
  approved: 'data',
  conditional: 'warm',
  rejected: 'period',
};

const CLAIM_TONE: Record<ClaimStatus, Tone> = {
  submitted: 'data',
  reviewing: 'data',
  docs_requested: 'warm',
  paid: 'data',
  rejected: 'period',
  appealed: 'warm',
};

const STEPS: readonly QuestionnaireStep[] = ['submitted', 'reviewing', 'doctor_review', 'approved', 'conditional', 'rejected'];

/** The step she most likely wants to note next: the current timeline row (a result row suggests «approved»). */
function suggestedStep(rows: TimelineRow[]): QuestionnaireStep {
  const cur = rows.find((r) => r.state === 'current');
  if (!cur) return 'approved';
  return cur.id === 'outcome' ? 'approved' : (cur.id as QuestionnaireStep);
}

function RecordSheet({ q, rows, onClose }: { q: InsuranceQuestionnaire; rows: TimelineRow[]; onClose: () => void }) {
  const t = useTranslations('insurance.status');
  const locale = useLocale() as Locale;
  const advance = useAdvanceQuestionnaire(q.id);
  const [step, setStep] = useState<QuestionnaireStep>(suggestedStep(rows));
  const [parts, setParts] = useState<DateParts | null>(toParts(today(), locale));

  const save = () => {
    if (!parts) return;
    advance.mutate({ status: step, occurred_on: toApiDate(partsToDate(parts, locale)) }, { onSuccess: onClose });
  };

  return (
    <AppSheet
      open
      onClose={onClose}
      size="full"
      title={t('record.title')}
      footer={
        <div className="ins-sheet-btns">
          <SecondaryButton block={false} onClick={onClose}>
            {t('record.cancel')}
          </SecondaryButton>
          <PrimaryButton block={false} disabled={!parts} loading={advance.isPending} onClick={save}>
            {t('record.save')}
          </PrimaryButton>
        </div>
      }
    >
      <div className="ins-form">
        <p className="ins-hint">{t('record.hint')}</p>
        <h3 className="ins-label">{t('record.step')}</h3>
        <ChipGroup label={t('record.step')}>
          {STEPS.map((s) => (
            <PillChip key={s} pressed={step === s} onPressedChange={() => setStep(s)}>
              {t(`state.${s}`)}
            </PillChip>
          ))}
        </ChipGroup>
        <h3 className="ins-label">{t('record.date')}</h3>
        <CalendarPicker value={parts} onSelect={setParts} />
        {advance.isError ? (
          <p className="ins-problem" role="alert">
            {t('record.error')}
          </p>
        ) : null}
      </div>
    </AppSheet>
  );
}

function QuestionnaireCard({ q }: { q: InsuranceQuestionnaire }) {
  const t = useTranslations('insurance.status');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const [recording, setRecording] = useState(false);
  const isDraft = q.status === 'draft';
  const rows = buildTimeline(q.timeline);
  const steps = rows.map((r) => {
    const on = r.on ? formatDayMonth(fromApiDate(r.on), locale) : null;
    const title = r.id === 'outcome' && r.status ? t(`outcome.${r.status}`) : t(`steps.${r.id}`);
    return { id: r.id, title, meta: on, state: r.state, tone: r.state === 'current' ? ('warm' as const) : undefined };
  });

  return (
    <section className="nb-card ins-card" aria-labelledby={`inss-${q.id}`}>
      <div className="ins-sec-head">
        <h2 id={`inss-${q.id}`} className="ins-h">
          {t('questionnaireTitle', { purpose: t(`purpose.${q.purpose}`) })}
        </h2>
        <StatusPill tone={STATE_TONE[q.status]}>{t(`state.${q.status}`)}</StatusPill>
      </div>
      {isDraft ? (
        <>
          <p className="ins-body">{t('draftHint')}</p>
          <PrimaryButton icon="fileDoc" onClick={() => router.push('/insurance/questionnaire')}>
            {t('continue')}
          </PrimaryButton>
        </>
      ) : (
        <>
          <StepTimeline
            steps={steps}
            label={t('timelineLabel')}
            stateLabels={{ done: t('stateLabels.done'), current: t('stateLabels.current'), todo: t('stateLabels.todo') }}
            marker="dot"
            locale={locale}
          />
          <SecondaryButton icon="plus" onClick={() => setRecording(true)}>
            {t('record.title')}
          </SecondaryButton>
          {recording ? <RecordSheet q={q} rows={rows} onClose={() => setRecording(false)} /> : null}
        </>
      )}
    </section>
  );
}

function OpenClaims() {
  const t = useTranslations('insurance.status.claims');
  const tc = useTranslations('insurance');
  const locale = useLocale() as Locale;
  const claims = useRecentClaims();
  const open = (claims.data?.claims ?? []).filter((c) => c.status !== 'paid' && c.status !== 'rejected').slice(0, 3);
  if (claims.isPending || claims.isError) return null;
  return (
    <section className="ins-sec" aria-labelledby="inss-claims-h">
      <div className="ins-sec-head">
        <h2 id="inss-claims-h" className="ins-label">
          {t('title')}
        </h2>
        <Link href="/insurance/claims" className="ins-more">
          {t('all')}
        </Link>
      </div>
      {open.length === 0 ? (
        <p className="nb-card ins-empty">{t('empty')}</p>
      ) : (
        <div className="ins-claims">
          {open.map((c) => (
            <Link key={c.id} href={`/insurance/claims/${c.id}`} className="nb-card ins-claim">
              <span className="ins-claim-text">
                <span className="ins-claim-t">{c.title ?? tc(`costTypes.${costTypeKey(c.costType)}`)}</span>
                <span className="ins-claim-s">
                  {c.needsAction ? t('needsAction') : formatDayMonth(fromApiDate(c.serviceDate), locale)}
                </span>
              </span>
              <StatusPill tone={CLAIM_TONE[c.status]}>{tc(`claimStatus.${c.status}`)}</StatusPill>
            </Link>
          ))}
        </div>
      )}
    </section>
  );
}

/**
 * Request status (CB-INS-05, nbl_Ins_Status) at `/insurance/status`: her questionnaires with a four-row timeline she
 * fills in herself (sent → first review → trusted doctor → result), plus the open claims. Ritme never asks the insurer.
 * A back-header screen, no bottom nav.
 */
export function InsuranceStatusPage() {
  const t = useTranslations('insurance');
  const router = useRouter();
  const list = useQuestionnaires();

  let body;
  if (list.isPending) {
    body = (
      <SkeletonGroup label={t('status.loading')} className="ins-skel">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (list.isError) {
    body = (
      <EmptyState
        icon="shield"
        title={t('status.error')}
        action={
          <PrimaryButton icon="refresh" loading={list.isFetching} onClick={() => void list.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  } else {
    const items = list.data ?? [];
    body = (
      <>
        <p className="ins-intro">{t('status.intro')}</p>
        {items.length === 0 ? (
          <EmptyState
            icon="shield"
            title={t('status.empty')}
            action={
              <PrimaryButton icon="fileDoc" onClick={() => router.push('/insurance/questionnaire')}>
                {t('status.begin')}
              </PrimaryButton>
            }
          />
        ) : (
          items.map((q) => <QuestionnaireCard key={q.id} q={q} />)
        )}
        <OpenClaims />
      </>
    );
  }

  return (
    <div className="view ins-page">
      <SkyLayer />
      <div className="scroll ins-scroll">
        <ScreenHeader title={t('status.title')} onBack={() => router.push('/insurance')} backLabel={t('back')} />
        {body}
      </div>
    </div>
  );
}
