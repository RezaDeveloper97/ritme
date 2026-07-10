'use client';

import { useTranslations } from 'next-intl';

import { useRouter } from '@/shared/i18n';
import { Icon, NavBack } from '@/shared/ui';
import {
  nextOnboardingRoute,
  stepPosition,
  useOnboardingStore,
  type ChronicCondition,
} from '@/entities/user';

const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
const faNum = (n: string | number) => String(n).replace(/[0-9]/g, d => FA[Number(d)]);

const CONDITIONS: ChronicCondition[] = [
  'pcos',
  'hypothyroidism',
  'hyperthyroidism',
  'hypertension',
  'heart_disease',
  'diabetes',
];

/**
 * Optional final question: self-reported chronic conditions, to sharpen the
 * recommendations. Shared by both branches. Nothing is required — an empty
 * selection is valid ("none"). Sensitive data (§11); never logged.
 */
export function ConditionsPage() {
  const t = useTranslations('onboarding');
  const router = useRouter();
  const { intention, chronicConditions, toggleCondition } = useOnboardingStore();
  const step = stepPosition('conditions', intention);

  return (
    <div className="view" style={{ background: '#fff' }}>
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="stepcount">{faNum(step.index)}<span style={{ opacity: .5 }}> / {faNum(step.total)}</span></span>
      </div>

      <div className="scroll" style={{ padding: '8px 22px 0' }}>
        <div style={{ textAlign: 'start', margin: '6px 0' }}>
          <div className="titr">{t('conditions.title')}</div>
          <p className="sub" style={{ margin: '10px 0 0', lineHeight: 1.7 }}>{t('conditions.subtitle')}</p>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 10, marginTop: 16 }}>
          {CONDITIONS.map((c) => {
            const on = chronicConditions.includes(c);
            return (
              <button
                key={c}
                type="button"
                onClick={() => toggleCondition(c)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 12,
                  padding: '14px 15px',
                  borderRadius: 14,
                  textAlign: 'start',
                  cursor: 'pointer',
                  border: `2px solid ${on ? 'var(--ritme-pink, #FB64B6)' : '#EBEEF2'}`,
                  background: on ? '#FFF0F7' : '#fff',
                  transition: '.18s',
                }}
              >
                <span
                  style={{
                    flex: '0 0 auto', width: 22, height: 22, borderRadius: 7,
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    background: on ? 'var(--ritme-pink, #FB64B6)' : '#fff',
                    border: `2px solid ${on ? 'var(--ritme-pink, #FB64B6)' : '#D8DEE5'}`,
                    color: '#fff',
                  }}
                >
                  {on && <Icon name="check" size={13} />}
                </span>
                <span style={{ fontSize: 14.5, fontWeight: 700, color: on ? 'var(--ritme-pink, #FB64B6)' : 'var(--ink)' }}>
                  {t(`conditions.items.${c}`)}
                </span>
              </button>
            );
          })}
        </div>

        <div style={{ height: 12 }} />
      </div>

      <div style={{ padding: '14px 16px 8px' }}>
        <button className="btn btn-primary" onClick={() => router.push(nextOnboardingRoute('conditions', intention))}>
          {t('finish')}
        </button>
      </div>
    </div>
  );
}
