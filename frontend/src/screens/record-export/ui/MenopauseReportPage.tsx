'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useId, useMemo, useState } from 'react';

import { MAX_REPORT_QUESTION, type ReportSelection, useHealthReport } from '@/entities/health-record';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatLongDate, formatNumber, fromApiDate } from '@/shared/lib/date';
import {
  Card,
  ChipGroup,
  EmptyState,
  HeaderButton,
  Icon,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SectionTitle,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

import { MENO_REPORT_MONTHS, type MenoReportMonths, useMenopauseReport } from '../api/menopause-report';
import { buildMenopauseModel, type MenoRow } from '../model/menopause';
import type { Translate } from '../model/paper';
import { ShareLinkSheet } from './ShareLinkSheet';
import { usePaperModel, usePdfDownload } from './usePaper';

export type MenopauseReportOrigin = 'home' | 'alert';

/**
 * «گزارش برای پزشک» of the menopause mode (`/menopause/report`, CB-MENO-11, nbl_Meno_Report): 1 / 3 / 6 month range,
 * the summary rows, the top-symptom bars, treatment and adherence, and her questions for the doctor. The preview reads
 * GET /menopause/report; the PDF and the 7-day share link are bloom's builder (B-N6-04) over the same window with
 * `sections=menopause` — one PDF renderer, one share flow. The questions never leave the device unless a link is made.
 * A back-header screen: no bottom nav.
 */
export function MenopauseReportPage({ from = 'home' }: { from?: MenopauseReportOrigin }) {
  const t = useTranslations('menopause.report');
  const tp = useTranslations('menopause.report.paper');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const [months, setMonths] = useState<MenoReportMonths>(3);
  const [question, setQuestion] = useState('');
  const [sharing, setSharing] = useState(false);
  const questionId = useId();

  const query = useMenopauseReport(months, loc);
  const data = query.data;
  const model = useMemo(
    () => (data ? buildMenopauseModel(data.report, tp as unknown as Translate, loc) : null),
    [data, tp, loc],
  );

  // The builder's selection over exactly the preview's window (custom `from` = the window start).
  const windowFrom = data?.report.window.from ?? null;
  const selection: ReportSelection = { range: 'custom', from: windowFrom, sections: ['menopause'], question };
  const builder = useHealthReport(selection, !!windowFrom);
  const paper = usePaperModel(builder.data, question);
  const pdf = usePdfDownload();

  const sub = data
    ? t('range', {
        from: formatLongDate(fromApiDate(data.report.window.from), loc),
        to: formatLongDate(fromApiDate(data.report.window.to), loc),
      })
    : undefined;

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="mrp-skel">
        <Skeleton shape="card" className="mrp-skel-card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError || !model || !data) {
    body = (
      <EmptyState
        icon="warning"
        title={t('loadError')}
        action={
          <PrimaryButton icon="refresh" block={false} loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  } else {
    body = (
      <>
        {data.empty ? <p className="mrp-empty">{t('empty')}</p> : null}
        <Card as="section" className="mrp-card" aria-labelledby="mrp-summary">
          <h2 id="mrp-summary" className="mrp-card-title">
            {t('summary')}
          </h2>
          <Rows rows={model.summary} />
        </Card>

        <section className="mrp-sec" aria-labelledby="mrp-symptoms">
          <SectionTitle id="mrp-symptoms" title={tp('symptomsTitle')} />
          <Card className="mrp-bars">
            {model.symptoms.length ? (
              <ul className="mrp-bar-list">
                {model.symptoms.map((s) => (
                  <li key={s.key} className="mrp-bar">
                    <span className="mrp-bar-label">{s.label}</span>
                    <span
                      className="mrp-bar-track"
                      role="meter"
                      aria-label={s.label}
                      aria-valuemin={0}
                      aria-valuemax={100}
                      aria-valuenow={s.percent}
                      aria-valuetext={s.share}
                    >
                      <span className="mrp-bar-fill" style={{ inlineSize: `${s.percent}%` }} />
                    </span>
                    <span className="mrp-bar-share">{s.share}</span>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="mrp-muted">{t('noSymptoms')}</p>
            )}
          </Card>
        </section>

        <section className="mrp-sec" aria-labelledby="mrp-treatment">
          <SectionTitle id="mrp-treatment" title={tp('treatmentTitle')} />
          <Card className="mrp-card">
            {model.treatment.length ? <Rows rows={model.treatment} /> : <p className="mrp-muted">{t('noTreatment')}</p>}
          </Card>
        </section>
      </>
    );
  }

  const ready = !!paper && !builder.isError;

  return (
    <div className="view rx-page mrp-page">
      <SkyLayer />
      <div className="scroll rx-scroll">
        <ScreenHeader
          title={t('title')}
          subtitle={sub}
          onBack={() => router.push(from === 'alert' ? '/menopause/alert' : '/home')}
          backLabel={t('back')}
          action={<HeaderButton icon="share" label={t('shareLink')} onClick={() => setSharing(true)} />}
        />

        <ChipGroup label={t('rangeLabel')} className="mrp-range">
          {MENO_REPORT_MONTHS.map((m) => (
            <PillChip key={m} pressed={months === m} onPressedChange={() => setMonths(m)}>
              {t('months', { count: m, n: formatNumber(m, loc) })}
            </PillChip>
          ))}
        </ChipGroup>

        {body}

        <section className="rx-question mrp-sec">
          <label htmlFor={questionId} className="mrp-question-label">
            {t('questions')}
          </label>
          <div className="rx-question-field mrp-question-field">
            <textarea
              id={questionId}
              className="rx-textarea mrp-textarea"
              rows={4}
              maxLength={MAX_REPORT_QUESTION}
              value={question}
              placeholder={t('questionsPlaceholder')}
              onChange={(e) => setQuestion(e.target.value)}
            />
          </div>
          {question.length > MAX_REPORT_QUESTION - 60 ? (
            <span className="rx-count">
              {t('questionsCount', { count: formatNumber(question.length, loc), max: formatNumber(MAX_REPORT_QUESTION, loc) })}
            </span>
          ) : null}
        </section>

        <Card className="rx-note" padding="sm">
          <Icon name="shield" size={18} className="rx-note-icon" />
          <p>{t('note')}</p>
        </Card>

        {pdf.failed || builder.isError ? (
          <p className="rx-error" role="alert">
            {t('downloadError')}
          </p>
        ) : null}

        <div className="rx-actions mrp-actions">
          <PrimaryButton
            icon="download"
            loading={pdf.busy || (!!windowFrom && builder.isPending)}
            disabled={!ready}
            onClick={() => paper && void pdf.run(paper)}
          >
            {pdf.busy ? t('downloading') : t('download')}
          </PrimaryButton>
        </div>
      </div>
      {windowFrom ? <ShareLinkSheet open={sharing} onClose={() => setSharing(false)} selection={selection} /> : null}
    </div>
  );
}

function Rows({ rows }: { rows: MenoRow[] }) {
  return (
    <dl className="mrp-rows">
      {rows.map((r) => (
        <div key={r.key} className="mrp-row">
          <dt className="mrp-row-label">{r.label}</dt>
          <dd className="mrp-row-value">{r.value}</dd>
        </div>
      ))}
    </dl>
  );
}
