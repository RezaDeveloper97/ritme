'use client';

import { useQueryClient } from '@tanstack/react-query';
import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';
import { useId, useRef, useState, type KeyboardEvent } from 'react';

import { useStopContraception } from '@/entities/contraception';
import { useDeactivatePregnancy } from '@/entities/pregnancy';
import {
  lifeStageKeys,
  useLifeStage,
  useUpdateLifeStage,
  writeLifeModeHint,
  type LifeMode,
  type LifeStage,
} from '@/entities/user';
import { useDirection, useRouter } from '@/shared/i18n';
import { AppSheet } from '@/shared/sheet';
import {
  EmptyState,
  Icon,
  IconCircle,
  ListRow,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  Switch,
} from '@/shared/ui';

import { MODE_CARDS, needsConfirm, planSwitch, type ModeCardDef, type SwitchPlan } from '../model/modes';

type Confirmable = Extract<SwitchPlan, { kind: 'enterPregnancy' | 'leavePregnancy' }>;

/** Next radio index for an arrow / Home / End key (APG radiogroup), wrapping; `null` = not a nav key. */
function radioStep(key: string, index: number, count: number, rtl: boolean): number | null {
  if (key === (rtl ? 'ArrowLeft' : 'ArrowRight') || key === 'ArrowDown') return (index + 1) % count;
  if (key === (rtl ? 'ArrowRight' : 'ArrowLeft') || key === 'ArrowUp') return (index - 1 + count) % count;
  if (key === 'Home') return 0;
  if (key === 'End') return count - 1;
  return null;
}

/**
 * «مرحله زندگی» (`/profile/mode`, B-N2-03, `nbl_Me_Mode` / `nbd_Me_Mode`): the
 * six life-stage modes as radio cards (tab + log hint chips), the IVF/IUI switch
 * inside the TTC card, «track contraception» and the «data kept» note. Reads and
 * writes `GET|PUT /profile/life-stage`; pregnancy goes through the pregnancy
 * endpoints (setup to enter, deactivate to leave). In pregnancy mode a calm exit
 * row opens the loss path (`/loss`, CB-LOSS-02).
 */
export function ModePage() {
  const t = useTranslations('me.mode');
  const router = useRouter();
  const query = useLifeStage();

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="mode-list">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError || !query.data) {
    body = (
      <EmptyState
        icon="modeRing"
        title={t('error.title')}
        body={t('error.body')}
        action={
          <PrimaryButton icon="refresh" loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('error.retry')}
          </PrimaryButton>
        }
      />
    );
  } else {
    body = <ModeBody stage={query.data} />;
  }

  return (
    <div className="view mode-page">
      <SkyLayer />
      <div className="scroll mode-scroll">
        <ScreenHeader
          title={t('title')}
          subtitle={t('subtitle')}
          onBack={() => router.push('/profile')}
          backLabel={t('back')}
        />
        {body}
      </div>
    </div>
  );
}

function ModeBody({ stage }: { stage: LifeStage }) {
  const t = useTranslations('me.mode');
  const tc = useTranslations('contraception.mode');
  const router = useRouter();
  const rtl = useDirection() === 'rtl';
  const queryClient = useQueryClient();
  const update = useUpdateLifeStage();
  const deactivate = useDeactivatePregnancy();
  const stopContraception = useStopContraception();
  const groupId = useId();
  const refs = useRef<Array<HTMLButtonElement | null>>([]);
  const [confirm, setConfirm] = useState<Confirmable | null>(null);
  const [failed, setFailed] = useState(false);
  const [savedMode, setSavedMode] = useState<LifeMode | null>(null);

  const current = stage.mode;
  const busy = update.isPending || deactivate.isPending;
  const checkedIndex = Math.max(0, MODE_CARDS.findIndex((c) => c.key === current));

  // A mode reshapes the whole app (nav, home, messages, TTC layout): refetch everything.
  const afterModeChange = (mode: LifeMode) => {
    writeLifeModeHint(mode);
    void queryClient.invalidateQueries();
  };

  const run = async (plan: SwitchPlan) => {
    setFailed(false);
    setSavedMode(null);
    try {
      if (plan.kind === 'enterPregnancy') {
        // Nothing is stored yet: the setup activates the pregnancy profile and
        // stores the mode on its last step, so leaving it half-way keeps the
        // current mode (stage B-3). It skips itself when one is already active.
        router.push('/pregnancy/setup');
      } else if (plan.kind === 'leavePregnancy') {
        await deactivate.mutateAsync();
        const saved = await update.mutateAsync({ mode: plan.target });
        afterModeChange(saved.mode);
        // `/home` is mode-aware: it routes or renders the new mode's Today.
        router.replace('/home');
      } else if (plan.kind === 'store') {
        const saved = await update.mutateAsync({ mode: plan.target });
        afterModeChange(saved.mode);
        setSavedMode(saved.mode);
      }
    } catch {
      setFailed(true);
    }
  };

  const choose = (mode: LifeMode) => {
    if (busy) return;
    const plan = planSwitch(current, mode);
    if (plan.kind === 'none') return;
    if (needsConfirm(plan)) setConfirm(plan as Confirmable);
    else void run(plan);
  };

  const onKeyDown = (event: KeyboardEvent<HTMLButtonElement>, index: number) => {
    const next = radioStep(event.key, index, MODE_CARDS.length, rtl);
    if (next === null) return;
    event.preventDefault();
    refs.current[next]?.focus();
  };

  const toggle = (patch: { ivfIui?: boolean; trackContraception?: boolean }) => {
    setFailed(false);
    update.mutate(patch, { onError: () => setFailed(true) });
  };

  // CB-CONTRA-02: switching contraception on opens the method setup (its save turns
  // the flag on); off stops tracking — method + its reminders go, the pill log stays.
  const toggleContraception = (on: boolean) => {
    setFailed(false);
    if (on) {
      router.push('/contraception/setup');
      return;
    }
    stopContraception.mutate(undefined, {
      onSuccess: () => void queryClient.invalidateQueries({ queryKey: lifeStageKeys.all }),
      onError: () => setFailed(true),
    });
  };

  const confirmTarget = confirm?.kind === 'leavePregnancy' ? confirm.target : 'pregnancy';

  return (
    <>
      {failed ? (
        <p className="mode-error" role="alert">
          {t('saveError')}
        </p>
      ) : null}
      <p className="sr-only" role="status" aria-live="polite">
        {busy ? t('saving') : savedMode ? t('saved', { mode: t(`cards.${savedMode}.title`) }) : ''}
      </p>

      <div role="radiogroup" aria-label={t('title')} aria-busy={busy || undefined} className="mode-list">
        {MODE_CARDS.map((card, index) => (
          <ModeCard
            key={card.key}
            card={card}
            checked={card.key === current}
            tabIndex={index === checkedIndex ? 0 : -1}
            buttonRef={(el) => {
              refs.current[index] = el;
            }}
            idPrefix={`${groupId}-${card.key}`}
            disabled={busy}
            onSelect={() => choose(card.key)}
            onKeyDown={(event) => onKeyDown(event, index)}
          >
            {card.key === 'ttc' ? (
              <div className="mode-sub">
                <span id={`${groupId}-ivf`} className="mode-sub-label">
                  {t('ivf')}
                </span>
                <Switch
                  checked={stage.ivfIui}
                  onCheckedChange={(ivfIui) => toggle({ ivfIui })}
                  labelledBy={`${groupId}-ivf`}
                />
              </div>
            ) : null}
            {card.key === 'pregnancy' && current === 'pregnancy' ? (
              <button type="button" className="mode-loss" onClick={() => router.push('/loss')}>
                <span className="mode-loss-text">
                  <b>{t('loss.row')}</b>
                  <span>{t('loss.rowSub')}</span>
                </span>
                <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} />
              </button>
            ) : null}
          </ModeCard>
        ))}
      </div>

      <div className="mode-contra">
        <ListRow
          id={`${groupId}-contra`}
          title={t('contraception.title')}
          description={t('contraception.desc')}
          trailing={
            <Switch
              checked={stage.trackContraception}
              disabled={stopContraception.isPending}
              onCheckedChange={toggleContraception}
              labelledBy={`${groupId}-contra-title`}
            />
          }
        />
        {stage.trackContraception ? (
          <ListRow
            icon="pill"
            iconTone="brand"
            title={tc('manage')}
            description={tc('manageDesc')}
            onClick={() => router.push('/contraception')}
          />
        ) : null}
      </div>

      <div className="mode-note">
        <Icon name="info" size={17} />
        <span>{t('dataKept')}</span>
      </div>

      <AppSheet
        open={confirm !== null}
        onClose={() => (busy ? undefined : setConfirm(null))}
        size="half"
        title={confirm?.kind === 'leavePregnancy' ? t('confirm.leaveTitle') : t('confirm.pregnancyTitle')}
        footer={
          <div className="mode-confirm-btns">
            <PrimaryButton
              loading={busy}
              onClick={() => {
                const plan = confirm;
                if (!plan) return;
                void run(plan).then(() => setConfirm(null));
              }}
            >
              {confirm?.kind === 'leavePregnancy' ? t('confirm.leaveGo') : t('confirm.pregnancyGo')}
            </PrimaryButton>
            <SecondaryButton variant="text" block disabled={busy} onClick={() => setConfirm(null)}>
              {t('confirm.cancel')}
            </SecondaryButton>
          </div>
        }
      >
        <p className="mode-confirm-body">
          {confirm?.kind === 'leavePregnancy'
            ? t('confirm.leaveBody', { mode: t(`cards.${confirmTarget}.title`) })
            : t('confirm.pregnancyBody')}
        </p>
      </AppSheet>
    </>
  );
}

interface ModeCardProps {
  card: ModeCardDef;
  checked: boolean;
  tabIndex: number;
  buttonRef: (el: HTMLButtonElement | null) => void;
  idPrefix: string;
  disabled: boolean;
  onSelect: () => void;
  onKeyDown: (event: KeyboardEvent<HTMLButtonElement>) => void;
  /** Controls that belong to the card but are not part of the radio (IVF switch, loss row). */
  children?: React.ReactNode;
}

/** One mode: the radio (icon, title, description, dot + the tab / log chips) and its extras below. */
function ModeCard({ card, checked, tabIndex, buttonRef, idPrefix, disabled, onSelect, onKeyDown, children }: ModeCardProps) {
  const t = useTranslations('me.mode');
  return (
    <div className={clsx('mode-card', `nb-tone-${card.tone}`, checked && 'is-on')}>
      <button
        ref={buttonRef}
        type="button"
        role="radio"
        aria-checked={checked}
        aria-labelledby={`${idPrefix}-t`}
        aria-describedby={`${idPrefix}-d ${idPrefix}-c`}
        tabIndex={tabIndex}
        disabled={disabled && !checked}
        className="mode-radio"
        onClick={onSelect}
        onKeyDown={onKeyDown}
      >
        <span className="mode-head">
          <IconCircle icon={card.icon} tone={card.tone} size="lg" />
          <span className="mode-text">
            <b id={`${idPrefix}-t`} className="mode-title">
              {t(`cards.${card.key}.title`)}
            </b>
            <span id={`${idPrefix}-d`} className="mode-desc">
              {t(`cards.${card.key}.desc`)}
            </span>
          </span>
          <span className="mode-dot" aria-hidden />
        </span>
        <span id={`${idPrefix}-c`} className="mode-chips">
          <span className="mode-chip">{t(`cards.${card.key}.tab`)}</span>
          <span className="mode-chip">{t(`cards.${card.key}.log`)}</span>
        </span>
      </button>
      {children}
    </div>
  );
}
