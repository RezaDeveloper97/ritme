'use client';

import { useLocale, useTranslations } from 'next-intl';

import { useUserProfile, type BmiCategory } from '@/entities/user';
import type { Locale } from '@/shared/i18n';

import { SectionHead } from './SectionHead';

const FA = ['۰', '۱', '۲', '۳', '۴', '۵', '۶', '۷', '۸', '۹'];
const localizeNum = (value: string | number, loc: Locale) =>
  loc === 'fa' ? String(value).replace(/[0-9]/g, (d) => FA[Number(d)]) : String(value);

/** Per-band accent class; sets `--bmi-accent` / `--bmi-soft` for scale + pill. */
const BAND_CLASS: Record<BmiCategory, string> = {
  underweight: 'is-under',
  normal: 'is-normal',
  overweight: 'is-over',
  obese: 'is-obese',
};

/** Visible span of the scale. WHO band edges (18.5 / 25 / 30) fall inside it. */
const SCALE_MIN = 15;
const SCALE_MAX = 35;

const BANDS: BmiCategory[] = ['underweight', 'normal', 'overweight', 'obese'];

/**
 * «شاخص توده بدنی (BMI)» — the server-computed index with its band and the
 * supportive, band-specific paragraph the backend owns (admin-editable, already
 * localized). The linear scale is decorative reinforcement of the value+label
 * text, so it is aria-hidden. Descriptive, never diagnostic (§11): shown,
 * never logged.
 */
export function BmiCard() {
  const t = useTranslations('cycle');
  const loc = useLocale() as Locale;
  const { data: profile } = useUserProfile();
  const bmi = profile?.bmi;

  if (!bmi) return null;

  // Keep the pin fully on the track even for out-of-scale values.
  const pos = Math.min(
    97,
    Math.max(3, ((bmi.value - SCALE_MIN) / (SCALE_MAX - SCALE_MIN)) * 100),
  );

  return (
    <div className="sec">
      <div className="card pad-card">
        <SectionHead title={t('bmi.title')} />

        <div className="bmi-row">
          <span className="bmi-val">{localizeNum(bmi.value, loc)}</span>
          <span className={`bmi-cat ${BAND_CLASS[bmi.category]}`}>{bmi.categoryLabel}</span>
        </div>

        <div className="bmi-scale" aria-hidden="true">
          <div className="bmi-track">
            <div className="bmi-band">
              {BANDS.map((band) => (
                <span
                  key={band}
                  className={`bmi-seg ${BAND_CLASS[band]}${band === bmi.category ? ' is-active' : ''}`}
                />
              ))}
            </div>
            <span
              className={`bmi-pin ${BAND_CLASS[bmi.category]}`}
              style={{ insetInlineStart: `${pos}%` }}
            >
              <span className="bmi-dot" />
            </span>
          </div>
          <div className="bmi-ticks">
            <span className="bmi-tick bmi-tick-a">{localizeNum('18.5', loc)}</span>
            <span className="bmi-tick bmi-tick-b">{localizeNum(25, loc)}</span>
            <span className="bmi-tick bmi-tick-c">{localizeNum(30, loc)}</span>
          </div>
        </div>

        <p className="bmi-note">{bmi.message}</p>
      </div>
    </div>
  );
}
