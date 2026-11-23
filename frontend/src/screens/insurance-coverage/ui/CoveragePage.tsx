'use client';

import { useLocale, useTranslations } from 'next-intl';

import {
  capped,
  categoryKey,
  deductibleTiles,
  type InsurancePolicy,
  type PolicyCoverage,
  remainingOf,
  useAmountFormat,
  useInsurancePolicy,
} from '@/entities/insurance';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatNumber, fromApiDate } from '@/shared/lib/date';
import { pilot } from '@/shared/pilot';
import {
  EmptyState,
  Icon,
  InfoNote,
  PrimaryButton,
  ProgressBar,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';
import '@/shared/ui/styles/ins.css';

function CapRow({ cov }: { cov: PolicyCoverage }) {
  const t = useTranslations('insurance.coverage');
  const tc = useTranslations('insurance');
  const locale = useLocale() as Locale;
  const amount = useAmountFormat();
  const left = remainingOf(cov);
  const total = cov.capAmount ?? 0;
  const months = cov.waitingDays ? Math.max(1, Math.round(cov.waitingDays / 30)) : null;
  let hint: string | null = null;
  if (cov.status === 'conditional') hint = t('conditional');
  else if (months !== null) {
    hint = cov.waitingOver
      ? t('waitingOver')
      : cov.waitingEndsOn
        ? t('waitingUntil', { date: formatDayMonth(fromApiDate(cov.waitingEndsOn), locale) })
        : t('waitingMonths', { months: formatNumber(months, locale) });
  }
  if (cov.note) hint = hint ? `${hint} · ${cov.note}` : cov.note;
  return (
    <ProgressBar
      tone={months !== null && !cov.waitingOver ? 'warm' : 'data'}
      label={tc(`categories.${categoryKey(cov.category)}`)}
      value={left}
      max={total}
      valueLabel={t('left', { left: amount(left), total: amount(total) })}
      hint={hint}
    />
  );
}

function Body({ policy, onEdit }: { policy: InsurancePolicy; onEdit: () => void }) {
  const t = useTranslations('insurance.coverage');
  const tc = useTranslations('insurance');
  const locale = useLocale() as Locale;
  const tiles = deductibleTiles(policy);
  const caps = policy.coverages.filter(capped);
  const noCoverage = policy.coverages.filter((c) => c.status === 'not_covered');

  return (
    <>
      <p className="ins-intro">
        {t('members', { count: policy.members.length })}
        {policy.planName ? ` · ${policy.planName}` : ''}
      </p>

      {tiles.length > 0 || policy.claimDeadlineDays ? (
        <div className="ins-tiles">
          {tiles.map((x) => (
            <div key={x.category} className="nb-card ins-tile">
              <span className="ins-tile-n">{t('pct', { value: formatNumber(x.pct, locale) })}</span>
              <span className="ins-tile-t">{t('deductibleOf', { category: tc(`categories.${categoryKey(x.category)}`) })}</span>
            </div>
          ))}
          {policy.claimDeadlineDays ? (
            <div className="nb-card ins-tile">
              <span className="ins-tile-n">{formatNumber(policy.claimDeadlineDays, locale)}</span>
              <span className="ins-tile-t">{t('deadlineTile')}</span>
            </div>
          ) : null}
        </div>
      ) : null}

      <section className="nb-card ins-card" aria-labelledby="ins-caps-h">
        <h2 id="ins-caps-h" className="ins-h">
          {t('capsTitle')}
        </h2>
        <p className="ins-hint">{t('capsHint')}</p>
        {caps.length === 0 ? <p className="ins-body">{t('empty')}</p> : caps.map((c) => <CapRow key={c.category} cov={c} />)}
      </section>

      <section className="nb-card ins-card" aria-labelledby="ins-nc-h">
        <h2 id="ins-nc-h" className="ins-h">
          <Icon name="warning" size={16} className="ins-doc-icon" /> {t('notCoveredTitle')}
        </h2>
        <p className="ins-body">{policy.notCoveredNote ?? t('notCoveredDefault')}</p>
        {noCoverage.length > 0 ? (
          <ul className="ins-docs">
            {noCoverage.map((c) => (
              <li key={c.category} className="ins-doc">
                <Icon name="minus" size={16} className="ins-doc-icon" />
                <span>{tc(`categories.${categoryKey(c.category)}`)}</span>
              </li>
            ))}
          </ul>
        ) : null}
      </section>

      <InfoNote icon="lock">{t('selfTrack')}</InfoNote>
      <SecondaryButton variant="text" icon="pencil" onClick={onEdit} {...pilot('ins-coverage.edit')}>
        {t('edit')}
      </SecondaryButton>
    </>
  );
}

/**
 * Insurance coverage (CB-INS-03, nbl_Ins_Coverage) at `/insurance/coverage`: deductible tiles, caps with what is
 * left, waiting notes and the not-covered note — all from what she entered. A back-header screen, no bottom nav.
 */
export function InsuranceCoveragePage() {
  const t = useTranslations('insurance');
  const router = useRouter();
  const query = useInsurancePolicy();
  const policy = query.data?.policy ?? null;

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="ins-skel">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError) {
    body = (
      <EmptyState
        icon="shield"
        title={t('error')}
        action={
          <PrimaryButton icon="refresh" loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  } else if (!policy) {
    // No policy yet: the setup form lives on the home route.
    body = (
      <EmptyState
        icon="shield"
        title={t('setup.title')}
        action={<PrimaryButton onClick={() => router.push('/insurance')}>{t('setup.save')}</PrimaryButton>}
      />
    );
  } else {
    body = <Body policy={policy} onEdit={() => router.push('/insurance')} />;
  }

  return (
    <div className="view ins-page">
      <SkyLayer />
      <div className="scroll ins-scroll">
        <ScreenHeader title={t('coverage.title')} onBack={() => router.push('/insurance')} backLabel={t('back')} />
        {body}
      </div>
    </div>
  );
}
