import type { ReactNode } from 'react';
import { clsx } from 'clsx';

import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';

import { Icon } from '../Icon';
import { toneClass, type Tone } from './tone';

export type StepState = 'done' | 'current' | 'todo';

export interface TimelineStep {
  id: string;
  title: ReactNode;
  /** Date / deadline / hint line («۵ مهر», «مهلت تا ۱۶ مهر», «حدود ۱۰ تا ۱۲ روز»). */
  meta?: ReactNode;
  state: StepState;
  /**
   * Accent of the current step (default `brand`); `bloom` marks «waiting on
   * you» (درخواست مدرک, nbl_Ins_ClaimDetail). Done steps are always `data`.
   */
  tone?: Tone;
}

interface StepTimelineProps {
  steps: readonly TimelineStep[];
  /** Accessible name, e.g. «مراحل درمان» / «روند خسارت». */
  label: string;
  /** Spoken state suffixes, e.g. { done: 'انجام شد', current: 'مرحله فعلی', todo: 'بعدی' }. */
  stateLabels: Record<StepState, string>;
  /**
   * `number` = 30px numbered discs, done → check (nbl_IVF_Home);
   * `dot` = 18px markers joined by a rail (nbl_Ins_ClaimDetail, Ins_Status).
   */
  marker?: 'number' | 'dot';
  /** Locale digits in `number` markers. */
  locale: Locale;
  className?: string;
}

/**
 * Vertical done / current / todo timeline. An ordered list; the current step
 * carries `aria-current="step"` and every step speaks its state.
 */
export function StepTimeline({ steps, label, stateLabels, marker = 'number', locale, className }: StepTimelineProps) {
  return (
    <ol aria-label={label} className={clsx('nb-tl', `is-${marker}`, className)}>
      {steps.map((step, index) => (
        <li
          key={step.id}
          aria-current={step.state === 'current' ? 'step' : undefined}
          className={clsx(
            'nb-tl-step',
            `is-${step.state}`,
            toneClass(step.state === 'done' ? 'data' : (step.tone ?? 'brand')),
          )}
        >
          <span className="nb-tl-marker" aria-hidden>
            {marker === 'number' ? (
              step.state === 'done' ? (
                <Icon name="check" size={14} strokeWidth={2.6} />
              ) : (
                formatNumber(index + 1, locale)
              )
            ) : step.state === 'done' ? (
              <Icon name="check" size={10} strokeWidth={3} />
            ) : null}
          </span>
          <div className="nb-tl-text">
            <span className="nb-tl-title">{step.title}</span>
            <span className="sr-only"> — {stateLabels[step.state]}</span>
            {step.meta ? <span className="nb-tl-meta">{step.meta}</span> : null}
          </div>
        </li>
      ))}
    </ol>
  );
}
