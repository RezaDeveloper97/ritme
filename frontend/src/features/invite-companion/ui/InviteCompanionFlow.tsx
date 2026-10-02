'use client';

import { useLocale, useTranslations } from 'next-intl';
import { type ReactNode, useId, useState } from 'react';

import {
  type CompanionSection,
  type CreatedCompanion,
  companionName,
  FamilyStrip,
  grantsByLevel,
  useCreateCompanion,
} from '@/entities/companion';
import { useUserProfile } from '@/entities/user';
import { getApiErrorMessage, getApiErrorStatus } from '@/shared/api';
import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import {
  Icon,
  InfoNote,
  PrimaryButton,
  ProgressSteps,
  RadioCardGroup,
  ScreenHeader,
  SecondaryButton,
  SkyLayer,
  StatusPill,
} from '@/shared/ui';

import {
  draftToInput,
  initialDraft,
  type InviteDraft,
  type InviteStep,
  nextStep,
  numberedSteps,
  phoneProblem,
  previousStep,
} from '../model/draft';
import { AccessEditor } from './AccessEditor';
import { InviteCodeCard } from './InviteCodeCard';

interface InviteCompanionFlowProps {
  /** Leave the wizard (header back on step 1, «برگشت به همدم‌ها»). */
  onExit: () => void;
}

/** A 422/429/503 carries a localized message from the API; anything else gets the generic copy. */
function createErrorText(error: unknown, fallback: string): string {
  const status = getApiErrorStatus(error);
  if (status === 422 || status === 429 || status === 503) return getApiErrorMessage(error) ?? fallback;
  return fallback;
}

/**
 * «افزودن همدم» (`/companions/new`, B-N4-04): Hamdam_Type → Hamdam_Access →
 * Hamdam_Children (spouse only) → Hamdam_Invite → Hamdam_Done. The invite is
 * created on the invite step (POST /companions); its one-time code is shown
 * there and lives only in this component's state.
 */
export function InviteCompanionFlow({ onExit }: InviteCompanionFlowProps) {
  const t = useTranslations('companions');
  const [step, setStep] = useState<InviteStep>('type');
  const [draft, setDraft] = useState<InviteDraft>(initialDraft);
  const [created, setCreated] = useState<CreatedCompanion | null>(null);
  const create = useCreateCompanion();
  const locale = useLocale() as Locale;

  const steps = numberedSteps(draft.type);
  const index = steps.indexOf(step);
  const numbered = index >= 0;

  const back = () => {
    // Once the invite exists, going back would only create a second one.
    if (created) return onExit();
    const prev = previousStep(step, draft.type);
    if (prev) setStep(prev);
    else onExit();
  };

  const stepLabel = numbered
    ? t(step === 'children' ? 'flow.stepSpouse' : 'flow.step', {
        current: formatNumber(index + 1, locale),
        total: formatNumber(steps.length, locale),
      })
    : undefined;

  let title: string | undefined;
  if (numbered) title = t('flow.title');
  else if (step === 'invite') title = t('flow.invite.title');

  return (
    <div className="view cmp-page">
      <SkyLayer />
      <div className="scroll cmp-scroll is-form">
        <ScreenHeader
          title={title ?? ''}
          subtitle={stepLabel}
          onBack={step === 'done' ? onExit : back}
          backLabel={t('back')}
          center={step === 'done' ? <span className="nb-hdr-titles" /> : undefined}
        />
        {numbered ? (
          <ProgressSteps total={steps.length} current={index + 1} label={stepLabel ?? ''} className="cmp-steps" />
        ) : null}

        {step === 'type' ? (
          <TypeStep draft={draft} setDraft={setDraft} onNext={() => setStep(nextStep('type', draft.type))} />
        ) : null}
        {step === 'access' ? (
          <AccessStep draft={draft} setDraft={setDraft} onNext={() => setStep(nextStep('access', draft.type))} />
        ) : null}
        {step === 'children' ? <ChildrenStep onNext={() => setStep('invite')} /> : null}
        {step === 'invite' ? (
          <InviteStepView
            draft={draft}
            setDraft={setDraft}
            created={created}
            pending={create.isPending}
            error={create.isError ? createErrorText(create.error, t('flow.invite.error')) : null}
            onEdit={() => setStep('type')}
            onCreate={(withPhone) => {
              const input = draftToInput(draft, withPhone);
              if (!input || create.isPending) return;
              create.mutate(input, { onSuccess: setCreated });
            }}
            onFinish={() => setStep('done')}
          />
        ) : null}
        {step === 'done' && created ? <DoneStep created={created} onExit={onExit} /> : null}
      </div>
    </div>
  );
}

interface StepProps {
  draft: InviteDraft;
  setDraft: (update: (d: InviteDraft) => InviteDraft) => void;
  onNext: () => void;
}

function TypeStep({ draft, setDraft, onNext }: StepProps) {
  const t = useTranslations('companions');
  return (
    <>
      <div className="cmp-body">
        <h2 className="cmp-question">{t('flow.type.question')}</h2>
        <RadioCardGroup
          label={t('flow.type.label')}
          className="cmp-types"
          value={draft.type}
          onChange={(type) => setDraft((d) => ({ ...d, type }))}
          options={[
            { value: 'partner', title: t('types.partner'), description: t('typeDesc.partner'), icon: 'heart', iconTone: 'bloom' },
            { value: 'spouse', title: t('types.spouse'), description: t('typeDesc.spouse'), icon: 'home', iconTone: 'bloom' },
          ]}
        />
      </div>
      <div className="cmp-footer">
        <PrimaryButton disabled={!draft.type} onClick={onNext}>
          {t('flow.next')}
        </PrimaryButton>
      </div>
    </>
  );
}

function AccessStep({ draft, setDraft, onNext }: StepProps) {
  const t = useTranslations('companions');
  return (
    <>
      <div className="cmp-body">
        <div className="cmp-intro">
          <h2 className="cmp-question">{t('flow.access.question')}</h2>
          <p className="cmp-lead">{t('flow.access.lead')}</p>
        </div>
        <AccessEditor value={draft.grants} onChange={(grants) => setDraft((d) => ({ ...d, grants }))} />
        <InfoNote className="cmp-note">{t('flow.access.note')}</InfoNote>
      </div>
      <div className="cmp-footer">
        <PrimaryButton onClick={onNext}>{t('flow.next')}</PrimaryButton>
      </div>
    </>
  );
}

/** Spouse only. Children arrive with B-N5-02 — until then the step explains and moves on. */
function ChildrenStep({ onNext }: { onNext: () => void }) {
  const t = useTranslations('companions');
  return (
    <>
      <div className="cmp-body">
        <div className="cmp-intro">
          <h2 className="cmp-question">{t('flow.children.question')}</h2>
          <p className="cmp-lead">{t('flow.children.lead')}</p>
        </div>
        <div className="nb-card cmp-soon">
          <span className="cmp-soon-icon" aria-hidden>
            <Icon name="sprout" size={22} />
          </span>
          <div className="cmp-soon-text">
            <b className="cmp-soon-title">{t('flow.children.soonTitle')}</b>
            <p className="cmp-soon-body">{t('flow.children.soonBody')}</p>
          </div>
        </div>
        <button type="button" className="cmp-add-child" disabled aria-disabled>
          <Icon name="plus" size={18} />
          {t('flow.children.add')}
          <StatusPill tone="neutral">{t('flow.children.soon')}</StatusPill>
        </button>
        <InfoNote className="cmp-note">{t('flow.children.partnerNote')}</InfoNote>
      </div>
      <div className="cmp-footer">
        <PrimaryButton onClick={onNext}>{t('flow.next')}</PrimaryButton>
      </div>
    </>
  );
}

interface InviteStepViewProps {
  draft: InviteDraft;
  setDraft: StepProps['setDraft'];
  created: CreatedCompanion | null;
  pending: boolean;
  error: string | null;
  onEdit: () => void;
  onCreate: (withPhone: boolean) => void;
  onFinish: () => void;
}

function InviteStepView({ draft, setDraft, created, pending, error, onEdit, onCreate, onFinish }: InviteStepViewProps) {
  const t = useTranslations('companions');
  const nameId = useId();
  const phoneId = useId();
  const phoneErrId = useId();
  const [touched, setTouched] = useState(false);
  const badPhone = phoneProblem(draft.phone);
  const showPhoneError = badPhone && touched;
  const locked = created !== null || pending;
  const { edit, view } = grantsByLevel(draft.grants);
  const list = (sections: CompanionSection[]) =>
    sections.length ? sections.map((s) => t(`sectionsShort.${s}`)).join(t('listSeparator')) : t('flow.invite.none');

  const send = () => {
    setTouched(true);
    if (badPhone) return;
    onCreate(true);
  };

  let status: ReactNode = null;
  if (created) {
    if (created.invite.smsSent && created.invite.phone) {
      status = t.rich('flow.invite.smsSent', {
        phone: created.invite.phone,
        ltr: (chunks) => <bdi dir="ltr">{chunks}</bdi>,
      });
    }
    else if (created.invite.phone) status = t('flow.invite.smsNotSent');
    else status = t('flow.invite.codeOnly');
  }

  return (
    <>
      <div className="cmp-body">
        <div className="cmp-intro">
          <h2 className="cmp-question">{t('flow.invite.heading')}</h2>
          <p className="cmp-lead">{t('flow.invite.lead')}</p>
        </div>

        <div className="cmp-field">
          <label htmlFor={nameId} className="cmp-field-label">
            {t('flow.invite.name')}
          </label>
          <input
            id={nameId}
            className="cmp-input"
            value={draft.name}
            maxLength={100}
            autoComplete="off"
            placeholder={t('flow.invite.namePlaceholder')}
            readOnly={locked}
            onChange={(e) => setDraft((d) => ({ ...d, name: e.target.value }))}
          />
        </div>
        <div className="cmp-field">
          <label htmlFor={phoneId} className="cmp-field-label">
            {t('flow.invite.phone')}
          </label>
          <input
            id={phoneId}
            className="cmp-input is-phone"
            type="tel"
            inputMode="tel"
            dir="ltr"
            autoComplete="off"
            maxLength={20}
            value={draft.phone}
            placeholder={t('flow.invite.phonePlaceholder')}
            readOnly={locked}
            aria-invalid={showPhoneError || undefined}
            aria-describedby={showPhoneError ? phoneErrId : undefined}
            onBlur={() => setTouched(true)}
            onChange={(e) => setDraft((d) => ({ ...d, phone: e.target.value }))}
          />
          {showPhoneError ? (
            <p id={phoneErrId} className="cmp-error" role="alert">
              {t('flow.invite.phoneInvalid')}
            </p>
          ) : null}
        </div>

        <div className="cmp-or" aria-hidden>
          <span>{t('flow.invite.or')}</span>
        </div>

        <InviteCodeCard
          invite={created?.invite ?? null}
          lead={t('flow.invite.codeLead')}
          action={
            <SecondaryButton icon="copy" loading={pending} disabled={locked} onClick={() => onCreate(false)}>
              {t('flow.invite.makeCode')}
            </SecondaryButton>
          }
        />

        {status ? (
          <p className="cmp-status" role="status">
            <Icon name="checkCircle" size={16} />
            {status}
          </p>
        ) : null}
        {error && !created ? (
          <p className="cmp-error" role="alert">
            {error}
          </p>
        ) : null}

        <section className="nb-card cmp-summary" aria-labelledby="cmp-summary-title">
          <div className="cmp-summary-head">
            <h3 id="cmp-summary-title" className="cmp-summary-title">
              {t('flow.invite.summary')}
            </h3>
            {created ? null : (
              <button type="button" className="cmp-link" onClick={onEdit}>
                {t('flow.invite.edit')}
              </button>
            )}
          </div>
          <dl className="cmp-summary-rows">
            <div className="cmp-summary-row">
              <dt>{t('flow.invite.relation')}</dt>
              <dd>{draft.type ? t(`types.${draft.type}`) : t('flow.invite.none')}</dd>
            </div>
            <div className="cmp-summary-row">
              <dt>{t('levels.edit')}</dt>
              <dd>{list(edit)}</dd>
            </div>
            <div className="cmp-summary-row">
              <dt>{t('levels.view')}</dt>
              <dd>{list(view)}</dd>
            </div>
            {draft.type === 'spouse' ? (
              <div className="cmp-summary-row">
                <dt>{t('flow.invite.children')}</dt>
                <dd>{t('flow.invite.none')}</dd>
              </div>
            ) : null}
          </dl>
        </section>
      </div>
      <div className="cmp-footer">
        {created ? (
          <PrimaryButton onClick={onFinish}>{t('flow.invite.finish')}</PrimaryButton>
        ) : (
          <PrimaryButton icon="send" loading={pending} onClick={send}>
            {t('flow.invite.send')}
          </PrimaryButton>
        )}
      </div>
    </>
  );
}

function DoneStep({ created, onExit }: { created: CreatedCompanion; onExit: () => void }) {
  const t = useTranslations('companions');
  const profile = useUserProfile();
  const { companion, invite } = created;
  const name = companionName(companion);
  const sent = invite.smsSent;
  let title: string;
  if (sent) title = name ? t('flow.done.titleSent', { name }) : t('flow.done.titleSentUnnamed');
  else title = name ? t('flow.done.titleCode', { name }) : t('flow.done.titleCodeUnnamed');

  return (
    <>
      <div className="cmp-body cmp-done">
        <span className="cmp-done-badge" aria-hidden>
          <Icon name="check" size={40} strokeWidth={2.4} />
        </span>
        <h2 className="cmp-done-title">{title}</h2>
        <p className="cmp-done-lead">
          {companion.type === 'spouse' ? t('flow.done.leadSpouse') : t('flow.done.leadPartner')}
        </p>
        {companion.type === 'spouse' ? (
          <FamilyStrip size="lg" selfName={profile.data?.name ?? null} companionName={name ?? t('unnamed')} />
        ) : null}
        <InfoNote className="cmp-note">{t('flow.done.note')}</InfoNote>
      </div>
      <div className="cmp-footer">
        <PrimaryButton onClick={onExit}>{t('flow.done.back')}</PrimaryButton>
      </div>
    </>
  );
}
