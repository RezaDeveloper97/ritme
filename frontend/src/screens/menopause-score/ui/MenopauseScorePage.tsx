'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import {
  type MenopausePattern,
  type MenopauseScoreHistory,
  useMenopausePatterns,
  useMenopauseScores,
} from '@/entities/menopause';
import { Link, type Locale, useDirection, useRouter } from '@/shared/i18n';
import { formatMonthLabel, formatNumber, fromApiDate, monthName, toParts } from '@/shared/lib/date';
import {
  Card,
  EmptyState,
  Icon,
  IconCircle,
  LineChart,
  PrimaryButton,
  ProgressBar,
  ScreenHeader,
  SecondaryButton,
  SectionTitle,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
} from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

import {
  bandSegments,
  bandTone,
  domainTone,
  filledThisMonth,
  hrtNote,
  isKnownDomain,
  markerPercent,
  QUESTIONNAIRE_PATH,
} from '../model/score';

const MONTHS = 6;

/**
 * «امتیاز علائم» (`/menopause/score`, CB-MENO-08, nbl_Meno_Score) — the
 * menopause mode tab «علائم». Header with the band bar, domain breakdown, the
 * 6-month chart with the HRT note, patterns from her own logs (never a
 * diagnosis), bloom's symptom analysis link and the monthly questionnaire CTA.
 */
export function MenopauseScorePage() {
  const t = useTranslations('menopause.score');
  const router = useRouter();
  const query = useMenopauseScores(MONTHS);

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="msc-skel">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError) {
    body = (
      <EmptyState
        icon="chart"
        title={t('loadError')}
        action={
          <PrimaryButton icon="refresh" loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  } else {
    body = <ScoreBody history={query.data} />;
  }

  return (
    <div className="view msc-page">
      <SkyLayer />
      <div className="scroll msc-scroll">
        <ScreenHeader title={t('title')} subtitle={t('sub')} onBack={() => router.push('/home')} backLabel={t('back')} />
        {body}
      </div>
      <BottomNav />
    </div>
  );
}

function ScoreBody({ history }: { history: MenopauseScoreHistory }) {
  const t = useTranslations('menopause.score');
  const router = useRouter();
  const filled = filledThisMonth(history);
  const latest = history.latest;

  return (
    <div className="msc-body">
      {latest ? (
        <>
          <ScoreHeader history={history} />
          <Domains history={history} />
        </>
      ) : (
        <Card className="msc-empty">
          <EmptyState icon="chart" title={t('empty.title')} body={t('empty.body')} />
        </Card>
      )}
      {history.trend.some((p) => p.total !== null) ? <Trend history={history} /> : null}
      <Patterns />
      {filled ? (
        <SecondaryButton className="msc-cta" onClick={() => router.push(QUESTIONNAIRE_PATH)}>
          {t('refill')}
        </SecondaryButton>
      ) : (
        <PrimaryButton className="msc-cta" onClick={() => router.push(QUESTIONNAIRE_PATH)}>
          {t('fill')}
        </PrimaryButton>
      )}
    </div>
  );
}

function ScoreHeader({ history }: { history: MenopauseScoreHistory }) {
  const t = useTranslations('menopause.score');
  const locale = useLocale() as Locale;
  const latest = history.latest!;
  const parts = toParts(fromApiDate(latest.month), locale);
  const segments = bandSegments(history.bands);
  const total = formatNumber(latest.total, locale);
  const max = formatNumber(latest.max, locale);

  return (
    <Card className="msc-head">
      <div className="msc-head-top">
        <div className="msc-head-num">
          <span className="msc-head-month">{formatMonthLabel(parts.year, parts.month, locale)}</span>
          <p className="msc-head-score">
            <b className="msc-total">{total}</b>
            <span className="msc-of">{t('of', { max })}</span>
          </p>
        </div>
        {latest.band?.title ? (
          <StatusPill tone={bandTone(latest.band.code, history.bands)} className="msc-band-pill">
            {latest.band.title}
          </StatusPill>
        ) : null}
      </div>
      {segments.length ? (
        <div className="msc-bands">
          <div
            className="msc-band-bar"
            role="img"
            aria-label={t('marker', { total, max, band: latest.band?.title ?? '' })}
          >
            {segments.map((s) => (
              <span
                key={s.code}
                className={clsx('msc-band-seg', `nb-tone-${s.tone}`, s.code === latest.band?.code && 'is-on')}
                style={{ flexGrow: s.share }}
              />
            ))}
            <span className="msc-band-marker" style={{ insetInlineStart: `${markerPercent(latest.total, history.bands)}%` }} />
          </div>
          <div className="msc-band-labels" aria-hidden>
            {segments.map((s) => (
              <span key={s.code} className={clsx('msc-band-label', s.code === latest.band?.code && 'is-on')}>
                {s.title}
              </span>
            ))}
          </div>
        </div>
      ) : null}
    </Card>
  );
}

function Domains({ history }: { history: MenopauseScoreHistory }) {
  const t = useTranslations('menopause.score');
  const locale = useLocale() as Locale;
  const latest = history.latest!;
  return (
    <section className="msc-sec" aria-labelledby="msc-domains">
      <SectionTitle id="msc-domains" title={t('domainsTitle')} />
      <Card className="msc-domains">
        {latest.domains.map((d) => (
          <ProgressBar
            key={d.code}
            value={d.score}
            max={d.max}
            tone={domainTone(d.code)}
            label={isKnownDomain(d.code) ? t(`domains.${d.code}`) : d.code}
            valueLabel={t('domainValue', { score: formatNumber(d.score, locale), max: formatNumber(d.max, locale) })}
          />
        ))}
      </Card>
    </section>
  );
}

function Trend({ history }: { history: MenopauseScoreHistory }) {
  const t = useTranslations('menopause.score');
  const locale = useLocale() as Locale;
  const months = formatNumber(history.trend.length || MONTHS, locale);
  const labels = history.trend.map((p) => monthName(toParts(fromApiDate(p.month), locale).month, locale));
  const note = hrtNote(history.hrt?.change);
  const hrtMonth = history.hrt ? monthName(toParts(fromApiDate(history.hrt.startedOn), locale).month, locale) : '';

  return (
    <section className="msc-sec" aria-labelledby="msc-trend">
      <SectionTitle id="msc-trend" title={t('chartTitle', { months })} />
      <Card className="msc-trend">
        <LineChart
          label={t('chartLabel', { months })}
          className="msc-chart"
          height={110}
          min={0}
          max={Math.max(history.max / 2, ...history.trend.map((p) => p.total ?? 0))}
          series={[{ values: history.trend.map((p) => p.total), tone: 'brand', points: true }]}
          xLabels={labels}
        />
        <table className="sr-only">
          <caption>{t('chartLabel', { months })}</caption>
          <thead>
            <tr>
              <th scope="col">{t('chartTableMonth')}</th>
              <th scope="col">{t('chartTableScore')}</th>
            </tr>
          </thead>
          <tbody>
            {history.trend.map((p, i) => (
              <tr key={p.month}>
                <th scope="row">{labels[i]}</th>
                <td>{p.total === null ? t('chartNoValue') : formatNumber(p.total, locale)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        {note ? (
          <p className="msc-hrt">
            {note.kind === 'same'
              ? t('hrt.same', { month: hrtMonth })
              : t(`hrt.${note.kind}`, { month: hrtMonth, points: formatNumber(note.points, locale) })}
          </p>
        ) : null}
      </Card>
    </section>
  );
}

function patternLook(p: MenopausePattern) {
  return p.key === 'night_sweats_fatigue'
    ? ({ icon: 'moon', tone: 'brand' } as const)
    : ({ icon: 'flame', tone: 'danger' } as const);
}

function Patterns() {
  const t = useTranslations('menopause.score');
  const locale = useLocale() as Locale;
  const rtl = useDirection() === 'rtl';
  const query = useMenopausePatterns();
  const data = query.data;
  const found = data?.items.filter((p) => p.found && p.text) ?? [];

  let content;
  if (query.isPending) {
    content = <Skeleton shape="card" />;
  } else if (query.isError || !data) {
    content = <p className="msc-pat-note">{t('patterns.error')}</p>;
  } else if (found.length) {
    content = found.map((p) => {
      const look = patternLook(p);
      return (
        <div key={`${p.key}-${p.trigger ?? ''}`} className="msc-pat">
          <IconCircle icon={look.icon} tone={look.tone} size="md" />
          <p className="msc-pat-text">{p.text}</p>
        </div>
      );
    });
  } else if (data.daysLogged < data.minDays) {
    content = (
      <p className="msc-pat-note">
        {t('patterns.notEnough', { days: formatNumber(data.daysLogged, locale), min: formatNumber(data.minDays, locale) })}
      </p>
    );
  } else {
    content = <p className="msc-pat-note">{t('patterns.none')}</p>;
  }

  return (
    <section className="msc-sec" aria-labelledby="msc-patterns">
      <div className="msc-sect-head">
        <SectionTitle id="msc-patterns" title={data?.disclaimer?.title || t('patterns.title')} />
        <p className="msc-sect-sub">{data?.disclaimer?.body || t('patterns.disclaimer')}</p>
      </div>
      <Card className="msc-patterns">
        {content}
        <Link href="/analysis/symptoms" className="msc-analysis">
          <Icon name="chart" size={18} />
          <span className="msc-analysis-label">{t('analysis')}</span>
          <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} className="msc-analysis-chev" />
        </Link>
      </Card>
    </section>
  );
}
