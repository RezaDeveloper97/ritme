'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  type ContraceptionMethod,
  hasMethodReminders,
  isPillMethod,
  METHOD_LOOK,
  methodToPayload,
  type PackCellState,
  packCellState,
  PACKS_LEFT_RANGE,
  packWeeks,
  type PillPack,
  useContraception,
  useLogPill,
  useSaveContraceptionMethod,
  useUndoPill,
} from '@/entities/contraception';
import { getApiSaveErrorMessage } from '@/shared/api';
import { type Locale, Link, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatNumber, fromApiDate } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import {
  Card,
  EmptyState,
  HeaderButton,
  Icon,
  IconCircle,
  ListGroup,
  ListRow,
  NumberStepper,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  SectionTitle,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

/**
 * «قرص پیشگیری» (`/contraception`, CB-CONTRA-02, nbl_Contra_Pill): today's
 * pill card («خوردم» / «یک قرص را جا انداختم» → `/contraception/missed`), the
 * 4 × 7 pack grid (taken / today / placebo / break), streak + next-pack tiles
 * and the refill row. A non-pill method shows its summary and a link to its
 * reminders; no method → empty state into setup. Back-header screen: no nav.
 */
export function ContraceptionPage() {
  const t = useTranslations('contraception');
  const router = useRouter();
  const query = useContraception();
  const data = query.data;
  const pill = data?.pill ?? null;
  const locale = useLocale() as Locale;

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="ctr-skel">
        <Skeleton shape="card" />
        <Skeleton width="short" />
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError || !data) {
    body = (
      <EmptyState
        icon="pill"
        title={t('error.title')}
        body={t('error.body')}
        action={
          <PrimaryButton icon="refresh" loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('error.retry')}
          </PrimaryButton>
        }
      />
    );
  } else if (!data.method) {
    body = (
      <EmptyState
        icon="shield"
        title={t('empty.title')}
        body={t('empty.body')}
        action={
          <PrimaryButton onClick={() => router.push('/contraception/setup')}>{t('empty.cta')}</PrimaryButton>
        }
      />
    );
  } else if (isPillMethod(data.method.method) && pill) {
    body = <PillBody method={data.method} pill={pill} />;
  } else {
    body = <MethodSummary method={data.method} />;
  }

  const isPill = Boolean(data?.method && isPillMethod(data.method.method));
  return (
    <div className="view ctr-page">
      <SkyLayer />
      <div className="scroll ctr-scroll">
        <ScreenHeader
          title={isPill || !data?.method ? t('pill.title') : t('setup.title')}
          subtitle={
            pill
              ? t('pill.subtitle', { pack: formatNumber(pill.packNumber, locale), day: formatNumber(pill.packDay, locale) })
              : undefined
          }
          onBack={() => router.push('/profile/mode')}
          backLabel={t('pill.back')}
          action={
            data?.method ? (
              <HeaderButton icon="cog" label={t('pill.settings')} onClick={() => router.push('/contraception/setup')} />
            ) : undefined
          }
        />
        {body}
      </div>
    </div>
  );
}

function PillBody({ method, pill }: { method: ContraceptionMethod; pill: PillPack }) {
  const [packsOpen, setPacksOpen] = useState(false);
  return (
    <>
      <TodayCard method={method} pill={pill} />
      <PackGrid pill={pill} />
      <StatTiles pill={pill} />
      <RefillRow pill={pill} onEdit={() => setPacksOpen(true)} />
      {packsOpen ? <PacksSheet method={method} onClose={() => setPacksOpen(false)} /> : null}
    </>
  );
}

function TodayCard({ method, pill }: { method: ContraceptionMethod; pill: PillPack }) {
  const t = useTranslations('contraception.pill');
  const locale = useLocale() as Locale;
  const log = useLogPill();
  const undo = useUndoPill();
  const today = pill.today;
  const n = (v: number | string) => formatNumber(v, locale);
  const busy = log.isPending || undo.isPending;
  const failed = log.isError || undo.isError;

  if (today.kind === 'break') {
    return (
      <Card className="ctr-today">
        <div className="ctr-today-head">
          <IconCircle icon="moon" tone="neutral" size="lg" />
          <div className="ctr-today-text">
            <p className="ctr-today-cap">{t('today')}</p>
            <p className="ctr-today-title is-small">{t('breakTitle')}</p>
          </div>
        </div>
        <p className="ctr-today-note">{t('breakBody')}</p>
      </Card>
    );
  }

  const time = method.reminder?.enabled ? method.reminder.time : null;
  const taken = today.status === 'taken';
  return (
    <Card className="ctr-today">
      <div className="ctr-today-head">
        <IconCircle icon="pill" tone="brand" size="lg" />
        <div className="ctr-today-text">
          <p className="ctr-today-cap">
            {time ? t('todayAt', { time: n(time) }) : t('today')}
          </p>
          <p className="ctr-today-title">
            {today.kind === 'placebo'
              ? t('placeboTitle', { day: n(pill.packDay) })
              : t('dayOf', { day: n(pill.packDay), total: n(pill.activeDays) })}
          </p>
        </div>
      </div>
      {taken ? (
        <div className="ctr-taken">
          <span className="ctr-taken-badge" role="status">
            <Icon name="checkCircle" size={20} />
            {t('takenDone')}
          </span>
          <SecondaryButton variant="text" block={false} loading={undo.isPending} onClick={() => undo.mutate(today.date)}>
            {t('undo')}
          </SecondaryButton>
        </div>
      ) : (
        <PrimaryButton icon="check" loading={log.isPending} disabled={busy} onClick={() => log.mutate({})}>
          {t('taken')}
        </PrimaryButton>
      )}
      {failed ? (
        <p className="ctr-error is-inline" role="alert">
          {getApiSaveErrorMessage(log.error ?? undo.error, t('logError'))}
        </p>
      ) : null}
      {today.kind === 'active' ? (
        <Link href="/contraception/missed" className="ctr-missed">
          {t('missed')}
        </Link>
      ) : null}
    </Card>
  );
}

const CELL_ICON: Partial<Record<PackCellState, 'check' | 'x'>> = {
  taken: 'check',
  todayTaken: 'check',
  missed: 'x',
};

function PackGrid({ pill }: { pill: PillPack }) {
  const t = useTranslations('contraception.pill');
  const locale = useLocale() as Locale;
  const weeks = packWeeks(pill.days);
  const hasBreak = pill.days.some((d) => d.kind !== 'active');
  return (
    <section className="ctr-pack" aria-labelledby="ctr-pack-title">
      <SectionTitle id="ctr-pack-title" title={t('packTitle')} />
      <Card className="ctr-pack-card">
        <ol className="ctr-grid" aria-label={t('packLabel')}>
          {weeks.flat().map((day) => {
            const state = packCellState(day, pill.today.date);
            const icon = CELL_ICON[state];
            return (
              <li
                key={day.day}
                className={clsx('ctr-cell', `is-${state}`)}
                aria-label={t('dayLabel', { day: formatNumber(day.day, locale), state: t(`states.${state}`) })}
                aria-current={state === 'today' || state === 'todayTaken' ? 'date' : undefined}
              >
                {icon ? <Icon name={icon} size={14} strokeWidth={2.6} /> : null}
              </li>
            );
          })}
        </ol>
        <p className="ctr-legend">
          <span>{t('legendTaken')}</span>
          {hasBreak ? <span>{t('legendBreak')}</span> : null}
        </p>
      </Card>
    </section>
  );
}

function StatTiles({ pill }: { pill: PillPack }) {
  const t = useTranslations('contraception.pill');
  const locale = useLocale() as Locale;
  const nextPack = formatDayMonth(fromApiDate(pill.nextPackOn), locale);
  return (
    <Card className="ctr-stats">
      <div className="ctr-stat is-streak">
        <span className="ctr-stat-num">{t('streak', { days: formatNumber(pill.streakDays, locale) })}</span>
        <span className="ctr-stat-cap">{t('streakCap')}</span>
      </div>
      <div className="ctr-stat is-next">
        {/* «22 October» / «۳۰ اردیبهشت» don't fit the half-width tile at the stat size — step down instead of wrapping. */}
        <span className={clsx('ctr-stat-num', nextPack.length > 9 && 'is-long')}>{nextPack}</span>
        <span className="ctr-stat-cap">{t('nextPackCap')}</span>
      </div>
    </Card>
  );
}

function RefillRow({ pill, onEdit }: { pill: PillPack; onEdit: () => void }) {
  const t = useTranslations('contraception.pill');
  const locale = useLocale() as Locale;
  const known = pill.packsLeft !== null;
  return (
    <ListGroup className="ctr-refill">
      <ListRow
        icon="box"
        iconTone="bloom"
        title={
          known
            ? t('packsLeft', { count: pill.packsLeft ?? 0, n: formatNumber(pill.packsLeft ?? 0, locale) })
            : t('packsUnknown')
        }
        description={known ? t('refillSub') : t('packsUnknownSub')}
        onClick={onEdit}
      />
    </ListGroup>
  );
}

/** «چند بسته داری؟» — re-saves the method with a new `packs_left` (the refill reminder follows). */
function PacksSheet({ method, onClose }: { method: ContraceptionMethod; onClose: () => void }) {
  const t = useTranslations('contraception.pill');
  const ts = useTranslations('contraception.setup');
  const locale = useLocale() as Locale;
  const save = useSaveContraceptionMethod();
  const [count, setCount] = useState(method.packsLeft ?? 1);
  return (
    <AppSheet
      open
      onClose={onClose}
      size="half"
      title={t('packsSheetTitle')}
      footer={
        <div className="ctr-sheet-btns">
          <SecondaryButton block={false} onClick={onClose} disabled={save.isPending}>
            {t('cancel')}
          </SecondaryButton>
          <PrimaryButton
            block={false}
            loading={save.isPending}
            onClick={() => save.mutate({ ...methodToPayload(method), packs_left: count }, { onSuccess: onClose })}
          >
            {t('save')}
          </PrimaryButton>
        </div>
      }
    >
      <NumberStepper
        label={t('packsStepper')}
        unit={t('packsUnit')}
        value={count}
        min={PACKS_LEFT_RANGE.min}
        max={PACKS_LEFT_RANGE.max}
        onChange={setCount}
        decrementLabel={ts('decrease', { what: t('packsStepper') })}
        incrementLabel={ts('increase', { what: t('packsStepper') })}
        locale={locale}
      />
      {save.isError ? (
        <p className="ctr-error is-inline" role="alert">
          {getApiSaveErrorMessage(save.error, ts('saveError'))}
        </p>
      ) : null}
    </AppSheet>
  );
}

/** A long-acting / barrier method: what is saved, and the way to its reminders (CB-CONTRA-03). */
function MethodSummary({ method }: { method: ContraceptionMethod }) {
  const t = useTranslations('contraception');
  const router = useRouter();
  const look = METHOD_LOOK[method.method];
  return (
    <>
      <Card className="ctr-today">
        <div className="ctr-today-head">
          <IconCircle icon={look.icon} tone={look.tone} size="lg" />
          <div className="ctr-today-text">
            <p className="ctr-today-cap">{t('other.current')}</p>
            <p className="ctr-today-title is-small">{t(`setup.methods.${method.method}`)}</p>
          </div>
        </div>
        <SecondaryButton onClick={() => router.push('/contraception/setup')}>{t('other.change')}</SecondaryButton>
      </Card>
      {hasMethodReminders(method.method) ? (
        <ListGroup className="ctr-refill">
          <ListRow
            icon="bellRing"
            iconTone="brand"
            title={t('other.reminders')}
            description={t('other.remindersSub')}
            onClick={() => router.push('/contraception/other')}
          />
        </ListGroup>
      ) : null}
    </>
  );
}
