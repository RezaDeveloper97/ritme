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
import { CheckTick, IconCircle, type IconName, RadioCardGroup, Switch, type Tone } from '@/shared/ui';

import { OFFERED_SIGNALS, toggleAction } from '../model/draft';

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

interface GoalPickerProps {
  value: CompanionGoal | null;
  onChange: (goal: CompanionGoal) => void;
}

/** «دوست داری همراهت بیشتر در چه چیزی کنارت باشد؟» — one of six, never a diagnosis. */
export function GoalPicker({ value, onChange }: GoalPickerProps) {
  const t = useTranslations('companions');
  return (
    <RadioCardGroup
      label={t('goalLabel')}
      className="cmp2-opts"
      value={value}
      onChange={onChange}
      options={COMPANION_GOALS.map((goal) => ({
        value: goal,
        title: t(`goals.${goal}.title`),
        description: t(`goals.${goal}.desc`),
        icon: GOAL_LOOK[goal].icon,
        iconTone: GOAL_LOOK[goal].tone,
      }))}
    />
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
  const baseId = useId();
  return (
    <div role="group" aria-label={t('actionsLabel')} className="nb-rcards cmp2-opts">
      {COMPANION_ACTIONS.map((action) => {
        const checked = value.includes(action);
        const titleId = `${baseId}-${action}-t`;
        const descId = `${baseId}-${action}-d`;
        return (
          <button
            key={action}
            type="button"
            role="checkbox"
            aria-checked={checked}
            aria-labelledby={titleId}
            aria-describedby={descId}
            disabled={disabled}
            className="nb-rcard cmp2-check-card"
            onClick={() => onChange(toggleAction(value, action))}
          >
            <IconCircle icon={ACTION_LOOK[action].icon} tone={ACTION_LOOK[action].tone} size="lg" />
            <span className="nb-rcard-text">
              <span id={titleId} className="nb-rcard-title">
                {t(`actions.${action}.title`)}
              </span>
              <span id={descId} className="nb-rcard-desc">
                {t(`actions.${action}.desc`)}
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
