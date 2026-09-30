'use client';

import { useQueryClient } from '@tanstack/react-query';
import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { type MouseEvent, useMemo, useState } from 'react';

import {
  type CalendarVisit,
  type CareItem,
  type PregnancyCalendar,
  type VisitStage,
  VISIT_STAGES,
  fetchPregnancyReport,
  pregnancyKeys,
  usePregnancyCalendar,
} from '@/entities/pregnancy';
import { REMIND_BEFORE, type RemindBefore } from '@/entities/care-reminder';
import { useUpdateAppointment } from '@/features/manage-appointment';
import { type Locale, Link, useDirection, useRouter } from '@/shared/i18n';
import {
  addDays,
  diffInDays,
  formatDayMonth,
  formatLongDate,
  formatMonthLabel,
  formatNumber,
  formatWeekday,
  fromApiDate,
  monthMatrix,
  shiftMonth,
  toApiDate,
  today,
  todayParts,
  weekdayLabels,
} from '@/shared/lib/date';
import { withHandoff } from '@/shared/lib/handoff';
import { type PdfBlock, loadPdfGenerator, shareOrDownloadFile } from '@/shared/lib/pdf';
import { AppSheet } from '@/shared/sheet';
import {
  Card,
  EmptyState,
  HeaderButton,
  Icon,
  IconCircle,
  InfoNote,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  type Tone,
} from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

import { BOOK_HREF, REPORT_DAYS, bookPrefill, directionsHref, monthKey, nextStage, stageProgress } from '../model/view';

const NEW_VISIT_HREF = '/reminders/appointment/new?kind=in_person&return_to=/pregnancy/calendar';

/** «تقویم و ویزیت‌ها» — `/pregnancy/calendar` (PregFull_Calendar). */
export function PregnancyCalendarPage() {
  const t = useTranslations('pregnancyV2');
  const locale = useLocale() as Locale;
  const dir = useDirection();
  const router = useRouter();
  const [ym, setYm] = useState(() => {
    const p = todayParts(locale);
    return { year: p.year, month: p.month };
  });
  const [picked, setPicked] = useState<string | null>(null);
  const query = usePregnancyCalendar(monthKey(ym.year, ym.month));
  const data = query.data;
  // Until the user picks a day, the month opens on its next visit (Calendar artboard), else today.
  const selected = picked ?? defaultSelection(data ?? null, toApiDate(today()));

  const move = (delta: number) => setYm((c) => shiftMonth(c.year, c.month, delta));

  const month = (
    <Card as="section" className="pgn-sect" aria-label={t('calendar.title')}>
      <div className="pgn-month-head">
        <button type="button" className="pgn-step" aria-label={t('calendar.prevMonth')} onClick={() => move(-1)}>
          <Icon name={dir === 'rtl' ? 'chevronRight' : 'chevronLeft'} size={18} />
        </button>
        <div className="pgn-month-title">
          <b>{data?.monthLabel ?? formatMonthLabel(ym.year, ym.month, locale)}</b>
          {data?.weekRange && (
            <span className="pgn-caption">
              {t('calendar.weekRange', { from: data.weekRange.from, to: data.weekRange.to })}
            </span>
          )}
        </div>
        <button type="button" className="pgn-step" aria-label={t('calendar.nextMonth')} onClick={() => move(1)}>
          <Icon name={dir === 'rtl' ? 'chevronLeft' : 'chevronRight'} size={18} />
        </button>
      </div>
      <MonthGrid year={ym.year} month={ym.month} data={data ?? null} selected={selected} onPick={setPicked} />
      <Legend />
    </Card>
  );

  let body: React.ReactNode;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('common.loading')} className="pgn-skel">
        <Skeleton shape="card" className="pgn-skel-hero" />
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError) {
    body = (
      <Card className="pgn-state" role="alert">
        <span className="pgn-state-disc" aria-hidden>
          <Icon name="warning" size={24} />
        </span>
        <p className="pgn-state-text">{t('common.loadError')}</p>
        <SecondaryButton icon="refresh" block={false} onClick={() => void query.refetch()}>
          {t('common.retry')}
        </SecondaryButton>
      </Card>
    );
  } else if (data) {
    body = <CalendarBody data={data} selected={selected} month={month} />;
  } else {
    body = (
      <Card>
        <EmptyState
          icon="heart"
          title={t('common.notActive')}
          action={
            <Link href="/pregnancy/setup" className="nb-btn is-primary is-block">
              {t('today.setupCta')}
            </Link>
          }
        />
      </Card>
    );
  }

  return (
    <div className="view preg-page pgn-screen">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader
          title={t('calendar.screenTitle')}
          onBack={() => router.push('/pregnancy')}
          backLabel={t('common.back')}
          action={<HeaderButton icon="plus" label={t('calendar.newVisitLabel')} onClick={() => router.push(NEW_VISIT_HREF)} />}
        />
        <div className="pgn-body">{body}</div>
      </div>
      <BottomNav />
    </div>
  );
}

function MonthGrid({
  year,
  month,
  data,
  selected,
  onPick,
}: {
  year: number;
  month: number;
  data: PregnancyCalendar | null;
  selected: string;
  onPick: (date: string) => void;
}) {
  const t = useTranslations('pregnancyV2.calendar');
  const locale = useLocale() as Locale;
  const rows = useMemo(() => monthMatrix(year, month, locale), [year, month, locale]);
  const markers = useMemo(() => new Map((data?.days ?? []).map((d) => [d.date, d])), [data]);
  const todayKey = toApiDate(today());
  const monthLabel = formatMonthLabel(year, month, locale);

  return (
    <>
      <div className="pgn-grid pgn-grid-wd" aria-hidden>
        {weekdayLabels(locale).map((w) => (
          <span key={w}>{w}</span>
        ))}
      </div>
      <div className="pgn-grid">
        {rows.flat().map((cell, i) => {
          if (!cell) return <span key={`e${i}`} aria-hidden />;
          const key = toApiDate(cell.date);
          const m = markers.get(key);
          const isToday = m ? m.isToday : key === todayKey;
          const isSel = key === selected;
          const aria = [
            `${formatNumber(cell.day, locale)} ${monthLabel}`,
            m?.hasVisit ? t('legend.visit') : null,
            m?.weekStart != null ? t('weekStart', { week: m.weekStart }) : null,
          ]
            .filter(Boolean)
            .join('، ');
          return (
            <button
              key={key}
              type="button"
              aria-label={aria}
              aria-pressed={isSel}
              aria-current={isToday ? 'date' : undefined}
              onClick={() => onPick(key)}
              className={clsx('pgn-day', isToday && 'is-today', m?.weekStart != null && 'is-week-start')}
            >
              <span>{formatNumber(cell.day, locale)}</span>
              <span aria-hidden className={clsx('pgn-day-dot', m?.hasVisit && 'is-on')} />
            </button>
          );
        })}
      </div>
    </>
  );
}

function Legend() {
  const t = useTranslations('pregnancyV2.calendar.legend');
  return (
    <div className="pgn-legend">
      <span>
        <span className="pgn-legend-mark is-today" aria-hidden />
        {t('today')}
      </span>
      <span>
        <span className="pgn-legend-mark is-visit" aria-hidden />
        {t('visit')}
      </span>
      <span>
        <span className="pgn-legend-mark is-week" aria-hidden />
        {t('weekStart')}
      </span>
    </div>
  );
}

function CalendarBody({ data, selected, month }: { data: PregnancyCalendar; selected: string; month: React.ReactNode }) {
  const t = useTranslations('pregnancyV2.calendar');
  return (
    <>
      {data.nextVisit && <NextVisitCard visit={data.nextVisit} />}
      {data.carePlan.length > 0 && (
        <Card as="section" className="pgn-sect" aria-labelledby="pgn-care-plan">
          <h2 id="pgn-care-plan" className="pgn-sect-title">
            {t('carePlanTitle')}
          </h2>
          <ul className="pgn-rows">
            {data.carePlan.map((item) => (
              <CareRow key={item.key} item={item} />
            ))}
          </ul>
        </Card>
      )}
      {month}
      <SelectedDay data={data} date={selected} />
      <ReportButton />
      {data.sourceNote && <InfoNote>{data.sourceNote}</InfoNote>}
    </>
  );
}

function SelectedDay({ data, date }: { data: PregnancyCalendar; date: string }) {
  const t = useTranslations('pregnancyV2.calendar');
  const tCommon = useTranslations('pregnancyV2.common');
  const sep = tCommon('separator');
  const locale = useLocale() as Locale;
  const marker = data.days.find((d) => d.date === date);
  const visits = data.visits.filter((v) => v.date === date);
  const title = [formatDayMonth(fromApiDate(date), locale), marker?.weekStart != null ? t('weekStart', { week: marker.weekStart }) : null]
    .filter(Boolean)
    .join(sep);
  // Each visit's parts are joined by `sep`, so the visits need a stronger mark
  // between them (fa «؛») — the same one twice would blur where a visit ends.
  const sub = visits.length
    ? visits
        .map((v) => [v.title, v.time ? formatNumber(v.time, locale) : null].filter(Boolean).join(sep))
        .join(tCommon('groupSeparator'))
    : t('dayEmpty');

  return (
    <Card className="pgn-row pgn-day-card" aria-live="polite">
      <IconCircle icon="calendar" tone="brand" size="md" />
      <span className="pgn-row-text">
        <b className="pgn-row-title">{title}</b>
        <span className="pgn-row-desc">{sub}</span>
      </span>
      {!visits.length && (
        <Link href={`${NEW_VISIT_HREF}&date=${date}`} aria-label={t('newVisitLabel')} className="pgn-icon-link">
          <Icon name="plus" size={18} strokeWidth={2.2} />
        </Link>
      )}
    </Card>
  );
}

function useSetStage() {
  const queryClient = useQueryClient();
  const update = useUpdateAppointment();
  const run = async (id: number, stage: VisitStage, resultNote?: string) => {
    try {
      await update.mutateAsync({
        id,
        patch: resultNote !== undefined ? { stage, resultNote } : { stage },
      });
      // The feature invalidates the care lists; the pregnancy calendar is ours.
      await queryClient.invalidateQueries({ queryKey: pregnancyKeys.v2.calendarAll() });
      return true;
    } catch {
      return false;
    }
  };
  return { busy: update.isPending, error: update.isError, run };
}

function NextVisitCard({ visit }: { visit: CalendarVisit }) {
  const t = useTranslations('pregnancyV2');
  const tc = useTranslations('care.appointmentForm.remindOptions');
  const locale = useLocale() as Locale;
  const date = fromApiDate(visit.date);
  const stage = useSetStage();
  const [noteOpen, setNoteOpen] = useState(false);
  const [note, setNote] = useState(visit.resultNote ?? '');
  const filled = stageProgress(visit.stage);
  const next = nextStage(visit.stage);
  const canAdvance = visit.appointmentId != null && next != null && !stage.busy;

  const advance = () => {
    if (!canAdvance || visit.appointmentId == null || next == null) return;
    if (next === 'result') setNoteOpen(true);
    else void stage.run(visit.appointmentId, next);
  };
  const saveNote = async () => {
    if (visit.appointmentId == null) return;
    if (await stage.run(visit.appointmentId, 'result', note)) setNoteOpen(false);
  };

  // Weekday · time · age on that day; the date itself is already on the tile (design audit E4).
  const weekday = formatWeekday(date, locale);
  const age = visit.ageLabel ?? visit.weekLabel;
  const meta =
    visit.time && age
      ? t('calendar.visitMeta', { weekday, time: formatNumber(visit.time, locale), week: age })
      : [weekday, visit.time ? formatNumber(visit.time, locale) : null, age].filter(Boolean).join(t('common.separator'));
  const who = [visit.doctor, visit.place].filter(Boolean).join(t('common.separator'));
  const remind = visit.remindBefore && (REMIND_BEFORE as readonly string[]).includes(visit.remindBefore)
    ? tc(visit.remindBefore as RemindBefore)
    : null;

  const lines = [meta, visit.daysUntil != null ? t('calendar.nextVisit', { days: visit.daysUntil }) : null]
    .filter(Boolean)
    .join(t('common.separator'));

  return (
    <>
      <Card as="section" className="pgn-sect pgn-visit" aria-labelledby="pgn-next-visit">
        <div className="pgn-sect-head">
          <h2 id="pgn-next-visit" className="pgn-sect-title">
            {visit.title}
          </h2>
          <span className="pgn-opill nb-tone-brand">{visit.stage
              ? t(`calendar.stages.${visit.stage}`)
              : visit.appointmentId != null
                ? t('calendar.stages.booked')
                : t('calendar.states.to_book')}</span>
        </div>
        <p className="pgn-visit-meta">
          <span className="pgn-visit-date">{formatDayMonth(date, locale)}</span>
          {t('common.separator')}
          {lines}
          {who && (
            <>
              <br />
              {who}
            </>
          )}
        </p>
        {visit.prep && (
          <p className="pgn-visit-prep">
            <b>{t('calendar.prep')}</b> {visit.prep}
          </p>
        )}

        <button
          type="button"
          className="pgn-stages"
          aria-label={t('calendar.stagesLabel')}
          disabled={!canAdvance}
          onClick={advance}
        >
          <span className="pgn-stages-bars">
            {VISIT_STAGES.map((s, i) => (
              <span key={s} aria-hidden className={clsx('pgn-stages-bar', i < filled && 'is-on')} />
            ))}
          </span>
          <span className="pgn-stages-labels">
            {VISIT_STAGES.map((s, i) => (
              <span key={s} className={clsx(i < filled && 'is-on')}>
                {t(`calendar.stages.${s}`)}
              </span>
            ))}
          </span>
        </button>
        {stage.error && (
          <p role="alert" className="pgn-error-text">
            {t('common.saveError')}
          </p>
        )}
        {visit.resultNote && <p className="pgn-body-text">{visit.resultNote}</p>}

        {(visit.appointmentId != null || visit.place) && (
          <div className="pgn-visit-actions">
            {visit.appointmentId != null && (
              <Link href={`/reminders/appointment/${visit.appointmentId}`} className="pgn-apill nb-tone-data">
                <Icon name="bell" size={15} />
                {remind ? t('calendar.reminder', { when: remind }) : t('calendar.stages.booked')}
              </Link>
            )}
            {visit.place && (
              <a
                href={directionsHref(visit.place)}
                target="_blank"
                rel="noopener noreferrer"
                className="pgn-apill nb-tone-brand"
              >
                <Icon name="mapPin" size={15} />
                {t('calendar.directions')}
              </a>
            )}
          </div>
        )}
      </Card>

      <AppSheet
        open={noteOpen}
        onClose={() => setNoteOpen(false)}
        size="half"
        title={t('calendar.stages.result')}
        footer={
          <PrimaryButton loading={stage.busy} onClick={() => void saveNote()}>
            {t('calendar.saveResult')}
          </PrimaryButton>
        }
      >
        <label className="pgn-field-label">
          {t('calendar.resultNote')}
          <textarea
            value={note}
            maxLength={2000}
            rows={4}
            onChange={(e) => setNote(e.target.value)}
            className="pgn-textarea"
          />
        </label>
        {stage.error && (
          <p role="alert" className="pgn-error-text">
            {t('common.saveError')}
          </p>
        )}
      </AppSheet>
    </>
  );
}

/** Day + month within a year of today (the care plan fits in one pregnancy), else the full date (audit E5). */
function carePlanDate(iso: string, locale: Locale): string {
  const d = fromApiDate(iso);
  return Math.abs(diffInDays(d, today())) <= 330 ? formatDayMonth(d, locale) : formatLongDate(d, locale);
}

/** The next visit's day when it falls in the shown month, else today. */
function defaultSelection(data: PregnancyCalendar | null, todayKey: string): string {
  const next = data?.nextVisit?.date;
  return next && data?.days.some((d) => d.date === next) ? next : todayKey;
}

function CareRow({ item }: { item: CareItem }) {
  const t = useTranslations('pregnancyV2.calendar');
  const tCommon = useTranslations('pregnancyV2.common');
  const locale = useLocale() as Locale;
  const router = useRouter();
  // «رزرو»: the item rides a one-time prefill, not the URL (audit M3-M7 #3).
  const book = (event: MouseEvent<HTMLAnchorElement>) => {
    event.preventDefault();
    router.push(withHandoff(BOOK_HREF, bookPrefill(item)));
  };
  const window =
    item.weekFrom != null && item.weekTo != null ? t('weekWindow', { from: item.weekFrom, to: item.weekTo }) : null;
  // Booked/done rows show their visit date; unbooked ones «حدود …» the suggested date (audit E1).
  const iso = item.date ?? item.suggestedDate;
  const when = iso ? carePlanDate(iso, locale) : null;
  const dateText = when ? (item.state === 'to_book' ? t('aroundDate', { date: when }) : when) : null;
  const sub = [window, dateText]
    .filter(Boolean)
    .join(tCommon('separator'));

  const tone: Tone = item.state === 'done' ? 'data' : item.state === 'booked' ? 'brand' : 'neutral';
  return (
    <li className="pgn-row">
      <IconCircle icon={item.state === 'done' ? 'check' : 'calendar'} tone={tone} size="sm" />
      <span className="pgn-row-text">
        <b className="pgn-row-title">{item.title}</b>
        {sub && <span className="pgn-row-desc">{sub}</span>}
      </span>
      {item.state === 'to_book' ? (
        <Link href={BOOK_HREF} onClick={book} className="pgn-apill nb-tone-brand">
          {t('book')}
        </Link>
      ) : (
        <span className={clsx('pgn-opill', `nb-tone-${tone}`)}>{t(`states.${item.state}`)}</span>
      )}
    </li>
  );
}

function ReportButton() {
  const t = useTranslations('pregnancyV2');
  const locale = useLocale() as Locale;
  const dir = useDirection();
  const [state, setState] = useState<'idle' | 'working' | 'error'>('idle');

  const run = async () => {
    setState('working');
    try {
      const to = today();
      const from = addDays(to, -(REPORT_DAYS - 1));
      const [report, { renderPdf }] = await Promise.all([
        fetchPregnancyReport({ from: toApiDate(from), to: toApiDate(to) }),
        loadPdfGenerator(),
      ]);
      const long = (ymd: string) => formatLongDate(fromApiDate(ymd), locale);
      const blocks: PdfBlock[] = [
        { kind: 'title', text: t('calendar.pdf.title') },
        {
          kind: 'subtitle',
          text: t('calendar.pdf.subtitle', { from: long(report.range.from), to: long(report.range.to) }),
        },
      ];
      if (report.age) {
        blocks.push({ kind: 'text', text: t('calendar.pdf.age', { age: t('common.age', { weeks: report.age.weeks, days: report.age.days }) }) });
      }
      if (report.dueDate) blocks.push({ kind: 'text', text: t('calendar.pdf.due', { date: long(report.dueDate) }) });

      blocks.push({ kind: 'rule' }, { kind: 'heading', text: t('calendar.pdf.daysTitle') });
      const days = report.days.filter(
        (d) => d.mood != null || Object.keys(d.symptoms).length > 0 || d.waterGlasses != null || d.visitNote,
      );
      if (days.length === 0) blocks.push({ kind: 'muted', text: t('calendar.pdf.none') });
      for (const d of days) {
        const symptoms = Object.entries(d.symptoms)
          .map(([k, sev]) => `${t(`log.symptoms.${k}` as 'log.symptoms.nausea')} (${t(`log.severities.${sev}` as 'log.severities.mild')})`)
          .join('، ');
        const line = [
          d.mood != null ? `${t('log.moodLabel')}: ${t(`log.moods.${d.mood}` as 'log.moods.5')}` : null,
          symptoms || null,
          d.waterGlasses != null ? t('calendar.pdf.water', { count: d.waterGlasses }) : null,
        ]
          .filter(Boolean)
          // fa «،», not «·»: a middle dot beside «۴ لیوان» reads as «۴۰».
          .join(t('common.separator'));
        blocks.push({ kind: 'text', text: `${long(d.date)}${line ? ' — ' + line : ''}` });
        if (d.visitNote) blocks.push({ kind: 'muted', text: t('calendar.pdf.note', { note: d.visitNote }) });
      }

      blocks.push({ kind: 'rule' }, { kind: 'heading', text: t('calendar.pdf.weightsTitle') });
      if (report.weights.length === 0) blocks.push({ kind: 'muted', text: t('calendar.pdf.none') });
      for (const w of report.weights) {
        blocks.push({
          kind: 'text',
          text: `${long(w.date)} — ${t('calendar.pdf.kg', { value: w.value })}`,
        });
      }
      blocks.push({ kind: 'rule' }, { kind: 'muted', text: t('calendar.pdf.disclaimer') });

      const blob = await renderPdf({ blocks, dir, locale, footer: t.raw('calendar.pdf.footer') as string });
      await shareOrDownloadFile(blob, t('calendar.pdf.filename'));
      setState('idle');
    } catch {
      // Health data: report nothing but the failure itself (§11).
      setState('error');
    }
  };

  return (
    <>
      <SecondaryButton icon={state === 'working' ? 'loader' : 'download'} loading={state === 'working'} onClick={() => void run()}>
        {state === 'working' ? t('calendar.pdf.working') : t('calendar.report')}
      </SecondaryButton>
      {state === 'error' && (
        <p role="alert" className="pgn-error-text is-center">
          {t('calendar.pdf.error')}
        </p>
      )}
    </>
  );
}
