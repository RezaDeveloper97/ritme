'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import {
  type CheckupDetail,
  type CheckupRecord,
  checkupIcon,
  checkupResultIcon,
  checkupStatusIcon,
  formatCheckupMonth,
  useCheckup,
} from '@/entities/checkup';
import { useUpdateCheckupSettings } from '@/features/manage-custom-checkup';
import { getApiErrorStatus } from '@/shared/api';
import { type Locale, Link, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { withHandoff } from '@/shared/lib/handoff';
import { openSheet } from '@/shared/sheet';
import { Icon, ScreenHeader, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';

import { MARK_DONE_SHEET, bookPrefill, heroRelative, markDoneSheetArg } from '../model/view';

const LIST_HREF = '/checkups';
const GUIDE_HREF = '/checkups/self-exam';
/** M3 AddAppointment: `?kind=` in the URL, the title through the `?prefill=` handoff. */
const BOOK_HREF = '/reminders/appointment/new?kind=in_person';

function Header({ detail }: { detail?: CheckupDetail }) {
  const t = useTranslations('checkups');
  const router = useRouter();
  const update = useUpdateCheckupSettings();
  const remind = detail?.settings.remind ?? false;

  return (
    <ScreenHeader
      title={detail?.title ?? t('title')}
      subtitle={detail?.intervalLabel || undefined}
      onBack={() => router.push(LIST_HREF)}
      backLabel={t('back')}
      action={
        detail ? (
          <button
            type="button"
            role="switch"
            aria-checked={remind}
            className={clsx('nb-hbtn', remind && 'is-on')}
            aria-label={remind ? t('detail.remindOn') : t('detail.remindOff')}
            disabled={update.isPending || !detail.settings.enabled}
            onClick={() => update.mutate({ id: detail.id, remind: !remind })}
          >
            <Icon name={remind ? 'bellRing' : 'bellPlain'} size={20} strokeWidth={1.8} />
          </button>
        ) : undefined
      }
    />
  );
}

function Hero({ detail }: { detail: CheckupDetail }) {
  const t = useTranslations('checkups');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const hasGuide = detail.guideSteps.length > 0;
  const next = detail.nextDueOn ? formatCheckupMonth(detail.nextDueOn, locale) : t('noDate');
  const rel = heroRelative(detail);
  // «حدود ۶ ماه از موعد گذشته، آخرین بار فروردین ۱۴۰۱» — months past the *due date*, and only when overdue.
  const relative =
    rel.kind === 'never'
      ? t('detail.neverDone')
      : [
          rel.overdueMonths ? t('detail.overdueMonths', { count: rel.overdueMonths }) : null,
          t('detail.lastTime', { date: formatCheckupMonth(rel.lastDoneOn, locale) }),
        ]
          .filter(Boolean)
          .join(t('separator'));

  return (
    <section className={clsx('ck-hero', `ck-tone-${detail.tone}`)}>
      <div className="flex items-start justify-between gap-3">
        <span className="ck-hero-chip">
          <Icon name={checkupIcon(detail.icon, { performedBy: detail.performedBy })} size={13} />
          {detail.subtitle ?? t(`section.${detail.category}`)}
        </span>
        <span className={clsx(`ck-status-${detail.status}`, 'ck-pill is-solid')}>
          <Icon name={checkupStatusIcon(detail.status)} size={12} strokeWidth={2.2} />
          {t(`status.${detail.status}`)}
        </span>
      </div>
      <div className="flex items-center gap-3">
        <div className="min-w-0 flex-1">
          <p className="text-[12px] font-semibold text-(--ink-3)">{t('detail.nextDue')}</p>
          <p className="font-['Lalezar'] text-[30px] leading-tight font-normal text-(--ink)">{next}</p>
          <p className="mt-1 text-[12px] font-semibold text-(--ink-2)">{relative}</p>
        </div>
        <span className="ck-hero-disc" aria-hidden>
          <Icon name={checkupIcon(detail.icon, { performedBy: detail.performedBy })} size={26} strokeWidth={1.8} />
        </span>
      </div>
      <div className="flex gap-2">
        {hasGuide ? (
          <Link href={GUIDE_HREF} className="btn flex-1 bg-(--surface) text-(--brand-strong)">
            <Icon name="bookOpen" size={16} />
            {t('detail.openGuide')}
          </Link>
        ) : (
          <button
            type="button"
            className="btn flex-1 bg-(--surface) text-(--brand-strong)"
            disabled={detail.status === 'disabled'}
            onClick={() => openSheet(MARK_DONE_SHEET, markDoneSheetArg(detail.id))}
          >
            <Icon name="check" size={16} strokeWidth={2.2} />
            {t('detail.markDone')}
          </button>
        )}
        {detail.performedBy !== 'self' && (
          <button
            type="button"
            className="btn ck-hero-alt flex-1"
            onClick={() => router.push(withHandoff(BOOK_HREF, bookPrefill(detail)))}
          >
            <Icon name="calendar" size={16} />
            {t('detail.book')}
          </button>
        )}
      </div>
    </section>
  );
}

function RecordRow({ detail, record }: { detail: CheckupDetail; record: CheckupRecord }) {
  const t = useTranslations('checkups');
  const locale = useLocale() as Locale;
  return (
    <li className={`ck-result-${record.result}`}>
      <button
        type="button"
        className="flex w-full items-center gap-3 py-2.5 text-start"
        aria-label={t('detail.editRecord')}
        onClick={() => openSheet(MARK_DONE_SHEET, markDoneSheetArg(detail.id, record.id))}
      >
        <span className="size-2.5 shrink-0 rounded-full bg-(--ck-ink)" aria-hidden />
        <span className="flex min-w-0 flex-1 flex-col gap-0.5">
          <span className="text-[13.5px] font-extrabold text-(--ink)">{formatCheckupMonth(record.doneOn, locale)}</span>
          <span className="text-[11.5px] text-(--ink-3)">
            {record.hasAttachment ? t('detail.attached') : t('detail.noAttachment')}
          </span>
        </span>
        <span className="ck-pill">
          <Icon name={checkupResultIcon(record.result)} size={12} strokeWidth={2.2} />
          {t(`result.${record.result}`)}
        </span>
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
      <SkeletonGroup label={t('loading')} className="rmd-form-skel">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
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
                <li key={i} className="flex items-start gap-2.5 text-[13px] leading-relaxed font-semibold text-(--ink)">
                  <span className="grid size-6 shrink-0 place-items-center rounded-full bg-(--pink-bg) text-[11px] font-extrabold text-(--brand)">
                    {formatNumber(i + 1, locale)}
                  </span>
                  {step}
                </li>
              ))}
            </ol>
            {hasCycleHint && (
              <p className="text-[12px] text-(--ink-3)">
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
          <Link href={`/checkups/custom/${detail.id}`} className="nb-btn is-outline is-block">
            <Icon name="pencil" size={16} />
            {t('custom.editTitle')}
          </Link>
        )}

        <p className="text-center text-[11.5px] leading-relaxed text-(--ink-3)">{t('detail.disclaimer')}</p>
      </>
    );
  }

  return (
    <div className="view rmd-page">
      <div className="scroll rmd-screen">
        <SkyLayer />
        <Header detail={detail} />
        <div className="rmd-body flex flex-col gap-3 pb-8">{body}</div>
      </div>
    </div>
  );
}
