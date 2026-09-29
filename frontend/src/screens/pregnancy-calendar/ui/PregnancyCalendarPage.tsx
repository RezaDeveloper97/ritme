'use client';

import { useQueryClient } from '@tanstack/react-query';
import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useMemo, useState } from 'react';

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
import { type Locale, Link, useDirection } from '@/shared/i18n';
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
  monthName,
  shiftMonth,
  toApiDate,
  today,
  todayParts,
  toParts,
  weekdayLabels,
} from '@/shared/lib/date';
import { type PdfBlock, loadPdfGenerator, shareOrDownloadFile } from '@/shared/lib/pdf';
import { AppSheet } from '@/shared/sheet';
import { Icon } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

import { REPORT_DAYS, bookHref, directionsHref, monthKey, nextStage, stageProgress } from '../model/view';

const CARD = 'rounded-2xl border border-(--line) bg-(--surface)';
const NEW_VISIT_HREF = '/reminders/appointment/new?kind=in_person';

/** «تقویم و ویزیت‌ها» — `/pregnancy/calendar` (Calendar.dc.html, T-M7-13). */
export function PregnancyCalendarPage() {
  const t = useTranslations('pregnancyV2');
  const locale = useLocale() as Locale;
  const dir = useDirection();
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

  return (
    <div className="view preg-page">
      <div className="scroll">
        <header className="flex items-center justify-between px-4 pt-4.5 pb-2">
          <h1 className="m-0 text-2xl font-black text-(--ink)">{t('calendar.title')}</h1>
          <Link
            href={NEW_VISIT_HREF}
            aria-label={t('calendar.newVisitLabel')}
            className="inline-flex h-11 items-center gap-1.5 rounded-full bg-(--brand-fill) px-3.5 text-[13px] font-extrabold text-(--on-accent) no-underline"
          >
            <Icon name="plus" size={16} strokeWidth={2.2} />
            {t('calendar.newVisit')}
          </Link>
        </header>

        <div className="flex flex-col gap-3.5 px-4 pt-1 pb-28">
          <section className={clsx(CARD, 'p-3.5')}>
            <div className="mb-2.5 flex items-center justify-between">
              <button
                type="button"
                className="flex size-11 items-center justify-center rounded-full border border-(--line) bg-(--surface) text-(--ink)"
                aria-label={t('calendar.prevMonth')}
                onClick={() => move(-1)}
              >
                <Icon name={dir === 'rtl' ? 'chevronRight' : 'chevronLeft'} size={18} />
              </button>
              <div className="text-center">
                <b className="text-base text-(--ink)">
                  {data?.monthLabel ?? formatMonthLabel(ym.year, ym.month, locale)}
                </b>
                {data?.weekRange && (
                  <div className="text-[11.5px] font-bold text-(--ink-2)">
                    {t('calendar.weekRange', { from: data.weekRange.from, to: data.weekRange.to })}
                  </div>
                )}
              </div>
              <button
                type="button"
                className="flex size-11 items-center justify-center rounded-full border border-(--line) bg-(--surface) text-(--ink)"
                aria-label={t('calendar.nextMonth')}
                onClick={() => move(1)}
              >
                <Icon name={dir === 'rtl' ? 'chevronLeft' : 'chevronRight'} size={18} />
              </button>
            </div>
            <MonthGrid year={ym.year} month={ym.month} data={data ?? null} selected={selected} onPick={setPicked} />
            <Legend />
          </section>

          {query.isPending ? (
            <p className="m-0 py-10 text-center text-sm font-semibold text-(--ink-3)">{t('common.loading')}</p>
          ) : query.isError ? (
            <div className="flex flex-col items-center gap-3 py-10 text-center">
              <p role="alert" className="m-0 text-sm font-semibold text-(--ink-3)">
                {t('common.loadError')}
              </p>
              <button type="button" className="btn btn-primary" onClick={() => void query.refetch()}>
                {t('common.retry')}
              </button>
            </div>
          ) : data ? (
            <CalendarBody data={data} selected={selected} />
          ) : (
            <div className="flex flex-col items-center gap-3 py-10 text-center">
              <p className="m-0 text-sm font-semibold text-(--ink-3)">{t('common.notActive')}</p>
              <Link href="/pregnancy" className="btn btn-primary no-underline">
                {t('common.back')}
              </Link>
            </div>
          )}
        </div>
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
      <div className="mb-1 grid grid-cols-7 gap-1 text-center text-[11px] font-extrabold text-(--ink-2)" aria-hidden>
        {weekdayLabels(locale).map((w) => (
          <span key={w}>{w}</span>
        ))}
      </div>
      <div className="grid grid-cols-7 gap-1">
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
              onClick={() => onPick(key)}
              className={clsx(
                'flex h-11.5 flex-col items-center justify-center gap-0.75 rounded-xl text-[13.5px]',
                isToday
                  ? 'bg-(--brand-fill) font-black text-(--on-accent)'
                  : isSel
                    ? 'border-[1.5px] border-(--brand) bg-(--pink-bg) font-black text-(--ink)'
                    : 'font-semibold text-(--ink)',
                !isToday && !isSel && (m?.weekStart != null ? 'border border-dashed border-(--brand-line)' : 'border border-transparent'),
              )}
            >
              <span>{formatNumber(cell.day, locale)}</span>
              <span
                aria-hidden
                className={clsx(
                  'size-1.25 rounded-full',
                  m?.hasVisit ? (isToday ? 'bg-(--on-accent)' : 'bg-(--data)') : 'bg-transparent',
                )}
              />
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
    <div className="mt-2.5 flex flex-wrap gap-3.5 text-[10.5px] font-bold text-(--ink-2)">
      <span className="inline-flex items-center gap-1.25">
        <span className="size-2.5 rounded-full bg-(--brand-fill)" aria-hidden />
        {t('today')}
      </span>
      <span className="inline-flex items-center gap-1.25">
        <span className="size-1.5 rounded-full bg-(--data)" aria-hidden />
        {t('visit')}
      </span>
      <span className="inline-flex items-center gap-1.25">
        <span className="size-2.5 rounded-[3px] border border-dashed border-(--brand-line)" aria-hidden />
        {t('weekStart')}
      </span>
    </div>
  );
}

function CalendarBody({ data, selected }: { data: PregnancyCalendar; selected: string }) {
  const t = useTranslations('pregnancyV2.calendar');
  return (
    <>
      <SelectedDay data={data} date={selected} />
      {data.nextVisit && <NextVisitCard visit={data.nextVisit} />}
      {data.carePlan.length > 0 && (
        <>
          <h2 className="m-0 mt-2 text-base font-black text-(--ink)">{t('carePlanTitle')}</h2>
          <section className={clsx(CARD, 'px-3.5 py-1.5')}>
            {data.carePlan.map((item, i) => (
              <CareRow key={item.key} item={item} last={i === data.carePlan.length - 1} />
            ))}
          </section>
        </>
      )}
      {data.sourceNote && (
        <p className="m-0 text-[11.5px] leading-[1.8] font-semibold text-(--ink-2)">{data.sourceNote}</p>
      )}
      <ReportButton />
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
    <section className={clsx(CARD, 'flex items-center gap-3 px-3.5 py-3')} aria-live="polite">
      <span className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-(--pink-bg) text-(--brand)" aria-hidden>
        <Icon name="calendar" size={18} />
      </span>
      <div className="min-w-0 flex-1">
        <b className="text-sm text-(--ink)">{title}</b>
        <div className="text-xs font-semibold text-(--ink-2)">{sub}</div>
      </div>
      {!visits.length && (
        <Link
          href={`${NEW_VISIT_HREF}&date=${date}`}
          aria-label={t('newVisitLabel')}
          className="flex size-11 shrink-0 items-center justify-center rounded-full text-(--brand)"
        >
          <Icon name="plus" size={18} strokeWidth={2.2} />
        </Link>
      )}
    </section>
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
  const parts = toParts(date, locale);
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

  return (
    <>
      {visit.daysUntil != null && (
        <span className="mt-1 text-xs font-black text-(--brand)">
          {t('calendar.nextVisit', { days: visit.daysUntil })}
        </span>
      )}
      <section className={clsx(CARD, 'flex flex-col gap-3.5 rounded-[20px] p-4 shadow-(--care-card-shadow)')}>
        <div className="flex items-center gap-3">
          <div className="flex h-15 w-14 shrink-0 flex-col items-center justify-center rounded-[14px] bg-(--pink-bg)">
            <span className="text-[11px] font-extrabold text-(--brand)">{monthName(parts.month, locale)}</span>
            <b className="text-[22px] leading-[1.1] font-black text-(--brand-deep)">{formatNumber(parts.day, locale)}</b>
          </div>
          <div className="min-w-0 flex-1">
            <b className="text-base text-(--ink)">{visit.title}</b>
            <div className="mt-0.5 text-[12.5px] font-semibold text-(--ink-2)">{meta}</div>
            {who && <div className="text-[12.5px] font-semibold text-(--ink-2)">{who}</div>}
          </div>
        </div>

        <button
          type="button"
          className="flex flex-col gap-1.5 border-0 bg-transparent p-0 text-start disabled:cursor-default"
          aria-label={t('calendar.stagesLabel')}
          disabled={!canAdvance}
          onClick={advance}
        >
          <span className="flex w-full gap-1">
            {VISIT_STAGES.map((s, i) => (
              <span
                key={s}
                aria-hidden
                className={clsx('h-1.25 flex-1 rounded-full', i < filled ? 'bg-(--brand-fill)' : 'bg-(--line)')}
              />
            ))}
          </span>
          <span className="flex w-full justify-between text-[11.5px] font-bold">
            {VISIT_STAGES.map((s, i) => (
              <span key={s} className={i < filled ? 'text-(--brand)' : 'text-(--ink-2)'}>
                {t(`calendar.stages.${s}`)}
              </span>
            ))}
          </span>
        </button>
        {stage.error && (
          <p role="alert" className="m-0 text-xs font-semibold text-(--danger)">
            {t('common.saveError')}
          </p>
        )}
        {visit.resultNote && (
          <p className="m-0 text-[12.5px] leading-[1.9] text-(--ink-2)">{visit.resultNote}</p>
        )}

        {visit.prep && (
          <div className="rounded-xl bg-(--pink-bg) px-3 py-2.5 text-[12.5px] leading-[1.9] text-(--ink-2)">
            <b className="text-(--ink)">{t('calendar.prep')}</b> {visit.prep}
          </div>
        )}

        <div className="grid grid-cols-2 gap-2">
          {visit.appointmentId != null ? (
            <Link
              href={`/reminders/appointment/${visit.appointmentId}`}
              className="flex h-11.5 items-center justify-center gap-1.5 rounded-[14px] border-[1.5px] border-(--line-2) bg-(--surface) text-[13px] font-extrabold text-(--ink) no-underline"
            >
              <Icon name="bell" size={16} />
              {remind ? t('calendar.reminder', { when: remind }) : t('calendar.stages.booked')}
            </Link>
          ) : (
            <span />
          )}
          {visit.place ? (
            <a
              href={directionsHref(visit.place)}
              target="_blank"
              rel="noopener noreferrer"
              className="flex h-11.5 items-center justify-center gap-1.5 rounded-[14px] border-[1.5px] border-(--line-2) bg-(--surface) text-[13px] font-extrabold text-(--ink) no-underline"
            >
              <Icon name="mapPin" size={16} />
              {t('calendar.directions')}
            </a>
          ) : (
            <span />
          )}
        </div>
      </section>

      <AppSheet
        open={noteOpen}
        onClose={() => setNoteOpen(false)}
        size="half"
        title={t('calendar.stages.result')}
        footer={
          <button type="button" className="btn btn-primary w-full" disabled={stage.busy} onClick={() => void saveNote()}>
            {t('calendar.saveResult')}
          </button>
        }
      >
        <label className="flex flex-col gap-2 text-sm font-bold text-(--ink)">
          {t('calendar.resultNote')}
          <textarea
            value={note}
            maxLength={2000}
            rows={4}
            onChange={(e) => setNote(e.target.value)}
            className="rounded-xl border border-(--field-border) bg-(--surface) p-3 text-sm font-medium text-(--ink)"
          />
        </label>
        {stage.error && (
          <p role="alert" className="m-0 mt-2 text-xs font-semibold text-(--danger)">
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

function CareRow({ item, last }: { item: CareItem; last: boolean }) {
  const t = useTranslations('pregnancyV2.calendar');
  const tCommon = useTranslations('pregnancyV2.common');
  const locale = useLocale() as Locale;
  const window =
    item.weekFrom != null && item.weekTo != null ? t('weekWindow', { from: item.weekFrom, to: item.weekTo }) : null;
  // Booked/done rows show their visit date; unbooked ones «حدود …» the suggested date (audit E1).
  const iso = item.date ?? item.suggestedDate;
  const when = iso ? carePlanDate(iso, locale) : null;
  const dateText = when ? (item.state === 'to_book' ? t('aroundDate', { date: when }) : when) : null;
  const sub = [window, dateText, item.state === 'done' ? t('states.done') : null]
    .filter(Boolean)
    .join(tCommon('separator'));

  return (
    <div className={clsx('flex min-h-16 items-center gap-3', !last && 'border-b border-(--line)')}>
      {item.state === 'done' ? (
        <span className="flex size-8.5 shrink-0 items-center justify-center rounded-full bg-(--success-soft) text-(--success)" aria-hidden>
          <Icon name="check" size={16} strokeWidth={2.4} />
        </span>
      ) : item.state === 'booked' ? (
        <span className="flex size-8.5 shrink-0 items-center justify-center rounded-full bg-(--data-soft)" aria-hidden>
          <span className="size-2.5 rounded-full bg-(--data)" />
        </span>
      ) : (
        <span className="size-8.5 shrink-0 rounded-full border-2 border-dashed border-(--line-2)" aria-hidden />
      )}
      <div className="min-w-0 flex-1">
        <b className="text-sm text-(--ink)">{item.title}</b>
        {sub && <div className="text-xs font-semibold text-(--ink-2)">{sub}</div>}
      </div>
      {item.state === 'booked' && (
        <span className="inline-flex h-6 items-center rounded-full bg-(--data-soft) px-2.25 text-[11px] font-extrabold text-(--data-deep)">
          {t('states.booked')}
        </span>
      )}
      {item.state === 'to_book' && (
        <Link href={bookHref(item)} className="inline-flex h-11 items-center text-[12.5px] font-extrabold text-(--brand)">
          {t('book')}
        </Link>
      )}
    </div>
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
      <button
        type="button"
        disabled={state === 'working'}
        onClick={() => void run()}
        className="flex h-13 items-center justify-center gap-2 rounded-[14px] border-[1.5px] border-(--line-2) bg-(--surface) text-sm font-extrabold text-(--ink)"
      >
        <Icon name={state === 'working' ? 'loader' : 'download'} size={18} />
        {state === 'working' ? t('calendar.pdf.working') : t('calendar.report')}
      </button>
      {state === 'error' && (
        <p role="alert" className="m-0 text-center text-xs font-semibold text-(--danger)">
          {t('calendar.pdf.error')}
        </p>
      )}
    </>
  );
}
