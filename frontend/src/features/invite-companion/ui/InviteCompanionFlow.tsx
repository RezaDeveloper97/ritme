'use client';

import { useTranslations } from 'next-intl';
import { type ReactNode, useId, useState } from 'react';

import { useModuleEnabled } from '@/entities/app-module';
import { type CreatedCompanion, useCreateCompanion } from '@/entities/companion';
import { getApiErrorMessage, getApiErrorStatus } from '@/shared/api';
import {
  Checkbox,
  Icon,
  type IconName,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  SkyLayer,
} from '@/shared/ui';
import { plusDenialOf, upgradeHelps } from '@/shared/ui/plus-gate';

import {
  draftToInput,
  initialDraft,
  type InviteDraft,
  type InviteStep,
  nextStep,
  phoneProblem,
  previousStep,
} from '../model/draft';
import { AccessEditor } from './AccessEditor';
import { ChildrenPicker } from './ChildrenPicker';
import { CompanionPreview } from './CompanionPreview';
import { InviteCodeBox } from './InviteCodeBox';
import { ActionPicker, GoalPicker } from './PreferenceFields';

interface InviteCompanionFlowProps {
  /** Leave the wizard (back on the first step, «متوجه شدم», back from the code). */
  onExit: () => void;
  /** Inviting a companion is a Ritme Plus feature and this account does not have it (`plus.companion`). */
  plusLocked?: boolean;
  /** «ساختن کد دعوت» without Plus (known up front, or a 402 from the API): the caller opens the Plus sheet. */
  onPlusRequired?: () => void;
}

/** A 422/429/503 carries a localized message from the API; anything else gets the generic copy. */
function createErrorText(error: unknown, fallback: string): string {
  const status = getApiErrorStatus(error);
  if (status === 422 || status === 429 || status === 503) return getApiErrorMessage(error) ?? fallback;
  return fallback;
}

type SetDraft = (update: (d: InviteDraft) => InviteDraft) => void;

/**
 * «ریتمی همراه» invite (`/companions/new`, companion-v2): intro + consent →
 * goal (+ «همسرم است» → shared children) → scope (per-section access) →
 * preferred actions (+ an optional name / phone) → generating (POST
 * /companions: the grants plus the preferences) → code → preview. Making the
 * code needs Ritme Plus: without it the last CTA hands over to the caller's
 * Plus sheet and nothing is sent. The one-time code lives only in this
 * component's state.
 */
export function InviteCompanionFlow({ onExit, plusLocked = false, onPlusRequired }: InviteCompanionFlowProps) {
  const t = useTranslations('companions.flow');
  const [step, setStep] = useState<InviteStep>('intro');
  const [draft, setDraft] = useState<InviteDraft>(initialDraft);
  const [created, setCreated] = useState<CreatedCompanion | null>(null);
  const create = useCreateCompanion();
  const childrenOn = useModuleEnabled('children');

  // The children step only exists for a spouse while the «کودک» module is on (nothing to pick otherwise).
  const route = { spouse: draft.spouse && childrenOn };
  const go = (from: InviteStep) => setStep(nextStep(from, route));
  const back = () => {
    if (create.isPending) return;
    const prev = previousStep(step, route);
    if (prev) setStep(prev);
    else onExit();
  };

  const generate = () => {
    const input = draftToInput(draft);
    if (!input || create.isPending) return;
    if (plusLocked && onPlusRequired) {
      onPlusRequired();
      return;
    }
    setStep('generating');
    create.mutate(input, {
      onSuccess: (result) => {
        setCreated(result);
        setStep('code');
      },
      onError: (error) => {
        // The API gate is the authority (a stale status, a trial that just ended): back to the CTA + the Plus sheet.
        const denial = plusDenialOf(error);
        if (denial && upgradeHelps(denial) && onPlusRequired) {
          setStep('actions');
          onPlusRequired();
        }
      },
    });
  };

  let view: ReactNode = null;
  switch (step) {
    case 'intro':
      view = <IntroStep draft={draft} setDraft={setDraft} onNext={() => go('intro')} />;
      break;
    case 'goal':
      view = <GoalStep draft={draft} setDraft={setDraft} onNext={() => go('goal')} />;
      break;
    case 'children':
      view = <ChildrenStep draft={draft} setDraft={setDraft} onNext={() => go('children')} />;
      break;
    case 'scope':
      view = <ScopeStep draft={draft} setDraft={setDraft} onNext={() => go('scope')} />;
      break;
    case 'actions':
      view = <ActionsStep draft={draft} setDraft={setDraft} onNext={generate} />;
      break;
    case 'generating':
      view = (
        <GeneratingStep
          error={create.isError ? createErrorText(create.error, t('generating.error')) : null}
          onRetry={generate}
          onEdit={() => setStep('actions')}
        />
      );
      break;
    case 'code':
      view = created ? <CodeStep created={created} onPreview={() => go('code')} /> : null;
      break;
    case 'preview':
      view = <PreviewStep draft={draft} onDone={onExit} />;
      break;
  }

  return (
    <div className="view cmp-page">
      <SkyLayer />
      <div className="scroll cmp-scroll is-form">
        <ScreenHeader title={t(`header.${step}`)} onBack={back} backLabel={t('back')} />
        {view}
      </div>
    </div>
  );
}

interface StepProps {
  draft: InviteDraft;
  setDraft: SetDraft;
  onNext: () => void;
}

function StepIntro({ title, lead, center }: { title: string; lead: string; center?: boolean }) {
  return (
    <div className={center ? 'cmp2-intro is-center' : 'cmp2-intro'}>
      <h2 className="cmp2-title">{title}</h2>
      <p className="cmp-lead">{lead}</p>
    </div>
  );
}

const HOW: ReadonlyArray<{ key: 'choose' | 'code' | 'control'; icon: IconName }> = [
  { key: 'choose', icon: 'eye' },
  { key: 'code', icon: 'clock' },
  { key: 'control', icon: 'shield' },
];

function IntroStep({ draft, setDraft, onNext }: StepProps) {
  const t = useTranslations('companions.flow.intro');
  const tf = useTranslations('companions.flow');
  return (
    <>
      <div className="cmp-body">
        <span className="cmp2-hero-illus" aria-hidden>
          <Icon name="users" size={34} strokeWidth={1.8} />
        </span>
        <StepIntro title={t('title')} lead={t('lead')} center />
        <section className="cmp2-how" aria-labelledby="cmp2-how-title">
          <h3 id="cmp2-how-title" className="cmp2-label">
            {t('howTitle')}
          </h3>
          <ul className="nb-card cmp2-how-list">
            {HOW.map(({ key, icon }) => (
              <li key={key} className="cmp2-how-row">
                <span className="cmp2-how-icon" aria-hidden>
                  <Icon name={icon} size={18} />
                </span>
                <span className="cmp2-how-text">
                  <b>{t(`how.${key}.title`)}</b>
                  <span>{t(`how.${key}.body`)}</span>
                </span>
              </li>
            ))}
          </ul>
        </section>
        <Checkbox
          className="nb-card cmp2-consent"
          checked={draft.consent}
          onCheckedChange={(consent) => setDraft((d) => ({ ...d, consent }))}
          label={t('consent')}
        />
      </div>
      <div className="cmp-footer">
        <PrimaryButton disabled={!draft.consent} onClick={onNext}>
          {tf('next')}
        </PrimaryButton>
      </div>
    </>
  );
}

function GoalStep({ draft, setDraft, onNext }: StepProps) {
  const t = useTranslations('companions.flow.goal');
  const tf = useTranslations('companions.flow');
  return (
    <>
      <div className="cmp-body">
        <StepIntro title={t('title')} lead={t('lead')} />
        <GoalPicker value={draft.goal} onChange={(goal) => setDraft((d) => ({ ...d, goal }))} />
      </div>
      <div className="cmp-footer">
        <PrimaryButton onClick={onNext}>{tf('next')}</PrimaryButton>
      </div>
    </>
  );
}

/**
 * «همسرم است» only (nbl_Hamdam_Children): which of her children become shared
 * (`child_ids` on create). Optional — none picked shares nothing; it can be
 * changed later on the companion detail page.
 */
function ChildrenStep({ draft, setDraft, onNext }: StepProps) {
  const t = useTranslations('companions.flow.children');
  const tf = useTranslations('companions.flow');
  return (
    <>
      <div className="cmp-body">
        <StepIntro title={t('question')} lead={t('lead')} />
        <ChildrenPicker value={draft.childIds} onChange={(childIds) => setDraft((d) => ({ ...d, childIds }))} />
      </div>
      <div className="cmp-footer">
        <PrimaryButton onClick={onNext}>{tf('next')}</PrimaryButton>
      </div>
    </>
  );
}

function ScopeStep({ draft, setDraft, onNext }: StepProps) {
  const t = useTranslations('companions.flow.access');
  const tf = useTranslations('companions.flow');
  return (
    <>
      <div className="cmp-body">
        <StepIntro title={t('title')} lead={t('lead')} />
        <AccessEditor value={draft.grants} onChange={(grants) => setDraft((d) => ({ ...d, grants }))} />
        <p className="cmp2-privacy">
          <Icon name="shield" size={16} className="cmp2-privacy-icon" />
          <span>{t('hidden')}</span>
        </p>
        <div className="cmp2-aside is-quiet">
          <p className="cmp2-aside-text">{t('note')}</p>
        </div>
      </div>
      <div className="cmp-footer">
        <PrimaryButton onClick={onNext}>{tf('next')}</PrimaryButton>
      </div>
    </>
  );
}

function ActionsStep({ draft, setDraft, onNext }: StepProps) {
  const t = useTranslations('companions.flow.actions');
  const badPhone = phoneProblem(draft.phone);
  return (
    <>
      <div className="cmp-body">
        <StepIntro title={t('title')} lead={t('lead')} />
        <ActionPicker value={draft.actionPrefs} onChange={(actionPrefs) => setDraft((d) => ({ ...d, actionPrefs }))} />
        <OptionalContact draft={draft} setDraft={setDraft} />
      </div>
      <div className="cmp-footer">
        <PrimaryButton disabled={badPhone} onClick={onNext}>
          {t('cta')}
        </PrimaryButton>
      </div>
    </>
  );
}

/**
 * «اسم و شماره (اختیاری)»: the name she sees him by and a mobile for an SMS
 * invite — collapsed, since the code alone is enough. A typed phone must be a
 * mobile before the code is made.
 */
function OptionalContact({ draft, setDraft }: { draft: InviteDraft; setDraft: SetDraft }) {
  const t = useTranslations('companions.flow.contact');
  const baseId = useId();
  const [open, setOpen] = useState(draft.name !== '' || draft.phone !== '');
  const [touched, setTouched] = useState(false);
  const showPhoneError = phoneProblem(draft.phone) && touched;
  const panelId = `${baseId}-panel`;
  const phoneErrId = `${baseId}-err`;

  return (
    <section className="nb-card cmp2-more">
      <button
        type="button"
        className="cmp2-more-toggle"
        aria-expanded={open}
        aria-controls={panelId}
        onClick={() => setOpen((v) => !v)}
      >
        <span className="cmp2-more-text">
          <b>{t('toggle')}</b>
          <span>{t('hint')}</span>
        </span>
        <Icon name="chevronDown" size={18} className="cmp2-more-chev" />
      </button>
      {open ? (
        <div id={panelId} className="cmp2-more-panel">
          <div className="cmp-field">
            <label htmlFor={`${baseId}-name`} className="cmp-field-label">
              {t('name')}
            </label>
            <input
              id={`${baseId}-name`}
              className="cmp-input"
              value={draft.name}
              maxLength={100}
              autoComplete="off"
              placeholder={t('namePlaceholder')}
              onChange={(e) => setDraft((d) => ({ ...d, name: e.target.value }))}
            />
          </div>
          <div className="cmp-field">
            <label htmlFor={`${baseId}-phone`} className="cmp-field-label">
              {t('phone')}
            </label>
            <input
              id={`${baseId}-phone`}
              className="cmp-input is-phone"
              type="tel"
              inputMode="tel"
              dir="ltr"
              autoComplete="off"
              maxLength={20}
              value={draft.phone}
              placeholder={t('phonePlaceholder')}
              aria-invalid={showPhoneError || undefined}
              aria-describedby={showPhoneError ? phoneErrId : undefined}
              onBlur={() => setTouched(true)}
              onChange={(e) => setDraft((d) => ({ ...d, phone: e.target.value }))}
            />
            {showPhoneError ? (
              <p id={phoneErrId} className="cmp-error" role="alert">
                {t('phoneInvalid')}
              </p>
            ) : (
              <p className="cmp2-field-hint">{t('phoneHint')}</p>
            )}
          </div>
        </div>
      ) : null}
    </section>
  );
}

function GeneratingStep({ error, onRetry, onEdit }: { error: string | null; onRetry: () => void; onEdit: () => void }) {
  const t = useTranslations('companions.flow.generating');
  return (
    <>
      <div className="cmp-body cmp2-center">
        <StepIntro title={t('title')} lead={t('lead')} center />
        {error ? (
          <p className="cmp-error" role="alert">
            {error}
          </p>
        ) : (
          <>
            <span className="cmp2-spinner" role="status" aria-label={t('title')} />
            <p className="cmp-lead">{t('wait')}</p>
          </>
        )}
      </div>
      <div className="cmp-footer">
        <p className="cmp2-privacy">
          <Icon name="shield" size={16} className="cmp2-privacy-icon" />
          <span>{t('privacy')}</span>
        </p>
        {error ? (
          <>
            <PrimaryButton icon="refresh" onClick={onRetry}>
              {t('retry')}
            </PrimaryButton>
            <SecondaryButton variant="text" onClick={onEdit}>
              {t('edit')}
            </SecondaryButton>
          </>
        ) : null}
      </div>
    </>
  );
}

function CodeStep({ created, onPreview }: { created: CreatedCompanion; onPreview: () => void }) {
  const t = useTranslations('companions.flow.code');
  const { invite } = created;
  let status: ReactNode = null;
  if (invite.smsSent && invite.phone) {
    status = t.rich('smsSent', { phone: invite.phone, ltr: (chunks) => <bdi dir="ltr">{chunks}</bdi> });
  } else if (invite.phone) status = t('smsNotSent');

  return (
    <>
      <div className="cmp-body">
        <div className="cmp2-intro is-center">
          <span className="cmp2-pill">{t('ready')}</span>
          <h2 className="cmp2-title">{t('title')}</h2>
        </div>
        <InviteCodeBox invite={invite} />
        {status ? (
          <p className="cmp-status" role="status">
            <Icon name="checkCircle" size={16} />
            {status}
          </p>
        ) : null}
      </div>
      <div className="cmp-footer">
        <SecondaryButton variant="text" icon="eye" onClick={onPreview}>
          {t('preview')}
        </SecondaryButton>
      </div>
    </>
  );
}

function PreviewStep({ draft, onDone }: { draft: InviteDraft; onDone: () => void }) {
  const t = useTranslations('companions.flow.preview');
  return (
    <>
      <div className="cmp-body">
        <StepIntro title={t('title')} lead={t('lead')} center />
        <CompanionPreview grants={draft.grants} />
      </div>
      <div className="cmp-footer">
        <PrimaryButton onClick={onDone}>{t('done')}</PrimaryButton>
      </div>
    </>
  );
}
