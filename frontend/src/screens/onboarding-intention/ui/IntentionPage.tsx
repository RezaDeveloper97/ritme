'use client';

import { useLocale, useTranslations } from 'next-intl';

import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Icon, NavBack } from '@/shared/ui';
import {
  nextOnboardingRoute,
  stepPosition,
  useOnboardingStore,
  type PregnancyIntention,
} from '@/entities/user';


const OPTIONS: PregnancyIntention[] = ['avoiding', 'pregnant', 'trying', 'unsure'];

/**
 * The branch point of onboarding: how the user relates to pregnancy. Choosing
 * "pregnant" routes to the dating-basis step (period tracking off); every other
 * answer continues to the cycle questions (see the step-sequence driver).
 */
export function IntentionPage() {
  const t = useTranslations('onboarding');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const { intention, setIntention } = useOnboardingStore();
  const step = stepPosition('intention', intention);

  const select = (choice: PregnancyIntention) => {
    setIntention(choice);
    // Route on the freshly chosen value — the store update isn't observable yet.
    router.push(nextOnboardingRoute('intention', choice));
  };

  return (
    <div className="view onb-page">
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="stepcount">{formatNumber(step.index, loc)}<span className="onb-dim"> / {formatNumber(step.total, loc)}</span></span>
      </div>

      <div className="scroll onb-body">
        <div className="onb-intro">
          <div className="titr">{t('intention.title')}</div>
          <p className="sub onb-intro-sub">{t('intention.subtitle')}</p>
        </div>

        <div className="onb-stack is-lg">
          {OPTIONS.map((opt) => {
            const on = intention === opt;
            return (
              <button
                key={opt}
                type="button"
                onClick={() => select(opt)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 12,
                  padding: '15px 16px',
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
                    flex: '0 0 auto',
                    width: 22,
                    height: 22,
                    borderRadius: '50%',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    background: on ? 'var(--pink)' : 'var(--line-2)',
                    color: 'var(--on-accent)',
                  }}
                >
                  {on && <Icon name="check" size={14} />}
                </span>
                <span style={{ fontSize: 15, fontWeight: 700, color: on ? 'var(--pink)' : 'var(--ink)' }}>
                  {t(`intention.options.${opt}`)}
                </span>
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
}
