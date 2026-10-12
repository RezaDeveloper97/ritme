'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  CLAIM_TONE,
  type ClaimStatus,
  costTypeKey,
  type InsuranceClaimDetail,
  useAddClaimEvent,
  useAttachClaimDocument,
  useClaim,
  useDeleteClaim,
  useDetachClaimDocument,
  useExactAmount,
} from '@/entities/insurance';
import { KIND_LOOK, type TimelineItemKind } from '@/entities/health-record';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatNumber, fromApiDate, toApiDate, today } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import {
  EmptyState,
  Icon,
  IconCircle,
  InfoNote,
  LocaleNumberField,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
  StepTimeline,
} from '@/shared/ui';
import { ClaimDocPicker } from '@/widgets/claim-documents';

import { buildSteps, nextStatuses, stepProblem, type StepDraft, stepToPayload } from '../model/steps';

const MAX_NOTE = 500;
const MAX_AMOUNT = 999_999_999;

function StepSheet({ claim, initial, onClose }: { claim: InsuranceClaimDetail; initial: ClaimStatus; onClose: () => void }) {
  const t = useTranslations('insurance');
  const tn = useTranslations('insurance.claim');
  const locale = useLocale() as Locale;
  const add = useAddClaimEvent(claim.id);
  const [draft, setDraft] = useState<StepDraft>({ status: initial, occurredOn: toApiDate(today()), amount: undefined, note: '' });
  const [touched, setTouched] = useState(false);
  const problem = stepProblem(draft);
  const appeal = draft.status === 'appealed';

  const submit = () => {
    setTouched(true);
    if (problem) return;
    add.mutate(stepToPayload(draft), { onSuccess: onClose });
  };

  return (
    <AppSheet
      open
      onClose={onClose}
      size="full"
      title={appeal ? tn('appealTitle') : tn('stepTitle')}
      footer={
        <div className="ins-sheet-btns">
          <SecondaryButton block={false} onClick={onClose}>
            {tn('cancel')}
          </SecondaryButton>
          <PrimaryButton block={false} loading={add.isPending} onClick={submit}>
            {tn('save')}
          </PrimaryButton>
        </div>
      }
    >
      <div className="ins-form">
        <p className="ins-hint">{tn('stepHint')}</p>
        <div className="ins-chips" role="group" aria-label={tn('stepStatus')}>
          {nextStatuses(claim.status).map((s) => (
            <PillChip key={s} pressed={draft.status === s} onPressedChange={() => setDraft((d) => ({ ...d, status: s }))}>
              {t(`claimStatus.${s}`)}
            </PillChip>
          ))}
        </div>
        {draft.status === 'paid' ? (
          <>
            <LocaleNumberField label={tn('paidAmount')} locale={locale} value={draft.amount} max={MAX_AMOUNT} onChange={(v) => setDraft((d) => ({ ...d, amount: v }))} />
            {touched && problem ? (
              <p className="ins-problem" role="alert">
                {tn('problems.amount')}
              </p>
            ) : null}
          </>
        ) : null}
        <label className="fld-label">
          <span className="fld-label-t">{appeal ? tn('appealNote') : tn('stepNote')}</span>
          <textarea
            className="field fld-input ins-textarea"
            rows={4}
            maxLength={MAX_NOTE}
            value={draft.note}
            onChange={(e) => setDraft((d) => ({ ...d, note: e.target.value }))}
          />
        </label>
        {add.isError ? (
          <p className="ins-problem" role="alert">
            {tn('saveError')}
          </p>
        ) : null}
        <InfoNote icon="lock">{tn('selfTrack')}</InfoNote>
      </div>
    </AppSheet>
  );
}

function Body({ claim }: { claim: InsuranceClaimDetail }) {
  const t = useTranslations('insurance');
  const tn = useTranslations('insurance.claim');
  const tk = useTranslations('record.kinds');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const exact = useExactAmount();
  const attach = useAttachClaimDocument(claim.id);
  const detach = useDetachClaimDocument(claim.id);
  const del = useDeleteClaim(claim.id);
  const [sheet, setSheet] = useState<ClaimStatus | null>(null);
  const [picker, setPicker] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const p = claim.payable;
  // A claim of a past policy is read-only (policy history): no steps, documents or delete.
  const ro = claim.readOnly;
  const steps = buildSteps(claim.events);
  const current = steps.find((s) => s.state === 'current');
  const waiting = claim.status === 'docs_requested';
  const dayOf = (iso: string) => formatDayMonth(fromApiDate(iso), locale);
  const kindOf = (kind: string): TimelineItemKind => (kind in KIND_LOOK ? (kind as TimelineItemKind) : 'other');

  return (
    <>
      <section className="nb-card ins-pay" aria-labelledby="ins-pay-h">
        <div className="ins-pay-top">
          <div className="ins-pay-main">
            <h2 id="ins-pay-h" className="ins-pay-l">
              {claim.status === 'paid' && claim.paidAmount !== null ? tn('paidLabel') : tn('payableLabel')}
            </h2>
            <p className="ins-pay-n">{exact(claim.status === 'paid' && claim.paidAmount !== null ? claim.paidAmount : p.known ? p.payable : claim.amount)}</p>
          </div>
          <StatusPill tone={CLAIM_TONE[claim.status]}>{t(`claimStatus.${claim.status}`)}</StatusPill>
        </div>
        <dl className="ins-pay-rows">
          <div className="ins-pay-row">
            <dt>{tn('invoice')}</dt>
            <dd>{exact(p.invoice || claim.amount)}</dd>
          </div>
          {p.known ? (
            <>
              <div className="ins-pay-row">
                <dt>{tn('overTariff')}</dt>
                <dd>{exact(p.overTariff)}</dd>
              </div>
              <div className="ins-pay-row">
                <dt>{tn('deductible', { pct: formatNumber(p.deductiblePct, locale) })}</dt>
                <dd className="is-neg">−{exact(p.deductible)}</dd>
              </div>
              <div className="ins-pay-row">
                <dt>{tn('yourShare')}</dt>
                <dd>{exact(p.userShare)}</dd>
              </div>
            </>
          ) : null}
        </dl>
        <p className="ins-hint">{p.known ? (p.capped ? tn('cappedHint') : tn('estimateHint')) : tn('unknownHint')}</p>
      </section>

      {waiting || claim.documents.length > 0 ? (
        <section className="ins-sec" aria-labelledby="ins-c-docs">
          <h2 id="ins-c-docs" className="ins-label">
            {waiting ? tn('missingTitle') : tn('docsTitle')}
          </h2>
          <div className="nb-card ins-card">
            {waiting ? (
              <div className="ins-msg">
                <IconCircle icon="chat" tone="brand" />
                <p className="ins-body">
                  {current?.occurredOn ? <strong>{tn('stepOn', { date: dayOf(current.occurredOn) })}</strong> : null}{' '}
                  {current?.note || tn('missingDefault')}
                </p>
              </div>
            ) : null}
            {claim.documents.length > 0 ? (
              <ul className="ins-docs">
                {claim.documents.map((d) => {
                  const kind = kindOf(d.kind);
                  return (
                    <li key={d.documentId} className="ins-slot">
                      <IconCircle icon={KIND_LOOK[kind].icon} tone="data" />
                      <span className="ins-claim-text">
                        <span className="ins-claim-t">{d.title ?? tk(kind)}</span>
                        <span className="ins-claim-s">{d.label ? tn(`labels.${d.label}`) : tk(kind)}</span>
                      </span>
                      {ro ? null : (
                        <button
                          type="button"
                          className="ins-icon-btn"
                          aria-label={tn('detach')}
                          disabled={detach.isPending}
                          onClick={() => detach.mutate(d.documentId)}
                        >
                          <Icon name="x" size={16} />
                        </button>
                      )}
                    </li>
                  );
                })}
              </ul>
            ) : null}
            {ro ? null : (
              <PrimaryButton icon="fileDoc" loading={attach.isPending} onClick={() => setPicker(true)}>
                {tn('pickFromRecord')}
              </PrimaryButton>
            )}
            {attach.isError || detach.isError ? (
              <p className="ins-problem" role="alert">
                {tn('saveError')}
              </p>
            ) : null}
          </div>
        </section>
      ) : ro ? null : (
        <SecondaryButton icon="fileDoc" onClick={() => setPicker(true)}>
          {tn('attachDoc')}
        </SecondaryButton>
      )}

      <section className="ins-sec" aria-labelledby="ins-c-flow">
        <h2 id="ins-c-flow" className="ins-label">
          {tn('flowTitle')}
        </h2>
        <div className="nb-card ins-card">
          <StepTimeline
            marker="dot"
            locale={locale}
            label={tn('flowLabel')}
            stateLabels={{ done: tn('state.done'), current: tn('state.current'), todo: tn('state.todo') }}
            steps={steps.map((s) => ({
              id: s.id,
              state: s.state,
              tone: s.tone,
              title: t(`claimStatus.${s.status}`),
              meta:
                s.state === 'todo'
                  ? tn('payHint')
                  : [
                      s.occurredOn ? dayOf(s.occurredOn) : null,
                      s.deadlineOn ? tn('deadlineUntil', { date: dayOf(s.deadlineOn) }) : null,
                      s.status === 'paid' && s.amount ? exact(s.amount) : null,
                      s.note || null,
                    ]
                      .filter(Boolean)
                      .join(' · '),
            }))}
          />
          {ro ? null : (
            <SecondaryButton icon="plus" onClick={() => setSheet(claim.status === 'submitted' ? 'reviewing' : nextStatuses(claim.status)[0])}>
              {tn('advance')}
            </SecondaryButton>
          )}
        </div>
      </section>

      {!ro && (claim.status === 'rejected' || claim.status === 'paid') ? (
        <SecondaryButton variant="text" onClick={() => setSheet('appealed')}>
          {tn('appeal')}
        </SecondaryButton>
      ) : null}

      {ro ? <InfoNote icon="history">{tn('readOnly')}</InfoNote> : null}
      <InfoNote icon="lock">{tn('selfTrack')}</InfoNote>

      {ro ? null : confirmDelete ? (
        <div className="nb-card ins-card">
          <p className="ins-body">{tn('deleteAsk')}</p>
          <div className="ins-sheet-btns">
            <SecondaryButton block={false} onClick={() => setConfirmDelete(false)}>
              {tn('cancel')}
            </SecondaryButton>
            <PrimaryButton block={false} loading={del.isPending} onClick={() => del.mutate(undefined, { onSuccess: () => router.replace('/insurance/claims') })}>
              {tn('deleteYes')}
            </PrimaryButton>
          </div>
          {del.isError ? (
            <p className="ins-problem" role="alert">
              {tn('saveError')}
            </p>
          ) : null}
        </div>
      ) : (
        <SecondaryButton variant="text" icon="trash" onClick={() => setConfirmDelete(true)}>
          {tn('delete')}
        </SecondaryButton>
      )}

      {sheet ? <StepSheet claim={claim} initial={sheet} onClose={() => setSheet(null)} /> : null}
      <ClaimDocPicker
        open={picker}
        onClose={() => setPicker(false)}
        takenIds={claim.documents.map((d) => d.documentId)}
        onPick={(doc) => attach.mutate({ documentId: doc.id, label: 'other' })}
      />
    </>
  );
}

/**
 * Claim detail (CB-INS-04, nbl_Ins_ClaimDetail) at `/insurance/claims/{id}`: the payable estimate with its breakdown,
 * the documents she attached from her record, and the timeline she advances herself. Self-tracked — nothing is sent
 * to an insurer. Back-header screen, no bottom nav.
 */
export function InsuranceClaimPage({ id }: { id: number }) {
  const t = useTranslations('insurance');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const query = useClaim(id);
  const claim = query.data;

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="ins-skel">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError || !claim) {
    body = (
      <EmptyState
        icon="fileDoc"
        title={t('claim.error')}
        action={
          <PrimaryButton icon="refresh" loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  } else {
    body = <Body claim={claim} />;
  }

  const subtitle = claim
    ? [claim.title ?? t(`costTypes.${costTypeKey(claim.costType)}`), formatDayMonth(fromApiDate(claim.serviceDate), locale)].join(' · ')
    : undefined;

  return (
    <div className="view ins-page">
      <SkyLayer />
      <div className="scroll ins-scroll">
        <ScreenHeader title={t('claim.title')} subtitle={subtitle} onBack={() => router.push('/insurance/claims')} backLabel={t('back')} />
        {body}
      </div>
    </div>
  );
}
