'use client';

import { useLocale, useTranslations } from 'next-intl';

import { useUserProfile, type BmiCategory } from '@/entities/user';
import type { Locale } from '@/shared/i18n';

import { SectionHead } from './SectionHead';

const FA = ['۰', '۱', '۲', '۳', '۴', '۵', '۶', '۷', '۸', '۹'];
const localizeNum = (value: string | number, loc: Locale) =>
  loc === 'fa' ? String(value).replace(/[0-9]/g, (d) => FA[Number(d)]) : String(value);

/** Band accent, matching how the summary rows read a value against a range. */
const BAND_TONE: Record<BmiCategory, string> = {
  underweight: 'is-warn',
  normal: 'is-ok',
  overweight: 'is-warn',
  obese: 'is-warn',
};

/**
 * «شاخص توده بدنی (BMI)» — the server-computed index with its band and the
 * supportive, band-specific paragraph the backend owns (admin-editable, already
 * localized). Descriptive, never diagnostic (§11): shown, never logged.
 */
export function BmiCard() {
  const t = useTranslations('cycle');
  const loc = useLocale() as Locale;
  const { data: profile } = useUserProfile();
  const bmi = profile?.bmi;

  if (!bmi) return null;

  return (
    <div className="sec">
      <div className="card pad-card">
        <SectionHead title={t('bmi.title')} />

        <div className="bmi-row">
          <span className="bmi-val">{localizeNum(bmi.value, loc)}</span>
          <span className={`bmi-cat ${BAND_TONE[bmi.category]}`}>{bmi.categoryLabel}</span>
        </div>

        <p className="bmi-note">{bmi.message}</p>
      </div>
    </div>
  );
}
