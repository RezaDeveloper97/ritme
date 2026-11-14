'use client';

import { useLocale, useTranslations } from 'next-intl';

import {
  type InsurancePolicyHistory,
  categoryKey,
  costTypeKey,
  type ClaimStatus,
  homeCoverages,
  type InsuranceClaimRow,
  type InsurancePolicyState,
  maskedTail,
  remainingOf,
  useAmountFormat,
  useRecentClaims,
  useRenewInsurancePolicy,
} from '@/entities/insurance';
import { Link, type Locale } from '@/shared/i18n';
import { formatDayMonth, formatLongDate, fromApiDate } from '@/shared/lib/date';
import { Icon, type IconName, InfoNote, ProgressBar, SecondaryButton, Skeleton, StatusPill, type Tone } from '@/shared/ui';

import { PolicyHistory } from './PolicyHistory';

const CLAIM_TONE: Record<ClaimStatus, Tone> = {
  submitted: 'neutral',
  reviewing: 'data',
  docs_requested: 'warm',
  paid: 'success',
  rejected: 'danger',
  appealed: 'data',
};

const ACTIONS: { key: 'claim' | 'centres' | 'health' | 'coverage' | 'status'; href: string; icon: IconName }[] = [
  { key: 'claim', href: '/insurance/claims/new', icon: 'plus' },
  { key: 'centres', href: '/insurance/centers', icon: 'mapPin' },
  { key: 'health', href: '/insurance/questionnaire', icon: 'fileDoc' },
  { key: 'coverage', href: '/insurance/coverage', icon: 'shield' },
  { key: 'status', href: '/insurance/status', icon: 'history' },
];

function ClaimRow({ claim }: { claim: InsuranceClaimRow }) {
  const t = useTranslations('insurance');
  const locale = useLocale() as Locale;
  const amount = useAmountFormat();
  const title = claim.title ?? t(`costTypes.${costTypeKey(claim.costType)}`);
  const who = [claim.memberName, formatDayMonth(fromApiDate(claim.serviceDate), locale)].filter(Boolean).join(' · ');
  return (
    <Link href={`/insurance/claims/${claim.id}`} className="nb-card ins-claim">
      <span className="ins-claim-text">
        <span className="ins-claim-t">{title}</span>
        <span className="ins-claim-s">{who}</span>
      </span>
      <span className="ins-claim-end">
        <span className="ins-claim-amt">{amount(claim.amount)}</span>
        <StatusPill tone={CLAIM_TONE[claim.status]}>{t(`claimStatus.${claim.status}`)}</StatusPill>
      </span>
    </Link>
  );
}

function RecentClaims({ policyId }: { policyId: number | null }) {
  const t = useTranslations('insurance.home');
  const claims = useRecentClaims(true, policyId);
  return (
    <section className="ins-sec" aria-labelledby="ins-recent-h">
      <div className="ins-sec-head">
        <h2 id="ins-recent-h" className="ins-label">
          {t('recentTitle')}
        </h2>
        {policyId === null ? (
          <Link href="/insurance/claims" className="ins-more">
            {t('all')}
          </Link>
        ) : null}
      </div>
      {claims.isPending ? (
        <Skeleton shape="card" />
      ) : claims.isError ? (
        <p className="ins-note">{t('recentError')}</p>
      ) : claims.data.claims.length === 0 ? (
        <p className="nb-card ins-empty">{t('recentEmpty')}</p>
      ) : (
        <div className="ins-claims">
          {claims.data.claims.slice(0, 3).map((c) => (
            <ClaimRow key={c.id} claim={c} />
          ))}
        </div>
      )}
    </section>
  );
}

interface InsuranceHomeProps {
  state: InsurancePolicyState;
  /** Her policy history (switcher); null while loading or when it failed. */
  history: InsurancePolicyHistory | null;
  onSelect: (id: number) => void;
  onRenewed: (id: number) => void;
  onEdit: () => void;
  onShowInfo: () => void;
}

/**
 * Insurance home (CB-INS-03, nbl_Ins_Home): her own policy card, the action card when a claim needs documents,
 * remaining coverage bars and the latest claims. A personal notebook — nothing here is sent to an insurer.
 * Policy history: the switcher lists her policies; a past one shows read-only (no actions, no edit) with its own
 * claims and usage; the newest one can be renewed.
 */
export function InsuranceHome({ state, history, onSelect, onRenewed, onEdit, onShowInfo }: InsuranceHomeProps) {
  const t = useTranslations('insurance.home');
  const tc = useTranslations('insurance');
  const locale = useLocale() as Locale;
  const amount = useAmountFormat();
  const renew = useRenewInsurancePolicy();
  const { policy } = state;
  if (!policy) return null;
  const isCurrent = policy.status === 'current';
  const bars = homeCoverages(policy);
  const names = policy.members.map((m) => m.name).join('، ');
  const tail = maskedTail(policy.numberLast4);

  return (
    <>
      <section className="nb-card ins-policy" aria-label={policy.planName ?? t('plan')}>
        <div className="ins-policy-top">
          <span className="ins-policy-ico" aria-hidden>
            <Icon name="shield" size={22} />
          </span>
          <div className="ins-policy-text">
            <h2 className="ins-policy-t">{policy.planName ?? t('plan')}</h2>
            <p className="ins-policy-s">{t('insurer', { name: policy.insurerName })}</p>
          </div>
        </div>
        <ul className="ins-policy-facts">
          {policy.validTo ? <li>{t('validUntil', { date: formatLongDate(fromApiDate(policy.validTo), locale) })}</li> : null}
          {tail ? (
            <li>
              {t('number', { tail: '' })}
              <bdi dir="ltr">{tail}</bdi>
            </li>
          ) : null}
          {names ? <li>{t('members', { names })}</li> : null}
        </ul>
      </section>

      {history ? <PolicyHistory history={history} selectedId={policy.id} onSelect={onSelect} /> : null}
      {policy.readOnly ? <InfoNote icon="history">{t('pastNote')}</InfoNote> : null}
      {policy.status === 'upcoming' ? <InfoNote icon="calendar">{t('upcomingNote')}</InfoNote> : null}
      {!isCurrent && history?.currentId ? (
        <SecondaryButton variant="text" icon="shield" onClick={() => onSelect(history.currentId ?? policy.id)}>
          {t('backToCurrent')}
        </SecondaryButton>
      ) : null}

      {isCurrent ? (
        <nav className="ins-actions" aria-label={t('actions.coverage')}>
          {ACTIONS.map((a) => (
            <Link key={a.key} href={a.href} className="ins-action">
              <span className="ins-action-ico" aria-hidden>
                <Icon name={a.icon} size={20} />
              </span>
              <span className="ins-action-t">{t(`actions.${a.key}`)}</span>
            </Link>
          ))}
        </nav>
      ) : null}

      {isCurrent && state.needsAction > 0 ? (
        <Link href="/insurance/claims" className="ins-attn">
          <Icon name="warning" size={20} className="ins-attn-ico" />
          <span className="ins-attn-text">
            <span className="ins-attn-t">{t('needsAction', { count: state.needsAction })}</span>
          </span>
        </Link>
      ) : null}

      {bars.length > 0 ? (
        <section className="nb-card ins-card" aria-labelledby="ins-rem-h">
          <div className="ins-sec-head">
            <h2 id="ins-rem-h" className="ins-h">
              {t('remainingTitle')}
            </h2>
            {isCurrent ? (
              <Link href="/insurance/coverage" className="ins-more">
                {t('details')}
              </Link>
            ) : null}
          </div>
          {bars.map((c) => {
            const left = remainingOf(c);
            const total = c.capAmount ?? 0;
            const waiting = c.waitingDays ? (c.waitingOver ? t('waitingOver') : c.waitingEndsOn ? t('waitingUntil', { date: formatDayMonth(fromApiDate(c.waitingEndsOn), locale) }) : null) : null;
            return (
              <ProgressBar
                key={c.category}
                tone={c.waitingDays && !c.waitingOver ? 'warm' : 'data'}
                label={tc(`categories.${categoryKey(c.category)}`)}
                value={left}
                max={total}
                valueLabel={t('left', { left: amount(left), total: amount(total) })}
                hint={waiting}
              />
            );
          })}
        </section>
      ) : null}

      <RecentClaims policyId={isCurrent ? null : policy.id} />

      {policy.canRenew ? (
        <div className="ins-renew">
          <SecondaryButton icon="refresh" loading={renew.isPending} onClick={() => renew.mutate(policy.id, { onSuccess: (s) => s.policy && onRenewed(s.policy.id) })}>
            {t('renew')}
          </SecondaryButton>
          <p className="ins-hint">{t('renewHint')}</p>
          {renew.isError ? (
            <p className="ins-problem" role="alert">
              {t('renewError')}
            </p>
          ) : null}
        </div>
      ) : null}
      {policy.readOnly ? null : (
        <SecondaryButton variant="text" icon="pencil" onClick={onEdit}>
          {t('edit')}
        </SecondaryButton>
      )}
      <SecondaryButton variant="text" icon="info" onClick={onShowInfo}>
        {t('howTitle')}
      </SecondaryButton>
      <InfoNote icon="lock">{t('selfTrack')}</InfoNote>
    </>
  );
}
