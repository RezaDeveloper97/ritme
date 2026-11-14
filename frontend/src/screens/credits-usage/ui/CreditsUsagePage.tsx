'use client';

import { useLocale, useTranslations } from 'next-intl';

import {
  formatCredits,
  planShare,
  useAiCredits,
  useAiUsage,
  type AiCredits,
  type AiUsagePage,
} from '@/entities/ai-credit';
import { useBillingChannel } from '@/shared/billing';
import { useRouter, type Locale } from '@/shared/i18n';
import { formatDayMonth, formatLongDate, formatNumber, formatTimeOfDay, fromApiDate, toParts } from '@/shared/lib/date';
import {
  Art3d,
  BarChart,
  Card,
  EmptyState,
  ListGroup,
  ListRow,
  PrimaryButton,
  ProgressBar,
  ProgressRing,
  ScreenHeader,
  SecondaryButton,
  SectionTitle,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';
import { useNavMode } from '@/widgets/bottom-nav';

import { costRows, dayRows, extraKept, featureRows, usageStateOf } from '../model/view';

function Hero({ credits }: { credits: AiCredits }) {
  const t = useTranslations('credits.usage');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const state = usageStateOf(credits);
  const remaining = formatCredits(credits.plan.remaining, loc);
  const total = formatCredits(credits.plan.total, loc);
  const until = credits.periodEndsAt ? formatLongDate(fromApiDate(credits.periodEndsAt.slice(0, 10)), loc) : null;

  if (state === 'no_subscription') {
    return (
      <Card className="cr-hero is-empty">
        <Art3d name="credits" size={72} />
        <h2 className="cr-hero-title">{t('noSubscription.title')}</h2>
        <p className="cr-hero-text">{t('noSubscription.body')}</p>
        <PrimaryButton block={false} onClick={() => router.push('/plus')}>
          {t('noSubscription.cta')}
        </PrimaryButton>
      </Card>
    );
  }
  return (
    <Card className="cr-hero">
      <ProgressRing
        value={planShare(credits)}
        size={176}
        thickness={14}
        tone="warm"
        label={t('hero.caption')}
        valueText={t('hero.spoken', { remaining, total })}
      >
        <span className="cr-ring-num">{remaining}</span>
        <span className="cr-ring-cap">{t('hero.of', { total })}</span>
      </ProgressRing>
      {state === 'exhausted' ? (
        <>
          <h2 className="cr-hero-title">{t('exhausted.title')}</h2>
          <p className="cr-hero-text">{t('exhausted.body')}</p>
        </>
      ) : null}
      {until ? <p className="cr-hero-until">{t('hero.until', { date: until })}</p> : null}
    </Card>
  );
}

function ExtraCard({ credits }: { credits: AiCredits }) {
  const t = useTranslations('credits.usage');
  const loc = useLocale() as Locale;
  const kept = extraKept(credits);
  return (
    <Card className="cr-extra">
      <div className="cr-extra-head">
        <h2 className="cr-extra-title">{t('extra.title')}</h2>
        <span className="cr-extra-value">{t('extra.value', { count: formatCredits(credits.extra.remaining, loc) })}</span>
      </div>
      <p className="cr-extra-note">{kept ? t('noSubscription.locked') : t('extra.never')}</p>
      {!kept ? <p className="cr-extra-note is-soft">{t('extra.usedAfter')}</p> : null}
    </Card>
  );
}

function UsageBlocks({ usage }: { usage: AiUsagePage[] }) {
  const t = useTranslations('credits.usage');
  const loc = useLocale() as Locale;
  const first = usage[0];
  const features = featureRows(first.byFeature);
  const days = dayRows(first.byDay);
  const max = Math.max(1, ...features.map((f) => f.credits));
  return (
    <>
      <section className="cr-sec" aria-labelledby="cr-last30">
        <SectionTitle id="cr-last30" title={t('last30.title')} />
        <Card className="cr-card">
          <p className="cr-total">{t('last30.total', { count: formatCredits(first.total, loc) })}</p>
          {features.length ? (
            <div className="cr-bars">
              {features.map((f) => (
                <ProgressBar
                  key={f.feature}
                  tone="warm"
                  value={f.credits}
                  max={max}
                  label={f.title}
                  valueLabel={t('last30.uses', { count: formatNumber(f.uses, loc) })}
                  valueText={t('costs.value', { count: formatCredits(f.credits, loc) })}
                />
              ))}
            </div>
          ) : null}
          {days.length ? (
            <BarChart
              className="cr-chart"
              label={t('last30.chart')}
              bars={days.map((d) => ({
                label: formatNumber(toParts(fromApiDate(d.date), loc).day, loc),
                value: d.credits,
                tone: 'warm' as const,
                highlight: true,
              }))}
            />
          ) : null}
        </Card>
      </section>
      <HistoryBlock usage={usage} />
    </>
  );
}

function HistoryBlock({ usage }: { usage: AiUsagePage[] }) {
  const t = useTranslations('credits.usage.history');
  const loc = useLocale() as Locale;
  const more = useAiUsage();
  const items = usage.flatMap((p) => p.history.items);
  return (
    <section className="cr-sec" aria-labelledby="cr-history">
      <SectionTitle id="cr-history" title={t('title')} />
      {items.length === 0 ? (
        <Card className="cr-card">
          <EmptyState icon="history" title={t('empty')} body={t('emptyBody')} />
        </Card>
      ) : (
        <ListGroup>
          {items.map((item) => {
            const at = new Date(item.at);
            return (
              <ListRow
                key={item.id}
                icon="sparkle"
                iconTone="warm"
                title={item.title}
                description={`${formatDayMonth(at, loc)} · ${formatTimeOfDay(at, loc)}`}
                value={t('credits', { count: formatCredits(item.credits, loc) })}
              />
            );
          })}
        </ListGroup>
      )}
      {more.hasNextPage ? (
        <SecondaryButton loading={more.isFetchingNextPage} onClick={() => void more.fetchNextPage()}>
          {t('more')}
        </SecondaryButton>
      ) : null}
    </section>
  );
}

/**
 * `/profile/credits` «مصرف اعتبار»: the period's credits as a ring, the extra credits that never expire, what each
 * feature costs, the last 30 days and the paginated history. Back goes to /profile (explicit route, §4.2).
 */
export function CreditsUsagePage() {
  const t = useTranslations('credits');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const credits = useAiCredits();
  const usage = useAiUsage();
  const navMode = useNavMode().mode;
  const { channel } = useBillingChannel();
  const canBuy = navMode !== 'teen' && navMode !== 'companion' && channel === 'web';

  let body;
  if (credits.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="cr-body">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (credits.isError || !credits.data) {
    body = (
      <EmptyState
        icon="sparkle"
        title={t('error.title')}
        body={t('error.body')}
        action={
          <PrimaryButton icon="refresh" loading={credits.isFetching} onClick={() => void credits.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  } else {
    const data = credits.data;
    const costs = costRows(data);
    body = (
      <div className="cr-body">
        <Hero credits={data} />
        <ExtraCard credits={data} />
        {canBuy ? (
          <PrimaryButton onClick={() => router.push('/plus/credits')}>{t('usage.buy')}</PrimaryButton>
        ) : null}
        {costs.length ? (
          <section className="cr-sec" aria-labelledby="cr-costs">
            <SectionTitle id="cr-costs" title={t('usage.costs.title')} />
            <ListGroup>
              {costs.map((f) => (
                <ListRow
                  key={f.key}
                  icon="sparkle"
                  iconTone="warm"
                  title={f.title}
                  value={t('usage.costs.value', { count: formatCredits(f.creditCost, loc) })}
                />
              ))}
            </ListGroup>
          </section>
        ) : null}
        {usage.isPending ? (
          <SkeletonGroup label={t('loading')}>
            <Skeleton shape="card" />
          </SkeletonGroup>
        ) : usage.isError || !usage.data ? (
          <EmptyState
            icon="warning"
            title={t('error.title')}
            action={
              <SecondaryButton block={false} onClick={() => void usage.refetch()}>
                {t('retry')}
              </SecondaryButton>
            }
          />
        ) : (
          <UsageBlocks usage={usage.data.pages} />
        )}
      </div>
    );
  }

  return (
    <div className="view cr-page">
      <SkyLayer />
      <div className="scroll cr-scroll">
        <ScreenHeader title={t('usage.title')} onBack={() => router.push('/profile')} backLabel={t('back')} />
        {body}
      </div>
    </div>
  );
}
