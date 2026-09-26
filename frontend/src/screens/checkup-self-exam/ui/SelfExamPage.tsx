'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useCycleToday } from '@/entities/cycle';
import { type CheckupDetail, useCheckup, useCheckupRecords, useCheckups } from '@/entities/checkup';
import { selfExamResult, toggleFinding, useCreateCheckupRecord } from '@/features/record-checkup';
import { type Locale, Link, useDirection } from '@/shared/i18n';
import { formatNumber, toApiDate, today } from '@/shared/lib/date';
import { Icon } from '@/shared/ui';

import { adherence, daysUntilWindow, doneThisMonth, findSelfExam } from '../model/view';

/** M3 AddAppointment (same target as the checkup detail «ثبت نوبت»). */
const BOOK_HREF = '/reminders/appointment/new?kind=in_person';
const RING_R = 30;
const RING_C = 2 * Math.PI * RING_R;

function CycleRing({ day, total }: { day: number; total: number }) {
  const t = useTranslations('checkups');
  const locale = useLocale() as Locale;
  const pct = Math.min(1, Math.max(0, day / total));
  return (
    <div
      className="relative grid size-[76px] shrink-0 place-items-center"
      role="img"
      aria-label={t('selfExam.ringLabel', { day: formatNumber(day, locale), total: formatNumber(total, locale) })}
    >
      <svg viewBox="0 0 76 76" className="absolute inset-0 -rotate-90" aria-hidden>
        <circle cx="38" cy="38" r={RING_R} fill="none" stroke="currentColor" strokeOpacity="0.25" strokeWidth="6" />
        <circle
          cx="38"
          cy="38"
          r={RING_R}
          fill="none"
          stroke="currentColor"
          strokeWidth="6"
          strokeLinecap="round"
          strokeDasharray={RING_C}
          strokeDashoffset={RING_C * (1 - pct)}
        />
      </svg>
      <span className="flex flex-col items-center leading-none" aria-hidden>
        <span className="text-[10px] opacity-85">{t('selfExam.ringDay')}</span>
        <span className="text-[20px] font-extrabold">{formatNumber(day, locale)}</span>
        <span className="text-[9.5px] opacity-85">{t('selfExam.ringOf', { total: formatNumber(total, locale) })}</span>
      </span>
    </div>
  );
}

function Hero({ detail }: { detail: CheckupDetail }) {
  const t = useTranslations('checkups');
  const locale = useLocale() as Locale;
  const { data } = useCycleToday();
  const calc = data?.calculation ?? null;
  const day = calc?.cycleDay ?? data?.cycleView?.cycleDay ?? null;
  const total = calc?.cycleLength ?? data?.cycleView?.metrics?.effectiveCycleLength ?? null;
  const from = detail.cycleDayFrom;
  const to = detail.cycleDayTo;

  let when: string | null = null;
  if (day !== null && total && from !== null && to !== null) {
    const days = daysUntilWindow(day, total, from, to);
    when = days === 0 ? t('selfExam.whenNow') : t('selfExam.whenIn', { count: days });
  }

  return (
    <section className="flex flex-col gap-3 rounded-3xl bg-(image:--gradient-brand) p-4 text-(--on-accent)">
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 flex-col items-start gap-2 text-start">
          <span className="rounded-full bg-white/90 px-2.5 py-1 text-[11px] font-extrabold text-(--brand)">
            {t('selfExam.thisMonth')}
          </span>
          <h2 className="font-['Lalezar'] text-[24px] leading-tight font-normal">{detail.title}</h2>
          {day !== null && when && (
            <p className="text-[12px] font-semibold opacity-90">
              {t('selfExam.dayWhen', { day: formatNumber(day, locale), when })}
            </p>
          )}
        </div>
        {day !== null && total ? <CycleRing day={day} total={total} /> : null}
      </div>
      <div className="flex items-start gap-2 rounded-2xl bg-white/15 p-3 text-start">
        <Icon name="clock" size={16} className="mt-0.5 shrink-0" />
        <div className="flex flex-col gap-0.5">
          <span className="text-[11px] opacity-85">{t('selfExam.bestTime')}</span>
          <span className="text-[13px] font-extrabold">{t('selfExam.bestTimeTitle')}</span>
          <span className="text-[11.5px] leading-relaxed opacity-90">{t('selfExam.bestTimeBody')}</span>
        </div>
      </div>
    </section>
  );
}

function Body({ detail }: { detail: CheckupDetail }) {
  const t = useTranslations('checkups');
  const locale = useLocale() as Locale;
  const [findings, setFindings] = useState<string[]>([]);
  const create = useCreateCheckupRecord();
  const records = useCheckupRecords({ type: detail.id });
  const todayYmd = toApiDate(today());
  const loaded = records.data?.pages.flatMap((p) => p.records) ?? [];
  const done = doneThisMonth(loaded, todayYmd) || create.isSuccess;
  const exclusive = detail.findingOptions.filter((o) => o.exclusive).map((o) => o.key);
  const hasFinding = findings.some((k) => !exclusive.includes(k));

  const submit = () =>
    create.mutate({
      typeId: detail.id,
      input: { doneOn: todayYmd, result: selfExamResult(findings, exclusive), findings },
    });

  return (
    <>
      <Hero detail={detail} />

      {detail.guideSteps.length > 0 && (
        <section className="card flex flex-col gap-2.5 text-start">
          <h2 className="text-[14px] font-extrabold text-(--ink)">
            {t('selfExam.stepsTitle', {
              count: formatNumber(detail.guideSteps.length, locale),
              minutes: formatNumber(Math.max(3, detail.guideSteps.length), locale),
            })}
          </h2>
          <ol className="flex flex-col gap-3">
            {detail.guideSteps.map((step, i) => (
              <li key={i} className="flex items-start gap-2.5">
                <span className="grid size-6 shrink-0 place-items-center rounded-full bg-(--pink-bg) text-[11px] font-extrabold text-(--brand)">
                  {formatNumber(i + 1, locale)}
                </span>
                <span className="flex flex-col gap-0.5">
                  <span className="text-[13px] font-extrabold text-(--ink)">{step.title}</span>
                  {step.body && <span className="text-[12px] leading-relaxed text-(--ink-2)">{step.body}</span>}
                </span>
              </li>
            ))}
          </ol>
        </section>
      )}

      {detail.findingOptions.length > 0 && (
        <section className="card flex flex-col gap-2.5 text-start">
          <h2 className="text-[14px] font-extrabold text-(--ink)">{t('selfExam.findingsTitle')}</h2>
          <p className="text-[12px] leading-relaxed text-(--ink-3)">{t('selfExam.findingsBody')}</p>
          <div className="flex flex-wrap gap-2" role="group" aria-label={t('selfExam.findingsTitle')}>
            {detail.findingOptions.map((o) => {
              const on = findings.includes(o.key);
              return (
                <button
                  key={o.key}
                  type="button"
                  aria-pressed={on}
                  disabled={done}
                  className={clsx(
                    'rounded-full border px-3 py-1.5 text-[12px] font-bold',
                    on
                      ? 'border-transparent bg-(--brand-fill) text-(--on-accent)'
                      : 'border-(--line) bg-(--surface) text-(--ink-2)',
                  )}
                  onClick={() => setFindings((s) => toggleFinding(s, o.key, exclusive))}
                >
                  {o.label}
                </button>
              );
            })}
          </div>
          {hasFinding && (
            <div className="flex items-center gap-2 rounded-2xl bg-(--amber-soft) p-3" role="status">
              <Icon name="stetho" size={16} className="shrink-0 text-(--amber-deep)" />
              <span className="flex-1 text-[12px] font-bold text-(--ink)">{t('selfExam.nudge')}</span>
              <Link href={BOOK_HREF} className="text-[12px] font-extrabold text-(--brand)">
                {t('selfExam.book')}
              </Link>
            </div>
          )}
        </section>
      )}

      <button type="button" className="btn btn-primary w-full" disabled={done || create.isPending} onClick={submit}>
        <Icon name="check" size={16} strokeWidth={2.2} />
        {done ? t('selfExam.doneAlready') : t('selfExam.done')}
      </button>
      {create.isError && (
        <p className="text-center text-[12px] text-(--danger)" role="alert">
          {t('saveError')}
        </p>
      )}
      {records.data && (
        <p className="text-center text-[12px] text-(--ink-3)">
          {t('selfExam.adherence', {
            done: formatNumber(adherence(loaded, todayYmd) + (create.isSuccess && !doneThisMonth(loaded, todayYmd) ? 1 : 0), locale),
            total: formatNumber(12, locale),
          })}
        </p>
      )}

      <p className="flex items-start gap-2 rounded-2xl bg-(--surface-2) p-3 text-start text-[11.5px] leading-relaxed text-(--ink-3)">
        <Icon name="info" size={16} className="mt-0.5 shrink-0" />
        {t('detail.disclaimer')}
      </p>
    </>
  );
}

/**
 * Breast self-exam guide (`v14_SelfExam`) — `/checkups/self-exam`. Health data
 * (§11): findings are sent only as the record's keys, never logged.
 */
export function SelfExamPage() {
  const t = useTranslations('checkups');
  const dir = useDirection();
  const list = useCheckups();
  const item = findSelfExam(list.data?.items);
  const query = useCheckup(item?.id ?? null);

  let body;
  if (list.isPending || (item && query.isPending)) {
    body = (
      <p className="rmd-state" role="status">
        {t('loading')}
      </p>
    );
  } else if (!item || !query.data) {
    body = (
      <div className="rmd-state" role="alert">
        <p>{item || list.isError ? t('loadError') : t('custom.notFound')}</p>
        {(list.isError || query.isError) && (
          <button
            type="button"
            className="rmd-retry"
            onClick={() => void (list.isError ? list.refetch() : query.refetch())}
          >
            {t('retry')}
          </button>
        )}
      </div>
    );
  } else {
    body = <Body detail={query.data} />;
  }

  return (
    <div className="view rmd-page">
      <div className="scroll">
        <header className="rmd-hdr">
          <Link href={item ? `/checkups/${item.id}` : '/checkups'} className="rmd-hdr-btn" aria-label={t('back')}>
            <Icon name={dir === 'rtl' ? 'chevronRight' : 'chevronLeft'} size={20} strokeWidth={1.8} />
          </Link>
          <div className="rmd-hdr-text">
            <h1 className="rmd-hdr-title">{item?.title ?? t('title')}</h1>
          </div>
          <Link href="/checkups/history" className="rmd-hdr-btn" aria-label={t('history.title')}>
            <Icon name="history" size={20} strokeWidth={1.8} />
          </Link>
        </header>
        <div className="rmd-body flex flex-col gap-3 pb-8">{body}</div>
      </div>
    </div>
  );
}
