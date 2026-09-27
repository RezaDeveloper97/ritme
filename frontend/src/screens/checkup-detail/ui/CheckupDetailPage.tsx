'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import { type CheckupDetail, type CheckupRecord, useCheckup } from '@/entities/checkup';
import { useUpdateCheckupSettings } from '@/features/manage-custom-checkup';
import { getApiErrorStatus } from '@/shared/api';
import { type Locale, Link, useDirection } from '@/shared/i18n';
import { formatLongDate, formatNumber, fromApiDate } from '@/shared/lib/date';
import { openSheet } from '@/shared/sheet';
import { Icon } from '@/shared/ui';

import { MARK_DONE_SHEET, markDoneSheetArg, monthsSince } from '../model/view';

const LIST_HREF = '/checkups';
const GUIDE_HREF = '/checkups/self-exam';
/** M3 AddAppointment; it reads `?kind=` only (see open items of T-M4-08). */
const BOOK_HREF = '/reminders/appointment/new?kind=in_person';

function Header({ detail }: { detail?: CheckupDetail }) {
  const t = useTranslations('checkups');
  const dir = useDirection();
  const update = useUpdateCheckupSettings();
  const remind = detail?.settings.remind ?? false;

  return (
    <header className="rmd-hdr">
      <Link href={LIST_HREF} className="rmd-hdr-btn" aria-label={t('back')}>
        <Icon name={dir === 'rtl' ? 'chevronRight' : 'chevronLeft'} size={20} strokeWidth={1.8} />
      </Link>
      <div className="rmd-hdr-text">
        <h1 className="rmd-hdr-title">{detail?.title ?? t('title')}</h1>
        {detail?.intervalLabel && <p className="rmd-hdr-sub">{detail.intervalLabel}</p>}
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
  );
}

function Hero({ detail }: { detail: CheckupDetail }) {
  const t = useTranslations('checkups');
  const locale = useLocale() as Locale;
  const hasGuide = detail.guideSteps.length > 0;
  const months = monthsSince(detail.lastDoneOn);
  const next = detail.nextDueOn ? formatLongDate(fromApiDate(detail.nextDueOn), locale) : t('noDate');
  const relative = detail.lastDoneOn
    ? [
        months !== null && months > 0 ? t('detail.passedMonths', { count: months }) : null,
        t('detail.lastTime', { date: formatLongDate(fromApiDate(detail.lastDoneOn), locale) }),
      ]
        .filter(Boolean)
        .join(t('separator'))
    : t('detail.neverDone');

  return (
    <section className="flex flex-col gap-3 rounded-3xl bg-(image:--gradient-brand) p-4 text-(--on-accent)">
      <div className="flex items-start justify-between gap-3">
        <div className="flex flex-wrap gap-1.5">
          <span className="rounded-full bg-white/20 px-2.5 py-1 text-[11px] font-extrabold">
            {t(`section.${detail.category}`)}
          </span>
          <span className="rounded-full bg-white/90 px-2.5 py-1 text-[11px] font-extrabold text-(--brand)">
            {t(`status.${detail.status}`)}
          </span>
        </div>
        <span className="grid size-12 shrink-0 place-items-center rounded-full bg-white/20" aria-hidden>
          <Icon name="shield" size={22} />
        </span>
      </div>
      <div className="text-start">
        <p className="text-[12px] font-semibold opacity-85">{t('detail.nextDue')}</p>
        <p className="font-['Lalezar'] text-[28px] leading-tight font-normal">{next}</p>
        <p className="mt-1 text-[12px] opacity-85">{relative}</p>
      </div>
      <div className="flex gap-2">
        {hasGuide ? (
          <Link href={GUIDE_HREF} className="btn flex-1 bg-white text-(--brand)">
            <Icon name="bookOpen" size={16} />
            {t('detail.openGuide')}
          </Link>
        ) : (
          <button
            type="button"
            className="btn flex-1 bg-white text-(--brand)"
            disabled={detail.status === 'disabled'}
            onClick={() => openSheet(MARK_DONE_SHEET, markDoneSheetArg(detail.id))}
          >
            <Icon name="check" size={16} strokeWidth={2.2} />
            {t('detail.markDone')}
          </button>
        )}
        {detail.performedBy !== 'self' && (
          <Link href={BOOK_HREF} className="btn flex-1 bg-white/20 text-(--on-accent)">
            <Icon name="calendar" size={16} />
            {t('detail.book')}
          </Link>
        )}
      </div>
    </section>
  );
}

function RecordRow({ detail, record }: { detail: CheckupDetail; record: CheckupRecord }) {
  const t = useTranslations('checkups');
  const locale = useLocale() as Locale;
  return (
    <li>
      <button
        type="button"
        className="flex w-full items-center gap-3 py-2.5 text-start"
        aria-label={t('detail.editRecord')}
        onClick={() => openSheet(MARK_DONE_SHEET, markDoneSheetArg(detail.id, record.id))}
      >
        <span className="flex min-w-0 flex-1 flex-col gap-0.5">
          <span className="text-[13px] font-extrabold text-(--ink)">
            {formatLongDate(fromApiDate(record.doneOn), locale)}
            {t('separator')}
            {t(`result.${record.result}`)}
          </span>
          <span className="flex items-center gap-1 text-[11.5px] text-(--ink-3)">
            <Icon name={record.hasAttachment ? 'note' : 'x'} size={12} />
            {record.hasAttachment ? t('detail.attached') : t('detail.noAttachment')}
          </span>
        </span>
        <Icon name="pencil" size={14} className="shrink-0 text-(--ink-3)" />
      </button>
    </li>
  );
}

/**
 * Checkup detail (`v14_CheckupDetail`) — `/checkups/[id]`. Health data (§11):
 * display only, never logged.
 */
export function CheckupDetailPage({ id }: { id: number }) {
  const t = useTranslations('checkups');
  const locale = useLocale() as Locale;
  const query = useCheckup(id);
  const detail = query.data;

  let body;
  if (query.isPending) {
    body = (
      <p className="rmd-state" role="status">
        {t('loading')}
      </p>
    );
  } else if (!detail) {
    const missing = getApiErrorStatus(query.error) === 404;
    body = (
      <div className="rmd-state" role="alert">
        <p>{missing ? t('custom.notFound') : t('loadError')}</p>
        {!missing && (
          <button type="button" className="rmd-retry" onClick={() => void query.refetch()}>
            {t('retry')}
          </button>
        )}
      </div>
    );
  } else {
    const hasCycleHint = detail.cycleDayFrom !== null && detail.cycleDayTo !== null;
    body = (
      <>
        <Hero detail={detail} />

        {detail.why && (
          <section className="card flex flex-col gap-2 p-4 text-start">
            <h2 className="text-[14px] font-extrabold text-(--ink)">{t('detail.why')}</h2>
            <p className="text-[13px] leading-relaxed text-(--ink-2)">{detail.why}</p>
          </section>
        )}

        {(detail.prepSteps.length > 0 || hasCycleHint) && (
          <section className="card flex flex-col gap-2.5 p-4 text-start">
            <h2 className="text-[14px] font-extrabold text-(--ink)">{t('detail.prep')}</h2>
            <ol className="flex flex-col gap-2">
              {detail.prepSteps.map((step, i) => (
                <li key={i} className="flex items-start gap-2.5 text-[13px] text-(--ink-2)">
                  <span className="grid size-6 shrink-0 place-items-center rounded-full bg-(--pink-bg) text-[11px] font-extrabold text-(--brand)">
                    {formatNumber(i + 1, locale)}
                  </span>
                  {step}
                </li>
              ))}
            </ol>
            {hasCycleHint && (
              <p className="flex items-start gap-2 rounded-2xl bg-(--surface-2) p-2.5 text-[12px] text-(--ink-3)">
                <Icon name="drop" size={14} className="mt-0.5 shrink-0 text-(--brand)" />
                {t('detail.cycleHint', {
                  from: formatNumber(detail.cycleDayFrom as number, locale),
                  to: formatNumber(detail.cycleDayTo as number, locale),
                })}
              </p>
            )}
          </section>
        )}

        <section className="card flex flex-col gap-1 p-4 text-start">
          <div className="flex items-center justify-between">
            <h2 className="text-[14px] font-extrabold text-(--ink)">{t('detail.history')}</h2>
            {detail.records.length > 0 && (
              <Link href={`/checkups/history?type=${detail.id}`} className="text-[12px] font-bold text-(--brand)">
                {t('detail.historyAll')}
              </Link>
            )}
          </div>
          {detail.records.length === 0 ? (
            <p className="py-2 text-[12.5px] text-(--ink-3)">{t('detail.noHistory')}</p>
          ) : (
            <ul className="flex flex-col divide-y divide-(--line)">
              {detail.records.slice(0, 2).map((r) => (
                <RecordRow key={r.id} detail={detail} record={r} />
              ))}
            </ul>
          )}
        </section>

        {detail.isCustom && (
          <Link href={`/checkups/custom/${detail.id}`} className="btn btn-ghost w-full">
            <Icon name="pencil" size={16} />
            {t('custom.editTitle')}
          </Link>
        )}

        <p className="flex items-start gap-2 rounded-2xl bg-(--surface-2) p-3 text-start text-[11.5px] leading-relaxed text-(--ink-3)">
          <Icon name="info" size={16} className="mt-0.5 shrink-0" />
          {t('detail.disclaimer')}
        </p>
      </>
    );
  }

  return (
    <div className="view rmd-page">
      <div className="scroll">
        <Header detail={detail} />
        <div className="rmd-body flex flex-col gap-3 pb-8">{body}</div>
      </div>
    </div>
  );
}
