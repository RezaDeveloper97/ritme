'use client';

import { useTranslations } from 'next-intl';
import { useId } from 'react';

import {
  COMPANION_ACTIONS,
  COMPANION_GOALS,
  type CompanionAction,
  type CompanionGoal,
  type CompanionSignal,
  type CompanionSignalScope,
} from '@/entities/companion';
import { pilot, pilotOption } from '@/shared/pilot';
import { CheckTick, IconCircle, type IconName, Switch, type Tone } from '@/shared/ui';

import { OFFERED_SIGNALS, toggleAction, toggleGoal } from '../model/draft';
import '@/shared/ui/styles/cmp.css';

/*
 * The three v2 preference editors («هدف همراهی», «چه چیزهایی را ببیند؟»,
 * «اقدام‌های ترجیحی»): used by the invite wizard and the companion detail
 * screen's «ترجیحات همراهی» section.
 */

const GOAL_LOOK: Record<CompanionGoal, { icon: IconName; tone: Tone }> = {
  cycle: { icon: 'book', tone: 'brand' },
  support: { icon: 'heart', tone: 'bloom' },
  low_energy: { icon: 'zap', tone: 'warm' },
  quality_time: { icon: 'clock', tone: 'data' },
  info: { icon: 'info', tone: 'brand' },
  unknown: { icon: 'moreH', tone: 'neutral' },
};

const SIGNAL_LOOK: Record<CompanionSignal, { icon: IconName; tone: Tone }> = {
  pain: { icon: 'symptom', tone: 'bloom' },
  mood: { icon: 'smile', tone: 'brand' },
  energy: { icon: 'zap', tone: 'warm' },
  sleep: { icon: 'moon', tone: 'data' },
};

const ACTION_LOOK: Record<CompanionAction, { icon: IconName; tone: Tone }> = {
  food: { icon: 'glass', tone: 'warm' },
  gift: { icon: 'sparkle', tone: 'bloom' },
  time: { icon: 'clock', tone: 'data' },
  calm: { icon: 'heart', tone: 'brand' },
  help: { icon: 'hand', tone: 'success' },
};

interface CheckCardsProps<T extends string> {
  label: string;
  options: readonly T[];
  value: readonly T[];
  look: Record<T, { icon: IconName; tone: Tone }>;
  title: (option: T) => string;
  description: (option: T) => string;
  onToggle: (option: T) => void;
  disabled?: boolean;
  /** Autopilot id of the group; each card is an option. */
  pilotId: string;
}

/** Many-of-many option cards (a tick box each): the goal and the action pickers. */
function CheckCards<T extends string>({
  label,
  options,
  value,
  look,
  title,
  description,
  onToggle,
  disabled,
  pilotId,
}: CheckCardsProps<T>) {
  const baseId = useId();
  return (
    <div role="group" aria-label={label} className="nb-rcards cmp2-opts" {...pilot(pilotId, 'choice')}>
      {options.map((option) => {
        const checked = value.includes(option);
        const titleId = `${baseId}-${option}-t`;
        const descId = `${baseId}-${option}-d`;
        return (
          <button
            key={option}
            type="button"
            role="checkbox"
            aria-checked={checked}
            aria-labelledby={titleId}
            aria-describedby={descId}
            disabled={disabled}
            className="nb-rcard cmp2-check-card"
            onClick={() => onToggle(option)}
            {...pilotOption(option)}
          >
            <IconCircle icon={look[option].icon} tone={look[option].tone} size="lg" />
            <span className="nb-rcard-text">
              <span id={titleId} className="nb-rcard-title">
                {title(option)}
              </span>
              <span id={descId} className="nb-rcard-desc">
                {description(option)}
              </span>
            </span>
            <span className="cmp2-check-box" aria-hidden>
              <CheckTick on={checked} />
            </span>
          </button>
        );
      })}
    </div>
  );
}

interface GoalPickerProps {
  /** The ticked cards, in the order she picked them (`unknown` / `info` only ever alone — `toggleGoal`). */
  value: readonly CompanionGoal[];
  onChange: (goals: CompanionGoal[]) => void;
  disabled?: boolean;
}

/** «دوست داری همراهت بیشتر در چه چیزی کنارت باشد؟» — one or more of six, never a diagnosis. */
export function GoalPicker({ value, onChange, disabled }: GoalPickerProps) {
  const t = useTranslations('companions');
  return (
    <>
      <p className="cmp-lead">{t('goalsHint')}</p>
      <CheckCards
        label={t('goalLabel')}
        options={COMPANION_GOALS}
        value={value}
        look={GOAL_LOOK}
        title={(goal) => t(`goals.${goal}.title`)}
        description={(goal) => t(`goals.${goal}.desc`)}
        onToggle={(goal) => onChange(toggleGoal(value, goal))}
        disabled={disabled}
        pilotId="companion.goals"
      />
    </>
  );
}

interface SignalSwitchesProps {
  value: CompanionSignalScope;
  onChange: (scope: CompanionSignalScope) => void;
  disabled?: boolean;
}

/** One switch — one consent — per coarse signal; all off by default. */
export function SignalSwitches({ value, onChange, disabled }: SignalSwitchesProps) {
  const t = useTranslations('companions');
  const baseId = useId();
  return (
    <ul className="cmp2-toggles">
      {OFFERED_SIGNALS.map((signal) => {
        const titleId = `${baseId}-${signal}-t`;
        const descId = `${baseId}-${signal}-d`;
        return (
          <li key={signal} className="nb-card cmp2-toggle">
            <IconCircle icon={SIGNAL_LOOK[signal].icon} tone={SIGNAL_LOOK[signal].tone} />
            <span className="cmp2-toggle-text">
              <b id={titleId} className="cmp2-toggle-title">
                {t(`signals.${signal}.title`)}
              </b>
              <span id={descId} className="cmp2-toggle-desc">
                {t(`signals.${signal}.desc`)}
              </span>
            </span>
            <Switch
              checked={value[signal]}
              labelledBy={titleId}
              describedBy={descId}
              disabled={disabled}
              pilot={pilot(`companion.signal:${signal}`)}
              onCheckedChange={(on) => onChange({ ...value, [signal]: on })}
            />
          </li>
        );
      })}
    </ul>
  );
}

interface ActionPickerProps {
  value: readonly CompanionAction[];
  onChange: (prefs: CompanionAction[]) => void;
  disabled?: boolean;
}

/** Many-of-many support kinds as check cards; none is required and the picked order is kept. */
export function ActionPicker({ value, onChange, disabled }: ActionPickerProps) {
  const t = useTranslations('companions');
  return (
    <CheckCards
      label={t('actionsLabel')}
      options={COMPANION_ACTIONS}
      value={value}
      look={ACTION_LOOK}
      title={(action) => t(`actions.${action}.title`)}
      description={(action) => t(`actions.${action}.desc`)}
      onToggle={(action) => onChange(toggleAction(value, action))}
      disabled={disabled}
      pilotId="companion.actions"
    />
  );
}
