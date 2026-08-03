'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useMemo, useState } from 'react';

import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber, todayParts, type DateParts } from '@/shared/lib/date';
import { Icon, NavBack } from '@/shared/ui';
import {
  nextOnboardingRoute,
  stepPosition,
  useOnboardingStore,
  type OnboardingAgeSource,
} from '@/entities/user';
import { DateWheels } from '@/features/edit-profile';
import { NumberField, Segmented } from '@/features/track-pregnancy';

const SOURCES: OnboardingAgeSource[] = ['lmp', 'ultrasound', 'manual'];

/**
 * Pregnant branch: pick a basis for dating the pregnancy (last period /
 * ultrasound / manual week) and enter what it needs. The answer is stashed in
 * the onboarding store; the setting-up screen turns it into the pregnancy
 * activation + onboarding calls. Sensitive data stays in the store (§11).
 */
export function PregnancyBasisPage() {
  const t = useTranslations('onboarding');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const { intention, pregnancyBasis, setPregnancyBasis } = useOnboardingStore();
  const step = stepPosition('pregnancyBasis', intention);

  const thisYear = todayParts(loc).year;
  const [source, setSource] = useState<OnboardingAgeSource | null>(pregnancyBasis.source);
  const [lmp, setLmp] = useState<DateParts>(pregnancyBasis.lmp ?? todayParts(loc));
  const [scanDate, setScanDate] = useState<DateParts>(pregnancyBasis.ultrasoundDate ?? todayParts(loc));
  const [scanWeeks, setScanWeeks] = useState<number | undefined>(pregnancyBasis.ultrasoundWeeks ?? undefined);
  const [scanDays, setScanDays] = useState<number | undefined>(pregnancyBasis.ultrasoundDays ?? undefined);
  const [manualWeeks, setManualWeeks] = useState<number | undefined>(pregnancyBasis.manualWeeks ?? undefined);
  const [manualDays, setManualDays] = useState<number | undefined>(pregnancyBasis.manualDays ?? undefined);
  const [error, setError] = useState<string | null>(null);

  const dayOptions = useMemo(
    () => Array.from({ length: 7 }, (_, i) => ({ value: String(i), label: formatNumber(i, loc) })),
    [loc],
  );

  const handleNext = () => {
    setError(null);
    if (!source) return setError(t('pregnancyBasis.selectSource'));
    if (source === 'ultrasound' && scanWeeks == null) return setError(t('pregnancyBasis.fillRequired'));
    if (source === 'manual' && manualWeeks == null) return setError(t('pregnancyBasis.fillRequired'));

    setPregnancyBasis({
      source,
      lmp: source === 'lmp' ? lmp : null,
      ultrasoundDate: source === 'ultrasound' ? scanDate : null,
      ultrasoundWeeks: source === 'ultrasound' ? (scanWeeks ?? null) : null,
      ultrasoundDays: source === 'ultrasound' ? (scanDays ?? 0) : null,
      manualWeeks: source === 'manual' ? (manualWeeks ?? null) : null,
      manualDays: source === 'manual' ? (manualDays ?? 0) : null,
    });
    router.push(nextOnboardingRoute('pregnancyBasis', 'pregnant'));
  };

  return (
    <div className="view onb-page">
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="stepcount">{formatNumber(step.index, loc)}<span className="onb-dim"> / {formatNumber(step.total, loc)}</span></span>
      </div>

      <div className="scroll onb-body">
        <div className="onb-intro">
          <div className="titr">{t('pregnancyBasis.title')}</div>
          <p className="sub onb-intro-sub">{t('pregnancyBasis.subtitle')}</p>
        </div>

        <div className="onb-stack is-spaced">
          {SOURCES.map((src) => {
            const on = source === src;
            return (
              <button
                key={src}
                type="button"
                onClick={() => setSource(src)}
                style={{
                  display: 'flex',
                  alignItems: 'flex-start',
                  gap: 11,
                  padding: '13px 14px',
                  borderRadius: 16,
                  textAlign: 'start',
                  cursor: 'pointer',
                  border: `2px solid ${on ? 'var(--pink)' : 'var(--line-2)'}`,
                  background: on ? 'var(--surface-2)' : 'var(--surface)',
                  transition: '.18s',
                }}
              >
                <span
                  style={{
                    flex: '0 0 auto', width: 22, height: 22, borderRadius: '50%', marginTop: 1,
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    background: on ? 'var(--pink)' : 'var(--line-2)', color: 'var(--on-accent)',
                  }}
                >
                  {on && <Icon name="check" size={14} />}
                </span>
                <span>
                  <span className="onb-choice-t" style={{ color: on ? 'var(--pink)' : 'var(--ink)' }}>
                    {t(`pregnancyBasis.source.${src}`)}
                  </span>
                  <span className="onb-choice-h">
                    {t(`pregnancyBasis.sourceHint.${src}`)}
                  </span>
                </span>
              </button>
            );
          })}
        </div>

        {source === 'lmp' && (
          <div className="onb-mt16">
            <DateWheels idPrefix="lmp" value={lmp} onChange={setLmp} minYear={thisYear - 1} maxYear={thisYear} />
          </div>
        )}

        {source === 'ultrasound' && (
          <div className="onb-mt16">
            <DateWheels idPrefix="scan" value={scanDate} onChange={setScanDate} minYear={thisYear - 1} maxYear={thisYear} />
            <div className="onb-mt14">
              <NumberField label={t('pregnancyBasis.weeks')} value={scanWeeks} onChange={setScanWeeks} min={1} max={42} />
            </div>
            <div className="onb-mt12">
              <span className="onb-sublabel">{t('pregnancyBasis.days')}</span>
              <Segmented options={dayOptions} value={scanDays != null ? String(scanDays) : undefined} onChange={(v) => setScanDays(v == null ? undefined : Number(v))} />
            </div>
          </div>
        )}

        {source === 'manual' && (
          <div className="onb-mt16">
            <NumberField label={t('pregnancyBasis.weeks')} value={manualWeeks} onChange={setManualWeeks} min={1} max={42} />
            <div className="onb-mt12">
              <span className="onb-sublabel">{t('pregnancyBasis.days')}</span>
              <Segmented options={dayOptions} value={manualDays != null ? String(manualDays) : undefined} onChange={(v) => setManualDays(v == null ? undefined : Number(v))} />
            </div>
          </div>
        )}

        <p className="sub onb-note is-sub">
          {t('pregnancyBasis.note')}
        </p>

        {error && (
          <p className="onb-error is-pink is-spaced">{error}</p>
        )}

        <div className="onb-tail" />
      </div>

      <div className="onb-actions">
        <button className="btn btn-primary" onClick={handleNext}>
          {t('continue')}
        </button>
      </div>
    </div>
  );
}
