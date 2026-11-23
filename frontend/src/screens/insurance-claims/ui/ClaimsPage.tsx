'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  CLAIM_TONE,
  costTypeKey,
  type InsuranceClaimRow,
  useAmountFormat,
  useInsurancePolicy,
  useRecentClaims,
} from '@/entities/insurance';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatMonthLabel, formatYear, fromApiDate, todayParts } from '@/shared/lib/date';
import { pilot, pilotOption } from '@/shared/pilot';
import {
  EmptyState,
  HeaderButton,
  IconCircle,
  InfoNote,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SearchField,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
} from '@/shared/ui';

import { CLAIM_FILTERS, type ClaimFilter, filterClaims, groupByMonth } from '../model/claims';
import '@/shared/ui/styles/ins.css';

function Row({ claim }: { claim: InsuranceClaimRow }) {
  const t = useTranslations('insurance');
  const locale = useLocale() as Locale;
  const amount = useAmountFormat();
  const title = claim.title ?? t(`costTypes.${costTypeKey(claim.costType)}`);
  const who = [claim.memberName, formatDayMonth(fromApiDate(claim.serviceDate), locale)].filter(Boolean).join(' · ');
  return (
    <li>
      <Link href={`/insurance/claims/${claim.id}`} className="ins-crow" {...pilot(`ins-claims.open:${claim.id}`)}>
        <IconCircle icon="fileDoc" tone="brand" />
        <span className="ins-claim-text">
          <span className="ins-claim-t">{title}</span>
          <span className="ins-claim-s">{who}</span>
        </span>
        <span className="ins-claim-end">
          <span className="ins-claim-amt">{amount(claim.amount)}</span>
          <StatusPill tone={CLAIM_TONE[claim.status]}>{t(`claimStatus.${claim.status}`)}</StatusPill>
        </span>
      </Link>
    </li>
  );
}

function Body() {
  const t = useTranslations('insurance.claims');
  const tc = useTranslations('insurance');
  const locale = useLocale() as Locale;
  const amount = useAmountFormat();
  const query = useRecentClaims();
  const [filter, setFilter] = useState<ClaimFilter>('all');
  const [q, setQ] = useState('');

  if (query.isPending) {
    return (
      <SkeletonGroup label={t('loading')} className="ins-skel">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  }
  if (query.isError) {
    return (
      <EmptyState
        icon="shield"
        title={t('error')}
        action={
          <PrimaryButton icon="refresh" loading={query.isFetching} onClick={() => void query.refetch()}>
            {tc('retry')}
          </PrimaryButton>
        }
      />
    );
  }

  const { claims, totals, needsAction } = query.data;
  const titleOf = (c: InsuranceClaimRow) => c.title ?? tc(`costTypes.${costTypeKey(c.costType)}`);
  const shown = filterClaims(claims, filter, q, titleOf);
  const months = groupByMonth(shown, locale);

  return (
    <>
      <section className="nb-card ins-stats" aria-label={t('statsLabel')}>
        <div className="ins-stat is-brand">
          <span className="ins-stat-n">{amount(totals.requested)}</span>
          <span className="ins-stat-t">{t('requested')}</span>
        </div>
        <div className="ins-stat is-data">
          <span className="ins-stat-n">{amount(totals.paid)}</span>
          <span className="ins-stat-t">{t('paid')}</span>
        </div>
        <div className="ins-stat is-warm">
          <span className="ins-stat-n">{t('needsActionCount', { count: needsAction })}</span>
          <span className="ins-stat-t">{t('needsAction')}</span>
        </div>
      </section>

      {claims.length > 0 ? (
        <>
          <SearchField value={q} onValueChange={setQ} label={t('search')} clearLabel={t('clear')} placeholder={t('search')} {...pilot('ins-claims.search', 'text')} />
          <div className="ins-chips" role="group" aria-label={t('filters')} {...pilot('ins-claims.filter', 'choice')}>
            {CLAIM_FILTERS.map((f) => (
              <PillChip key={f} pressed={filter === f} onPressedChange={() => setFilter(f)} {...pilotOption(f)}>
                {t(`filter.${f}`)}
              </PillChip>
            ))}
          </div>
        </>
      ) : null}

      {claims.length === 0 ? (
        <EmptyState
          icon="fileDoc"
          title={t('empty')}
          body={t('emptyHint')}
        />
      ) : months.length === 0 ? (
        <p className="nb-card ins-empty">{t('noMatch')}</p>
      ) : (
        months.map((m) => (
          <section key={m.key} className="ins-month" aria-label={formatMonthLabel(m.year, m.month, locale)}>
            <h2 className="ins-label">{formatMonthLabel(m.year, m.month, locale)}</h2>
            <ul className="nb-card ins-clist">
              {m.claims.map((c) => (
                <Row key={c.id} claim={c} />
              ))}
            </ul>
          </section>
        ))
      )}
      <InfoNote icon="lock">{t('selfTrack')}</InfoNote>
    </>
  );
}

/**
 * Claims list (CB-INS-04, nbl_Ins_Claims) at `/insurance/claims`: policy-period totals, search, status chips and the
 * claims grouped by month. Self-tracked: nothing here is sent to an insurer. Back-header screen, no bottom nav.
 */
export function InsuranceClaimsPage() {
  const t = useTranslations('insurance');
  const locale = useLocale() as Locale;
  const router = useRouter();
  // The policy decides whether claims can be added at all (404 without one).
  const policy = useInsurancePolicy();
  const hasPolicy = policy.data?.policy != null;

  return (
    <div className="view ins-page">
      <SkyLayer />
      <div className="scroll ins-scroll">
        <ScreenHeader
          title={t('claims.title')}
          subtitle={formatYear(todayParts(locale).year, locale)}
          onBack={() => router.push('/insurance')}
          backLabel={t('back')}
          action={hasPolicy ? <HeaderButton label={t('claims.add')} icon="plus" onClick={() => router.push('/insurance/claims/new')} pilot={pilot('ins-claims.add')} /> : undefined}
        />
        <Body />
      </div>
    </div>
  );
}
