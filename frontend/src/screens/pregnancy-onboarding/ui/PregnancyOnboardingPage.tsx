'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useMemo, useState } from 'react';

import {
  useCompleteOnboarding,
  usePregnancyEnums,
  type AgeSource,
  type OnboardingInput,
} from '@/entities/pregnancy';
import { JalaliDateWheels, jalaliPartsToApiDate } from '@/features/edit-profile';
import { Chip, NumberField, PgCard, Segmented, Toggle } from '@/features/track-pregnancy';
import { useRouter, type Locale } from '@/shared/i18n';
import { todayJalali, type JalaliParts } from '@/shared/lib/date';
import { Icon } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

const AGE_SOURCES: AgeSource[] = ['lmp', 'ultrasound', 'manual'];
const BLOOD_TYPES = ['A', 'B', 'AB', 'O'];
const RH_FACTORS = ['positive', 'negative'];
const CONDITIONS = ['chronic_hypertension', 'diabetes', 'hypothyroidism', 'hyperthyroidism', 'none'];

type DynT = (key: string) => string;

/** Onboarding: pick a dating method, enter the dates/weeks it needs, and add
 *  optional history. Submits to POST /pregnancy/onboarding, then lands on the
 *  tracker. Sensitive data never leaves the form except in the request (§11). */
export function PregnancyOnboardingPage() {
  const t = useTranslations('pregnancy');
  const locale = useLocale() as Locale;
  const isRtl = locale === 'fa';
  const router = useRouter();
  const dyn = t as unknown as DynT;

  const enums = usePregnancyEnums();
  const onboard = useCompleteOnboarding();

  const [ageSource, setAgeSource] = useState<AgeSource | undefined>(undefined);
  const [lmp, setLmp] = useState<JalaliParts>(() => todayJalali());
  const [scanDate, setScanDate] = useState<JalaliParts>(() => todayJalali());
  const [scanWeeks, setScanWeeks] = useState<number | undefined>();
  const [scanDays, setScanDays] = useState<number | undefined>();
  const [manualWeeks, setManualWeeks] = useState<number | undefined>();
  const [manualDays, setManualDays] = useState<number | undefined>();
  const [miscarriage, setMiscarriage] = useState(false);
  const [highRisk, setHighRisk] = useState(false);
  const [conditions, setConditions] = useState<string[]>([]);
  const [bloodType, setBloodType] = useState<string | undefined>();
  const [rhFactor, setRhFactor] = useState<string | undefined>();
  const [error, setError] = useState<string | null>(null);

  const thisJalaliYear = todayJalali().year;
  const dayOptions = useMemo(
    () => Array.from({ length: 7 }, (_, i) => ({ value: String(i), label: String(i) })),
    [],
  );

  const toggleCondition = (value: string) => {
    setConditions((prev) => {
      if (value === 'none') return prev.includes('none') ? [] : ['none'];
      const next = prev.filter((c) => c !== 'none');
      return next.includes(value) ? next.filter((c) => c !== value) : [...next, value];
    });
  };

  const buildPayload = (): OnboardingInput | null => {
    if (!ageSource) {
      setError(t('onboarding.selectAgeSource'));
      return null;
    }
    const base: OnboardingInput = { age_source: ageSource };
    if (ageSource === 'lmp') {
      base.lmp_date = jalaliPartsToApiDate(lmp);
    } else if (ageSource === 'ultrasound') {
      if (scanWeeks == null) {
        setError(t('onboarding.fillRequired'));
        return null;
      }
      base.ultrasound_date = jalaliPartsToApiDate(scanDate);
      base.ultrasound_weeks = scanWeeks;
      base.ultrasound_days = scanDays ?? 0;
    } else {
      if (manualWeeks == null) {
        setError(t('onboarding.fillRequired'));
        return null;
      }
      base.manual_weeks = manualWeeks;
      base.manual_days = manualDays ?? 0;
    }
    if (miscarriage) base.has_miscarriage_history = true;
    if (highRisk) base.has_high_risk_history = true;
    if (conditions.length) base.pre_existing_conditions = conditions;
    if (bloodType) base.blood_type = bloodType;
    if (rhFactor) base.rh_factor = rhFactor;
    return base;
  };

  const handleSubmit = () => {
    setError(null);
    const payload = buildPayload();
    if (!payload) return;
    onboard.mutate(payload, {
      onSuccess: () => router.replace('/pregnancy'),
      onError: () => setError(t('error')),
    });
  };

  const ageSources = enums.data?.ageSources ?? AGE_SOURCES;
  const bloodTypes = enums.data?.bloodTypes ?? BLOOD_TYPES;
  const rhFactors = enums.data?.rhFactors ?? RH_FACTORS;
  const conditionOptions = enums.data?.preExistingConditions ?? CONDITIONS;

  return (
    <div className="view" style={{ background: 'var(--page)' }}>
      <div className="scroll">
        <div style={{ padding: '10px 20px 0', textAlign: 'start' }}>
          <button className="iconbtn" onClick={() => router.back()} aria-label={t('back')}>
            <Icon name={isRtl ? 'chevronRight' : 'chevronLeft'} size={20} />
          </button>
          <div className="titr" style={{ marginTop: 8 }}>{t('onboarding.title')}</div>
          <p className="sub" style={{ margin: '6px 0 0' }}>{t('onboarding.subtitle')}</p>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 12, padding: '16px 16px 0' }}>
          {/* Dating method */}
          <PgCard title={t('onboarding.ageSourceLabel')} icon="calendar">
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {ageSources.map((src) => {
                const on = ageSource === src;
                return (
                  <button
                    key={src}
                    type="button"
                    onClick={() => setAgeSource(src as AgeSource)}
                    className="card"
                    style={{
                      display: 'flex',
                      alignItems: 'flex-start',
                      gap: 10,
                      padding: '11px 12px',
                      textAlign: 'start',
                      cursor: 'pointer',
                      border: on ? '2px solid var(--brand)' : '1px solid var(--line)',
                      background: on ? 'var(--surface-2)' : 'var(--surface)',
                    }}
                  >
                    <span className="dot" style={{ width: 20, height: 20, marginTop: 2, background: on ? 'var(--brand)' : 'var(--track)', color: 'var(--on-accent)' }}>
                      {on && <Icon name="check" size={13} />}
                    </span>
                    <span>
                      <span style={{ display: 'block', fontSize: 14, fontWeight: 800, color: 'var(--ink)' }}>
                        {dyn(`onboarding.ageSource.${src}`)}
                      </span>
                      <span style={{ display: 'block', fontSize: 12, color: 'var(--muted)', marginTop: 2 }}>
                        {dyn(`onboarding.ageSourceHint.${src}`)}
                      </span>
                    </span>
                  </button>
                );
              })}
            </div>
          </PgCard>

          {/* Conditional dating inputs */}
          {ageSource === 'lmp' && (
            <PgCard title={t('onboarding.lmpDate')}>
              <JalaliDateWheels idPrefix="lmp" value={lmp} onChange={setLmp} minYear={thisJalaliYear - 1} maxYear={thisJalaliYear} />
            </PgCard>
          )}

          {ageSource === 'ultrasound' && (
            <PgCard title={t('onboarding.ultrasoundDate')}>
              <JalaliDateWheels idPrefix="scan" value={scanDate} onChange={setScanDate} minYear={thisJalaliYear - 1} maxYear={thisJalaliYear} />
              <div style={{ marginTop: 12 }}>
                <span style={{ fontSize: 13, fontWeight: 700, color: 'var(--ink)' }}>{t('onboarding.ultrasoundAge')}</span>
                <div style={{ display: 'flex', gap: 10, marginTop: 8 }}>
                  <div style={{ flex: 1 }}>
                    <NumberField label={t('onboarding.weeks')} value={scanWeeks} onChange={setScanWeeks} min={1} max={42} />
                  </div>
                </div>
                <div style={{ marginTop: 10 }}>
                  <span style={{ fontSize: 13, fontWeight: 700, color: 'var(--ink)', display: 'block', marginBottom: 8 }}>{t('onboarding.days')}</span>
                  <Segmented options={dayOptions} value={scanDays != null ? String(scanDays) : undefined} onChange={(v) => setScanDays(v == null ? undefined : Number(v))} />
                </div>
              </div>
            </PgCard>
          )}

          {ageSource === 'manual' && (
            <PgCard title={t('onboarding.manualAge')}>
              <NumberField label={t('onboarding.weeks')} value={manualWeeks} onChange={setManualWeeks} min={1} max={42} />
              <div style={{ marginTop: 10 }}>
                <span style={{ fontSize: 13, fontWeight: 700, color: 'var(--ink)', display: 'block', marginBottom: 8 }}>{t('onboarding.days')}</span>
                <Segmented options={dayOptions} value={manualDays != null ? String(manualDays) : undefined} onChange={(v) => setManualDays(v == null ? undefined : Number(v))} />
              </div>
            </PgCard>
          )}

          {/* History */}
          <PgCard title={t('onboarding.history.title')} icon="shield">
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '8px 0' }}>
              <span style={{ fontSize: 13.5, fontWeight: 700, color: 'var(--ink)' }}>{t('onboarding.history.miscarriage')}</span>
              <Toggle on={miscarriage} onClick={() => setMiscarriage((v) => !v)} />
            </div>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '8px 0' }}>
              <span style={{ fontSize: 13.5, fontWeight: 700, color: 'var(--ink)' }}>{t('onboarding.history.highRisk')}</span>
              <Toggle on={highRisk} onClick={() => setHighRisk((v) => !v)} />
            </div>
          </PgCard>

          {/* Conditions */}
          <PgCard title={t('onboarding.conditions.title')} icon="stetho" hint={t('onboarding.optional')}>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
              {conditionOptions.map((c) => (
                <Chip key={c} on={conditions.includes(c)} label={dyn(`onboarding.conditions.${c}`)} onClick={() => toggleCondition(c)} />
              ))}
            </div>
          </PgCard>

          {/* Blood group + Rh */}
          <PgCard title={t('onboarding.bloodTypeLabel')} icon="drop" hint={t('onboarding.optional')}>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
              {bloodTypes.map((b) => (
                <Chip key={b} on={bloodType === b} label={b} onClick={() => setBloodType(bloodType === b ? undefined : b)} />
              ))}
            </div>
            <div style={{ marginTop: 12 }}>
              <span style={{ fontSize: 13, fontWeight: 700, color: 'var(--ink)', display: 'block', marginBottom: 8 }}>{t('onboarding.rhLabel')}</span>
              <div style={{ display: 'flex', gap: 8 }}>
                {rhFactors.map((r) => (
                  <Chip key={r} on={rhFactor === r} label={dyn(`onboarding.rh.${r}`)} onClick={() => setRhFactor(rhFactor === r ? undefined : r)} />
                ))}
              </div>
            </div>
          </PgCard>

          {error && (
            <p style={{ color: 'var(--brand)', fontSize: 12.5, fontWeight: 700, textAlign: 'center', margin: 0 }}>{error}</p>
          )}
        </div>

        <div style={{ height: 20 }} />
      </div>

      <div style={{ padding: '10px 16px 4px', background: 'var(--page)' }}>
        <button className="btn btn-primary" onClick={handleSubmit} disabled={onboard.isPending} style={{ borderRadius: 16 }}>
          {onboard.isPending ? t('onboarding.submitting') : t('onboarding.submit')}
        </button>
      </div>

      <BottomNav />
    </div>
  );
}
