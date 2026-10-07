import type { TreatmentItem, TreatmentKind, TreatmentTip } from '@/entities/menopause';
import type { Locale } from '@/shared/i18n';
import { addDays, fromApiDate, toApiDate, WEEKDAY_KEYS, weekdayKeys } from '@/shared/lib/date';
import type { IconName, Tone, WeekDotState } from '@/shared/ui';

/*
 * Pure helpers of «درمان و مراقبت» (CB-MENO-10, nbl_Meno_Treatment).
 */

/** The API's Saturday → Friday week in the locale's column order (en starts on Sunday). */
export function inLocaleOrder<T>(saturdayFirst: readonly T[], locale: Locale): T[] {
  return weekdayKeys(locale).flatMap((key) => {
    const day = saturdayFirst[WEEKDAY_KEYS.indexOf(key)];
    return day === undefined ? [] : [day];
  });
}

/**
 * The HRT card's «این هفته» dots: a day is done when every daily HRT item
 * running that day was taken (weekly ones don't count per day), future after
 * `today`. Saturday → Friday.
 */
export function groupWeekDots(items: readonly TreatmentItem[], weekDates: readonly string[], today: string): WeekDotState[] {
  const daily = items.filter((it) => it.schedule !== 'weekly');
  return weekDates.map((date) => {
    if (date > today) return 'future';
    const running = daily.filter((it) => !it.startedOn || it.startedOn <= date);
    if (!running.length) return 'missed';
    const done = running.every((it) => it.week.some((d) => d.date === date && d.taken));
    return done ? 'done' : 'missed';
  });
}

/** «۶ از ۷ روز»: done dots over the days that have come. */
export function dotsSummary(dots: readonly WeekDotState[]): { done: number; days: number } {
  return { done: dots.filter((d) => d === 'done').length, days: dots.filter((d) => d !== 'future').length };
}

/** The seven dates of the screen's week, Saturday first. */
export function weekDates(from: string, items: readonly TreatmentItem[]): string[] {
  const fromItems = items.find((it) => it.week.length === 7)?.week.map((d) => d.date);
  if (fromItems) return fromItems;
  const start = fromApiDate(from);
  return Array.from({ length: 7 }, (_, i) => toApiDate(addDays(start, i)));
}

const ITEM_LOOKS: Record<Exclude<TreatmentKind, 'lifestyle'>, readonly { icon: IconName; tone: Tone }[]> = {
  hrt: [
    { icon: 'pill', tone: 'data' },
    { icon: 'pill', tone: 'brand' },
  ],
  supplement: [{ icon: 'capsule', tone: 'brand' }],
};

const LIFESTYLE_LOOKS: readonly { icon: IconName; tone: Tone }[] = [
  { icon: 'run', tone: 'data' },
  { icon: 'heart', tone: 'bloom' },
  { icon: 'brain', tone: 'brand' },
];

/** Icon disc of a row: the board alternates tones down each card. */
export function itemLook(kind: TreatmentKind, index: number): { icon: IconName; tone: Tone } {
  const looks = kind === 'lifestyle' ? LIFESTYLE_LOOKS : ITEM_LOOKS[kind];
  return looks[index % looks.length] ?? { icon: 'pill', tone: 'brand' };
}

/** The catalog's lifestyle tip of the same name (its body is the row's «why»). */
export function lifestyleTipOf(item: TreatmentItem, tips: readonly TreatmentTip[]): TreatmentTip | null {
  const name = item.name.trim();
  return tips.find((tip) => tip.placement === 'treatment_lifestyle' && tip.title?.trim() === name) ?? null;
}

/** Catalog lifestyle goals she hasn't added yet (by name), offered as one-tap suggestions. */
export function lifestyleSuggestions(items: readonly TreatmentItem[], tips: readonly TreatmentTip[]): TreatmentTip[] {
  const names = new Set(items.map((it) => it.name.trim()));
  return tips.filter(
    (tip) => tip.placement === 'treatment_lifestyle' && tip.title && tip.weeklyGoal && tip.goalUnit && !names.has(tip.title.trim()),
  );
}

/** A tip by code (the `treatment` placement copy), else null — callers fall back to the bundled string. */
export function tipBody(tips: readonly TreatmentTip[], code: string): string | null {
  return tips.find((tip) => tip.code === code)?.body ?? null;
}

/** Earliest start among the HRT items («شروع: مرداد ۱۴۰۵» under the section title). */
export function earliestStart(items: readonly TreatmentItem[]): string | null {
  return items.reduce<string | null>((min, it) => (it.startedOn && (!min || it.startedOn < min) ? it.startedOn : min), null);
}

/** Today's minutes / sessions of a lifestyle item. */
export function todayAmount(item: TreatmentItem, today: string): number {
  return item.week.find((d) => d.date === today)?.amount ?? 0;
}
