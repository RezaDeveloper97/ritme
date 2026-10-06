'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useMemo } from 'react';

import type { Locale } from '@/shared/i18n';
import { addDays, formatDayMonth, formatDecimal, formatNumber, fromApiDate, toApiDate, today } from '@/shared/lib/date';

import { glucoseText, unitSymbol } from '../model/units';
import type { GlucoseUnit, VitalReading, VitalType } from '../model/types';

/**
 * Display helpers of the vitals screens: locale digits (۱۱۸/۷۶), the reading's
 * value line, its «when» («امروز ۸:۱۰», «دیروز ۲۲:۰۰», «۲ مهر ۸:۰۰») and the
 * meta line («فشار خون · دست چپ، نشسته · امروز ۸:۱۰»). Dates go through
 * `shared/lib/date` (Jalali in fa).
 */
export function useVitalFormat() {
  const t = useTranslations('vitals');
  const locale = useLocale() as Locale;
  return useMemo(() => {
    const num = (n: number) => formatNumber(n, locale);
    const dec = (n: number, decimals = 1) => formatDecimal(n.toFixed(decimals), locale);
    const bp = (sys: number, dia: number) => `${num(sys)}/${num(dia)}`;
    const glucose = (value: number, unit: GlucoseUnit) => formatDecimal(glucoseText(value, unit), locale);
    const clock = (hhmm: string) => formatNumber(hhmm.replace(/^0(\d)/, '$1'), locale);
    const todayKey = toApiDate(today());
    const yesterdayKey = toApiDate(addDays(today(), -1));
    const day = (date: string) =>
      date === todayKey ? t('common.today') : date === yesterdayKey ? t('common.yesterday') : formatDayMonth(fromApiDate(date), locale);
    const when = (r: Pick<VitalReading, 'date' | 'time' | 'source'>) =>
      r.time ? t('common.at', { day: day(r.date), time: clock(r.time) }) : `${day(r.date)}${t('common.separator')}${t('common.fromLog')}`;
    const classLabel = (type: VitalType, code: string) => t(`classes.${type}.${code}` as 'classes.bp.normal');

    /** Main number of a reading («۱۱۸/۷۶», «۹۴», «۷۲»), without the unit. */
    const value = (r: VitalReading): string => {
      if (r.bloodPressure) return bp(r.bloodPressure.systolic, r.bloodPressure.diastolic);
      if (r.glucose) return glucose(r.glucose.value, r.glucose.unit);
      if (r.heartRate) return num(r.heartRate.bpm);
      return '';
    };
    const unit = (r: VitalReading): string =>
      r.glucose ? unitSymbol(r.glucose.unit) : r.type === 'bp' ? t('units.mmhg') : t('units.bpm');

    /** The list title: «۱۱۸/۷۶ · نبض ۷۰», «۹۴ mg/dL», «۷۲ bpm». */
    const title = (r: VitalReading): string => {
      const sep = t('common.separator');
      if (r.bloodPressure) {
        const p = r.bloodPressure.pulse;
        return p ? `${value(r)}${sep}${t('common.pulseValue', { n: num(p) })}` : value(r);
      }
      return `${value(r)} ${unit(r)}`;
    };

    /** Conditions of a reading («دست چپ، نشسته», «ناشتا · گلوکومتر», «در حال استراحت»). */
    const conditions = (r: VitalReading): string[] => {
      if (r.bloodPressure) {
        const parts = [
          r.bloodPressure.arm ? t(`arms.${r.bloodPressure.arm}`) : null,
          r.bloodPressure.position ? t(`positions.${r.bloodPressure.position}`) : null,
        ].filter(Boolean);
        return parts.length ? [parts.join(t('common.listSeparator'))] : [];
      }
      if (r.glucose) {
        return [
          r.glucose.context ? t(`glucoseContexts.${r.glucose.context}`) : null,
          r.glucose.method ? t(`methods.${r.glucose.method}`) : null,
        ].filter((x): x is string => !!x);
      }
      if (r.heartRate?.context) return [t(`hrStates.${r.heartRate.context}`)];
      return [];
    };

    /** The list sub-line; `withType` prefixes «فشار خون» (hub mixes every type). */
    const meta = (r: VitalReading, withType: boolean): string =>
      [withType ? t(`types.${r.type}`) : null, ...conditions(r), when(r)].filter(Boolean).join(t('common.separator'));

    return { locale, num, dec, bp, glucose, clock, day, when, classLabel, value, unit, title, conditions, meta };
  }, [t, locale]);
}

export type VitalFormat = ReturnType<typeof useVitalFormat>;
