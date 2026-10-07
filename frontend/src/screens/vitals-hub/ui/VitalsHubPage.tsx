'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  ClassPill,
  ReadingRow,
  VITAL_LOOK,
  VitalIcon,
  vitalAddPath,
  vitalReportPath,
  useVitalFormat,
  useVitalsHub,
  type GlucoseUnit,
  type PlanWeek,
  type VitalReading,
  type VitalType,
} from '@/entities/vital';
import { Link, useDirection, type Locale } from '@/shared/i18n';
import { formatWeekday, formatWeekdayDayMonth, fromApiDate, today } from '@/shared/lib/date';
import { Card, EmptyState, HubHeader, Icon, PrimaryButton, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

import { PlanEditorSheet } from './PlanEditorSheet';

/**
 * `/vitals` — «علائم حیاتی» hub (nbl_/nbd_Vitals_Hub): the latest reading of
 * each type with its class, this week's measuring plan, recent readings
 * (timed + read-only log-sheet values) and the three add buttons. A hub: the
 * bottom nav shows (خدمات tab).
 */
export function VitalsHubPage() {
  const t = useTranslations('vitals');
  const locale = useLocale() as Locale;
  const hub = useVitalsHub();
  const [planOpen, setPlanOpen] = useState(false);

  let body;
  if (hub.isPending) {
    body = (
      <SkeletonGroup label={t('common.loading')} className="vt-hub">
        <Skeleton shape="card" className="vt-skel-hero" />
        <div className="vt-latest">
          <Skeleton shape="card" />
          <Skeleton shape="card" />
        </div>
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (hub.isError) {
    body = (
      <Card>
        <EmptyState
          icon="warning"
          title={t('common.loadError')}
          action={
            <PrimaryButton icon="refresh" loading={hub.isFetching} onClick={() => void hub.refetch()}>
              {t('common.retry')}
            </PrimaryButton>
          }
        />
      </Card>
    );
  } else {
    const h = hub.data;
    body = (
      <div className="vt-hub">
        <div className="vt-latest">
          {(['bp', 'hr', 'glucose'] as const).map((type) => (
            <LatestCard key={type} type={type} reading={h.latest[type]} />
          ))}
        </div>
        <PlanCard week={h.plan} onEdit={() => setPlanOpen(true)} />
        <RecentCard readings={h.recent} glucoseUnit={h.latest.glucose?.glucose?.unit} />
      </div>
    );
  }

  return (
    <div className="view vt-screen vt-hub-screen">
      <SkyLayer />
      <div className="scroll vt-scroll">
        <HubHeader
          date={formatWeekdayDayMonth(today(), locale)}
          greeting={t('hub.title')}
          actions={
            <Link href="/record" className="nb-hbtn is-soft" aria-label={t('hub.record')}>
              <Icon name="note" size={20} strokeWidth={1.8} />
            </Link>
          }
        />
        {body}
      </div>
      <nav className="vt-add-bar" aria-label={t('hub.addGroup')}>
        {(['bp', 'hr', 'glucose'] as const).map((type) => (
          <Link key={type} href={vitalAddPath(type)} className={clsx('vt-add-btn', `is-${type}`)} aria-label={t('hub.addLabel', { type: t(`types.${type}`) })}>
            <Icon name="plus" size={16} strokeWidth={2.6} />
            {t(`short.${type}`)}
          </Link>
        ))}
      </nav>
      {planOpen ? <PlanEditorSheet onClose={() => setPlanOpen(false)} /> : null}
      <BottomNav />
    </div>
  );
}

function LatestCard({ type, reading }: { type: VitalType; reading: VitalReading | null }) {
  const t = useTranslations('vitals');
  const f = useVitalFormat();
  const rtl = useDirection() === 'rtl';
  const href = reading ? vitalReportPath(type) : vitalAddPath(type);
  const condition = reading ? f.conditions(reading)[0] : null;
  const verdict = reading?.classification ? f.classLabel(type, reading.classification.code) : null;
  const pill = type === 'glucose' && condition && verdict ? `${condition}${t('common.separator')}${verdict}` : type === 'hr' && condition ? condition : verdict;
  return (
    <Link href={href} className={clsx('nb-card vt-latest-card', `is-${type}`)} aria-label={reading ? t('hub.openReport', { type: t(`types.${type}`) }) : t('hub.addLabel', { type: t(`types.${type}`) })}>
      <span className="vt-latest-head">
        <VitalIcon type={type} />
        <b className="vt-latest-title">{t(`types.${type}`)}</b>
        <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={15} strokeWidth={2} className="vt-chev" />
      </span>
      {reading ? (
        <>
          <span className="vt-latest-value">
            <bdi dir="ltr" className="vt-big">
              {f.value(reading)}
            </bdi>
            <span className="vt-unit-text" dir="ltr">
              {f.unit(reading)}
            </span>
          </span>
          <span className="vt-latest-foot">
            {pill ? (
              <ClassPill type={type} cls={reading.classification}>
                {pill}
              </ClassPill>
            ) : null}
            <span className="vt-latest-when">{f.when(reading)}</span>
          </span>
        </>
      ) : (
        <span className="vt-latest-empty">
          <span>{t('hub.latestEmpty')}</span>
          <span className={clsx('vt-latest-add', `nb-tone-${VITAL_LOOK[type].tone}`)}>{t('hub.addFirst')}</span>
        </span>
      )}
    </Link>
  );
}

function PlanCard({ week, onEdit }: { week: PlanWeek; onEdit: () => void }) {
  const t = useTranslations('vitals');
  const f = useVitalFormat();
  const locale = useLocale() as Locale;
  return (
    <Card as="section" className="vt-card vt-plan" aria-labelledby="vt-plan-title">
      <div className="vt-card-head">
        <h2 id="vt-plan-title" className="vt-card-title">
          {t('hub.planTitle')}
        </h2>
        {week.items.length ? <span className="vt-card-aside">{t('hub.planCount', { done: f.num(week.done), planned: f.num(week.planned) })}</span> : null}
      </div>
      {week.items.length ? (
        <ul className="vt-plan-rows">
          {week.items.map((item) => {
            const label = t('hub.planRow', { type: t(`short.${item.type}`), slot: t(`slots.${item.slot}` as 'slots.morning') });
            return (
              <li key={`${item.type}-${item.slot}`} className="vt-plan-row">
                <span className="vt-plan-label">{label}</span>
                <ol className={clsx('vt-plan-days', `nb-tone-${VITAL_LOOK[item.type].tone}`)} aria-label={label}>
                  {item.days.map((d) => {
                    const day = formatWeekday(fromApiDate(d.date), locale);
                    const name = t(d.state === 'done' ? 'hub.dayDone' : d.state === 'missed' ? 'hub.dayMissed' : 'hub.dayDue', { day });
                    return (
                      <li key={d.date} className={clsx('vt-plan-day', `is-${d.state}`)} aria-label={name}>
                        {d.state === 'done' ? <Icon name="check" size={12} strokeWidth={3} /> : null}
                      </li>
                    );
                  })}
                </ol>
              </li>
            );
          })}
        </ul>
      ) : (
        <p className="vt-card-note">{t('hub.planEmpty')}</p>
      )}
      <button type="button" className="vt-link-btn" onClick={onEdit}>
        <Icon name={week.items.length ? 'pencil' : 'plus'} size={16} />
        {t(week.items.length ? 'hub.planEdit' : 'hub.planSetup')}
      </button>
    </Card>
  );
}

/** `glucoseUnit` = the unit of the latest glucose card, so the list shows every glucose value in it. */
function RecentCard({ readings, glucoseUnit }: { readings: readonly VitalReading[]; glucoseUnit?: GlucoseUnit }) {
  const t = useTranslations('vitals');
  return (
    <Card as="section" className="vt-card vt-list-card" aria-labelledby="vt-recent-title">
      <div className="vt-card-head is-tight">
        <h2 id="vt-recent-title" className="vt-card-title">
          {t('hub.recentTitle')}
        </h2>
        {readings.length ? (
          <Link href={vitalReportPath(readings[0].type)} className="vt-head-link">
            {t('hub.all')}
          </Link>
        ) : null}
      </div>
      {readings.length ? (
        <ul className="vt-rows">
          {readings.map((r, i) => (
            <ReadingRow key={r.id ?? `log-${r.type}-${r.date}-${i}`} reading={r} withType glucoseUnit={glucoseUnit} />
          ))}
        </ul>
      ) : (
        <p className="vt-card-note">{t('hub.recentEmpty')}</p>
      )}
    </Card>
  );
}
