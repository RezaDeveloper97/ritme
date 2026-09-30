'use client';

import { useLocale, useTranslations } from 'next-intl';

import { highlightGlyph, type HighlightTone, type PregnancyWeek } from '@/entities/pregnancy';
import { type Locale, Link, useDirection } from '@/shared/i18n';
import { formatLongDate, formatNumber, fromApiDate } from '@/shared/lib/date';
import { Card, Icon, IconCircle, InfoNote, type Tone } from '@/shared/ui';
import { FetusSize } from '@/shared/ui/illustrations';
import { PregnancyCareChecklist } from '@/widgets/pregnancy-care-checklist';

import { parseWeekStat, type WeekStat } from '../model/stats';

const TONE: Record<HighlightTone, Tone> = { brand: 'brand', pink: 'bloom', teal: 'data' };

/** The stacked sections of PregFull_Week: hero, stats, baby, body, tasks, warning, review. */
export function WeekSections({ data }: { data: PregnancyWeek }) {
  const t = useTranslations('pregnancyV2.week');
  const tv = useTranslations('pregnancyV2');
  const locale = useLocale() as Locale;
  const dir = useDirection();
  const d = data.details;
  const stats = [
    { key: 'length', value: d.length, unit: t('stats.cm') },
    { key: 'weight', value: d.weight, unit: t('stats.g') },
    { key: 'heart', value: d.heartRate, unit: t('stats.bpm') },
  ].filter((s): s is { key: 'length' | 'weight' | 'heart'; value: string; unit: string } => !!s.value);
  const statText = (s: WeekStat) =>
    s.kind === 'range'
      ? t('stats.range', { from: s.from, to: s.to })
      : s.kind === 'less'
        ? t('stats.less', { value: s.value })
        : s.kind === 'approx'
          ? t('stats.approx', { value: s.value })
          : s.value;
  const hasBody = d.bodySymptoms.length > 0 || !!d.bodyText;

  return (
    <>
      <Card as="section" className="pgn-week-hero" aria-labelledby="pgn-week-size">
        <span className="pgn-week-disc">
          <FetusSize illustrationKey={d.illustrationKey} label={t('illustration')} size={88} />
        </span>
        <h2 id="pgn-week-size" className="pgn-week-size">
          {d.sizeLabel ? t('sizePill', { item: d.sizeLabel }) : t('tabWeek', { week: formatNumber(data.week, locale) })}
        </h2>
        {d.headline && <p className="pgn-week-headline">{d.headline}</p>}
      </Card>

      {stats.length > 0 && (
        <>
          <dl className={stats.length === 3 ? 'pgn-stats is-3' : 'pgn-stats'}>
            {stats.map((s) => (
              <Card key={s.key} className="pgn-stat">
                <dt className="pgn-stat-label">{t(`stats.labels.${s.key}`)}</dt>
                <dd className="pgn-stat-value">{statText(parseWeekStat(s.value, locale))}</dd>
                <dd className="pgn-stat-unit">{s.unit}</dd>
              </Card>
            ))}
          </dl>
          <InfoNote>{t('averagesNote')}</InfoNote>
        </>
      )}

      <Card as="section" className="pgn-sect" aria-labelledby="pgn-week-baby">
        <h2 id="pgn-week-baby" className="pgn-sect-title">
          {t('babyTitle')}
        </h2>
        {d.highlights.length === 0 ? (
          <p className="pgn-muted">{t('empty')}</p>
        ) : (
          <ul className="pgn-rows">
            {d.highlights.map((h, i) => (
              <li key={`${h.title}-${i}`} className="pgn-row">
                <IconCircle icon={highlightGlyph(h.icon)} tone={TONE[h.tone]} size="sm" />
                <span className="pgn-row-text">
                  <b className="pgn-row-title">{h.title}</b>
                  <span className="pgn-row-desc">{h.body}</span>
                </span>
              </li>
            ))}
          </ul>
        )}
      </Card>

      {hasBody && (
        <Card as="section" className="pgn-sect" aria-labelledby="pgn-week-body">
          <h2 id="pgn-week-body" className="pgn-sect-title">
            {t('bodyTitle')}
          </h2>
          {d.bodySymptoms.length > 0 && (
            <ul className="pgn-feel">
              {d.bodySymptoms.map((s) => (
                <li key={s} className="pgn-feel-chip">
                  {s}
                </li>
              ))}
            </ul>
          )}
          {d.bodyText && <p className="pgn-body-text">{d.bodyText}</p>}
          <Link href="/pregnancy/log" className="pgn-link">
            {t('logToday')}
            <Icon name={dir === 'rtl' ? 'chevronLeft' : 'chevronRight'} size={15} strokeWidth={2.2} />
          </Link>
        </Card>
      )}

      <PregnancyCareChecklist week={data.week} tasks={data.tasks} title={t('tasksTitle')} variant="week" />

      {d.warning && (
        <section className="pgn-warn" role="note">
          <Icon name="warning" size={20} className="pgn-warn-icon" />
          <p className="pgn-warn-text">
            <b>{t('warningLead')}</b> {d.warning}
          </p>
        </section>
      )}

      <InfoNote
        icon="doctor"
        source={
          d.sources.length > 0 ? (
            <details className="pgn-sources">
              <summary>{t('sources')}</summary>
              <ul>
                {d.sources.map((s, i) => (
                  <li key={`${s.title}-${i}`}>
                    {s.url ? (
                      <a href={s.url} target="_blank" rel="noopener noreferrer">
                        {s.title}
                      </a>
                    ) : (
                      s.title
                    )}
                  </li>
                ))}
              </ul>
            </details>
          ) : undefined
        }
      >
        {d.reviewerName ? (
          <>
            {t('reviewedBy', { name: d.reviewerName })}
            {d.reviewedAt && (
              <>
                {tv('common.separator')}
                {formatLongDate(fromApiDate(d.reviewedAt.slice(0, 10)), locale)}
              </>
            )}
          </>
        ) : (
          t('notReviewed')
        )}
      </InfoNote>
    </>
  );
}
