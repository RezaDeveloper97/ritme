'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useCycleToday } from '@/entities/cycle';
import { type CheckupDetail, useCheckup, useCheckupRecords, useCheckups } from '@/entities/checkup';
import { useUpdateCheckupSettings } from '@/features/manage-custom-checkup';
import { selfExamResult, toggleFinding, useCreateCheckupRecord } from '@/features/record-checkup';
import { type Locale, Link, useDirection } from '@/shared/i18n';
import { formatNumber, toApiDate, today } from '@/shared/lib/date';
import { Icon } from '@/shared/ui';

import { SELF_EXAM_MINUTES, adherence, daysUntilWindow, doneThisMonth, findSelfExam } from '../model/view';

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
        <circle cx="38" cy="38" r={RING_R} fill="none" strokeWidth="6" className="stroke-(--track)" />
        <circle
          cx="38"
          cy="38"
          r={RING_R}
          fill="none"
          strokeWidth="6"
          className="stroke-(--brand)"
          strokeLinecap="round"
          strokeDasharray={RING_C}
          strokeDashoffset={RING_C * (1 - pct)}
        />
      </svg>
      <span className="flex flex-col items-center leading-none" aria-hidden>
        <span className="text-[10px] text-(--ink-3)">{t('selfExam.ringDay')}</span>
        <span className="font-['Lalezar'] text-[22px] text-(--ink)">{formatNumber(day, locale)}</span>
        <span className="text-[9.5px] text-(--ink-3)">{t('selfExam.ringOf', { total: formatNumber(total, locale) })}</span>
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

  // The window chip names the *window* («روز ۷ سیکل، ۳ روز دیگر»), not today's cycle day (audit E2).
  let when: string | null = null;
  if (day !== null && total && from !== null && to !== null) {
    const days = daysUntilWindow(day, total, from, to);
    when = days === 0 ? t('selfExam.whenNow') : t('selfExam.whenIn', { count: formatNumber(days, locale) });
  }

  return (
    <section className="ck-hero ck-tone-violet">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="ck-hero-chip">
          <Icon name="calendar" size={13} />
          {t('selfExam.thisMonth')}
        </span>
        {from !== null && when && (
          <span className="ck-hero-chip">{t('selfExam.dayWhen', { day: formatNumber(from, locale), when })}</span>
        )}
      </div>
      <div className="flex items-center gap-3">
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <span className="text-[12px] font-semibold text-(--ink-3)">{t('selfExam.bestTime')}</span>
          <h2 className="font-['Lalezar'] text-[26px] leading-tight font-normal text-(--ink)">
            {t('selfExam.bestTimeTitle')}
          </h2>
          <p className="text-[12.5px] leading-relaxed font-semibold text-(--ink-2)">{t('selfExam.bestTimeBody')}</p>
        </div>
        {day !== null && total ? <CycleRing day={day} total={total} /> : null}
      </div>
    </section>
  );
}

function Body({ detail }: { detail: CheckupDetail }) {
  const t = useTranslations('checkups');
  const locale = useLocale() as Locale;
  const exclusive = detail.findingOptions.filter((o) => o.exclusive).map((o) => o.key);
  // «چیزی متفاوت نبود» starts selected (v14_SelfExam).
  const [findings, setFindings] = useState<string[]>(() => exclusive.slice(0, 1));
  const create = useCreateCheckupRecord();
  const records = useCheckupRecords({ type: detail.id });
  const todayYmd = toApiDate(today());
  const loaded = records.data?.pages.flatMap((p) => p.records) ?? [];
  const done = doneThisMonth(loaded, todayYmd) || create.isSuccess;
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
        <section className="card flex flex-col gap-2.5 p-4 text-start">
          <h2 className="text-[14px] font-extrabold text-(--ink)">
            {t('selfExam.stepsTitle', {
              count: formatNumber(detail.guideSteps.length, locale),
              minutes: formatNumber(SELF_EXAM_MINUTES, locale),
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
        <section className="card flex flex-col gap-2.5 p-4 text-start">
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
                    'min-h-11 rounded-full border-[1.5px] px-4 text-[12.5px] font-bold',
                    on
                      ? 'border-(--brand) bg-(--pink-bg) text-(--ink)'
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
        <Link
          href={`/checkups/history?type=${detail.id}`}
          className="flex min-h-11 items-center justify-center gap-1.5 text-[12px] font-semibold text-(--ink-3)"
        >
          <Icon name="history" size={15} />
          {t('selfExam.adherence', {
            done: formatNumber(adherence(loaded, todayYmd) + (create.isSuccess && !doneThisMonth(loaded, todayYmd) ? 1 : 0), locale),
            total: formatNumber(12, locale),
          })}
        </Link>
      )}

      <p className="text-center text-[11.5px] leading-relaxed text-(--ink-3)">{t('detail.disclaimer')}</p>
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
  const update = useUpdateCheckupSettings();
  const detail = query.data;
  const remind = detail?.settings.remind ?? false;
  const subtitle = [item?.intervalLabel, item?.timingLabel].filter(Boolean).join(t('separator'));

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
            {subtitle && <p className="rmd-hdr-sub">{subtitle}</p>}
          </div>
          {detail ? (
            <button
              type="button"
              role="switch"
              aria-checked={remind}
              className={clsx('rmd-hdr-btn', remind && 'text-(--brand)')}
              aria-label={remind ? t('detail.remindOn') : t('detail.remindOff')}
              disabled={update.isPending || !detail.settings.enabled}
              onClick={() => update.mutate({ id: detail.id, remind: !remind })}
            >
              <Icon name={remind ? 'bellRing' : 'bellPlain'} size={20} strokeWidth={1.8} />
            </button>
          ) : (
            <span className="rmd-hdr-btn invisible" aria-hidden />
          )}
        </header>
        <div className="rmd-body flex flex-col gap-3 pb-8">{body}</div>
      </div>
    </div>
  );
}
