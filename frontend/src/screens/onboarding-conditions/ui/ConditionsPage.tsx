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
    <div className="view onb-page">
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="stepcount">{faNum(step.index)}<span className="onb-dim"> / {faNum(step.total)}</span></span>
      </div>

      <div className="scroll onb-scroll">
        <div className="onb-intro">
          <div className="titr">{t('conditions.title')}</div>
          <p className="sub onb-intro-sub is-loose">{t('conditions.subtitle')}</p>
        </div>

        <div className="onb-stack is-spaced">
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
                  border: `2px solid ${on ? 'var(--pink)' : 'var(--line-2)'}`,
                  background: on ? 'var(--surface-2)' : 'var(--surface)',
                  transition: '.18s',
                }}
              >
                <span
                  style={{
                    flex: '0 0 auto', width: 22, height: 22, borderRadius: 7,
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    background: on ? 'var(--pink)' : 'var(--surface)',
                    border: `2px solid ${on ? 'var(--pink)' : 'var(--track)'}`,
                    color: 'var(--on-accent)',
                  }}
                >
                  {on && <Icon name="check" size={13} />}
                </span>
                <span style={{ fontSize: 14.5, fontWeight: 700, color: on ? 'var(--pink)' : 'var(--ink)' }}>
                  {t(`conditions.items.${c}`)}
                </span>
              </button>
            );
          })}
        </div>

        <div className="onb-tail" />
      </div>

      <div className="onb-actions">
        <button className="btn btn-primary" onClick={() => router.push(nextOnboardingRoute('conditions', intention))}>
          {t('finish')}
        </button>
      </div>
    </div>
  );
}
