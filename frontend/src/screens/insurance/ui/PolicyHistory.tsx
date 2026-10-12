'use client';

import { useLocale, useTranslations } from 'next-intl';

import { type InsurancePolicyHistory, type PolicyStatus } from '@/entities/insurance';
import { type Locale } from '@/shared/i18n';
import { formatLongDate, fromApiDate } from '@/shared/lib/date';
import { StatusPill, type Tone } from '@/shared/ui';

const STATUS_TONE: Record<PolicyStatus, Tone> = { current: 'success', upcoming: 'data', past: 'neutral' };

/**
 * Policy switcher (policy history, user decision 2026-10-12): her policies newest period first. Picking one shows it on
 * the home; a past policy opens read-only. Shown only once she has more than one policy.
 */
export function PolicyHistory({ history, selectedId, onSelect }: { history: InsurancePolicyHistory; selectedId: number; onSelect: (id: number) => void }) {
  const t = useTranslations('insurance.home');
  const locale = useLocale() as Locale;
  if (history.policies.length < 2) return null;
  const day = (iso: string | null) => (iso ? formatLongDate(fromApiDate(iso), locale) : '…');

  return (
    <section className="ins-sec" aria-labelledby="ins-hist-h">
      <h2 id="ins-hist-h" className="ins-label">
        {t('historyTitle')}
      </h2>
      <ul className="ins-hist">
        {history.policies.map((p) => {
          const selected = p.id === selectedId;
          return (
            <li key={p.id}>
              <button type="button" className={`nb-card ins-hist-row${selected ? ' is-selected' : ''}`} aria-pressed={selected} onClick={() => onSelect(p.id)}>
                <span className="ins-claim-text">
                  <span className="ins-claim-t">{p.planName ?? p.insurerName}</span>
                  <span className="ins-claim-s">
                    {p.validFrom || p.validTo ? t('period', { from: day(p.validFrom), to: day(p.validTo) }) : t('periodOpen')}
                    {' · '}
                    {t('claimsCount', { count: p.claimsCount })}
                  </span>
                </span>
                <StatusPill tone={STATUS_TONE[p.status]}>{t(`policyStatus.${p.status}`)}</StatusPill>
              </button>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
