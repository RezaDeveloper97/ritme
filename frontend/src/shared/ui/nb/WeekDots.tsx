import { clsx } from 'clsx';

import type { Locale } from '@/shared/i18n';
import { weekdayLabels } from '@/shared/lib/date';

import { Icon } from '../Icon';
import { toneClass, type Tone } from './tone';

export type WeekDotState = 'done' | 'missed' | 'future';

interface WeekDotsProps {
  /** Seven states in locale week order (Saturday → Friday in fa). */
  days: readonly WeekDotState[];
  locale: Locale;
  /** Accessible name, e.g. «این هفته». */
  label: string;
  /** Spoken state per day, e.g. { done: 'مصرف شد', missed: 'ثبت نشده', future: 'هنوز نرسیده' }. */
  stateLabels: Record<WeekDotState, string>;
  /** Index (0–6) of today — outlined in the brand. */
  todayIndex?: number;
  tone?: Tone;
  className?: string;
}

/**
 * Seven adherence dots with the short weekday under each (ش…ج) — HRT doses
 * (nbl_Meno_Treatment), pelvic sessions (nbl_Pelvic_Plan). Read-only: a list
 * where each day speaks «weekday: state».
 */
export function WeekDots({ days, locale, label, stateLabels, todayIndex, tone = 'data', className }: WeekDotsProps) {
  const names = weekdayLabels(locale);
  return (
    <ul aria-label={label} className={clsx('nb-wdots', toneClass(tone), className)}>
      {days.slice(0, 7).map((state, index) => (
        <li
          key={index}
          className={clsx('nb-wdot', `is-${state}`, index === todayIndex && 'is-today')}
          aria-label={`${names[index]}: ${stateLabels[state]}`}
        >
          <span className="nb-wdot-disc" aria-hidden>
            {state === 'done' ? <Icon name="check" size={16} strokeWidth={2.6} /> : null}
          </span>
          <span className="nb-wdot-day" aria-hidden>
            {names[index]}
          </span>
        </li>
      ))}
    </ul>
  );
}
