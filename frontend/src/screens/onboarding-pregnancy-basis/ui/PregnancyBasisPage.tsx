'use client';

import { useTranslations } from 'next-intl';
import { useMemo, useState } from 'react';

import { useRouter } from '@/shared/i18n';
import { todayJalali, type JalaliParts } from '@/shared/lib/date';
import { Icon, NavBack } from '@/shared/ui';
import {
  nextOnboardingRoute,
  stepPosition,
  useOnboardingStore,
  type OnboardingAgeSource,
} from '@/entities/user';
import { JalaliDateWheels } from '@/features/edit-profile';
import { NumberField, Segmented } from '@/features/track-pregnancy';

const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
const faNum = (n: string | number) => String(n).replace(/[0-9]/g, d => FA[Number(d)]);

const SOURCES: OnboardingAgeSource[] = ['lmp', 'ultrasound', 'manual'];

/**
 * Pregnant branch: pick a basis for dating the pregnancy (last period /
 * ultrasound / manual week) and enter what it needs. The answer is stashed in
 * the onboarding store; the setting-up screen turns it into the pregnancy
 * activation + onboarding calls. Sensitive data stays in the store (§11).
 */
export function PregnancyBasisPage() {
  const t = useTranslations('onboarding');
  const router = useRouter();
  const { intention, pregnancyBasis, setPregnancyBasis } = useOnboardingStore();
  const step = stepPosition('pregnancyBasis', intention);

  const thisYear = todayJalali().year;
  const [source, setSource] = useState<OnboardingAgeSource | null>(pregnancyBasis.source);
  const [lmp, setLmp] = useState<JalaliParts>(pregnancyBasis.lmp ?? todayJalali());
  const [scanDate, setScanDate] = useState<JalaliParts>(pregnancyBasis.ultrasoundDate ?? todayJalali());
  const [scanWeeks, setScanWeeks] = useState<number | undefined>(pregnancyBasis.ultrasoundWeeks ?? undefined);
  const [scanDays, setScanDays] = useState<number | undefined>(pregnancyBasis.ultrasoundDays ?? undefined);
  const [manualWeeks, setManualWeeks] = useState<number | undefined>(pregnancyBasis.manualWeeks ?? undefined);
  const [manualDays, setManualDays] = useState<number | undefined>(pregnancyBasis.manualDays ?? undefined);
  const [error, setError] = useState<string | null>(null);

  const dayOptions = useMemo(
    () => Array.from({ length: 7 }, (_, i) => ({ value: String(i), label: faNum(i) })),
    [],
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
    <div className="view" style={{ background: '#fff' }}>
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="stepcount">{faNum(step.index)}<span style={{ opacity: .5 }}> / {faNum(step.total)}</span></span>
      </div>

      <div className="scroll" style={{ padding: '8px 22px 0' }}>
        <div style={{ textAlign: 'start', margin: '6px 0' }}>
          <div className="titr">{t('pregnancyBasis.title')}</div>
          <p className="sub" style={{ margin: '10px 0 0' }}>{t('pregnancyBasis.subtitle')}</p>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 10, marginTop: 16 }}>
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
                  border: `2px solid ${on ? 'var(--ritme-pink, #FB64B6)' : '#EBEEF2'}`,
                  background: on ? '#FFF0F7' : '#fff',
                  transition: '.18s',
                }}
              >
                <span
                  style={{
                    flex: '0 0 auto', width: 22, height: 22, borderRadius: '50%', marginTop: 1,
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    background: on ? 'var(--ritme-pink, #FB64B6)' : '#EBEEF2', color: '#fff',
                  }}
                >
                  {on && <Icon name="check" size={14} />}
                </span>
                <span>
                  <span style={{ display: 'block', fontSize: 14.5, fontWeight: 800, color: on ? 'var(--ritme-pink, #FB64B6)' : 'var(--ink)' }}>
                    {t(`pregnancyBasis.source.${src}`)}
                  </span>
                  <span style={{ display: 'block', fontSize: 12, color: 'var(--muted, #8A94A0)', marginTop: 3 }}>
                    {t(`pregnancyBasis.sourceHint.${src}`)}
                  </span>
                </span>
              </button>
            );
          })}
        </div>

        {source === 'lmp' && (
          <div style={{ marginTop: 16 }}>
            <JalaliDateWheels idPrefix="lmp" value={lmp} onChange={setLmp} minYear={thisYear - 1} maxYear={thisYear} />
          </div>
        )}

        {source === 'ultrasound' && (
          <div style={{ marginTop: 16 }}>
            <JalaliDateWheels idPrefix="scan" value={scanDate} onChange={setScanDate} minYear={thisYear - 1} maxYear={thisYear} />
            <div style={{ marginTop: 14 }}>
              <NumberField label={t('pregnancyBasis.weeks')} value={scanWeeks} onChange={setScanWeeks} min={1} max={42} />
            </div>
            <div style={{ marginTop: 12 }}>
              <span style={{ fontSize: 13, fontWeight: 700, color: 'var(--ink)', display: 'block', marginBottom: 8 }}>{t('pregnancyBasis.days')}</span>
              <Segmented options={dayOptions} value={scanDays != null ? String(scanDays) : undefined} onChange={(v) => setScanDays(v == null ? undefined : Number(v))} />
            </div>
          </div>
        )}

        {source === 'manual' && (
          <div style={{ marginTop: 16 }}>
            <NumberField label={t('pregnancyBasis.weeks')} value={manualWeeks} onChange={setManualWeeks} min={1} max={42} />
            <div style={{ marginTop: 12 }}>
              <span style={{ fontSize: 13, fontWeight: 700, color: 'var(--ink)', display: 'block', marginBottom: 8 }}>{t('pregnancyBasis.days')}</span>
              <Segmented options={dayOptions} value={manualDays != null ? String(manualDays) : undefined} onChange={(v) => setManualDays(v == null ? undefined : Number(v))} />
            </div>
          </div>
        )}

        <p className="sub" style={{ margin: '18px 0 0', fontSize: 12, color: 'var(--muted, #8A94A0)', lineHeight: 1.7 }}>
          {t('pregnancyBasis.note')}
        </p>

        {error && (
          <p style={{ color: 'var(--ritme-pink, #FB64B6)', fontSize: 12.5, fontWeight: 700, textAlign: 'center', margin: '12px 0 0' }}>{error}</p>
        )}

        <div style={{ height: 12 }} />
      </div>

      <div style={{ padding: '14px 16px 8px' }}>
        <button className="btn btn-primary" onClick={handleNext}>
          {t('continue')}
        </button>
      </div>
    </div>
  );
}
