'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import { useUserProfile } from '@/entities/user';
import { QuickEditSheet, type QuickEditField } from '@/features/edit-profile';
import { type Locale, useRouter } from '@/shared/i18n';
import { pilot } from '@/shared/pilot';
import { formatLongDate, formatNumber } from '@/shared/lib/date';
import {
  EmptyState,
  ListGroup,
  ListRow,
  NumberStepper,
  PrimaryButton,
  ScreenHeader,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  Switch,
} from '@/shared/ui';

import { useCycleSettings, useUpdateCycleSettings } from '../api/settings';
import {
  clamp,
  CYCLE_RANGE,
  type CycleLengths,
  type CycleSettings,
  type CycleSettingsPatch,
  PERIOD_RANGE,
} from '../model/settings';
import './cys.css';

type Save = (patch: CycleSettingsPatch) => void;

/** Manual lengths are saved after the steppers settle, not on every tap. */
const LENGTH_SAVE_DELAY = 600;

function LoadingState({ label }: { label: string }) {
  return (
    <SkeletonGroup label={label} className="cys-skel">
      <div className="nb-card cys-skel-card">
        <Skeleton width="short" />
        <div className="cys-tiles">
          <Skeleton shape="block" className="cys-skel-tile" />
          <Skeleton shape="block" className="cys-skel-tile" />
        </div>
      </div>
      {[3].map((rows, g) => (
        <div key={g} className="nb-card cys-skel-card">
          <Skeleton width="short" />
          {Array.from({ length: rows }, (_, i) => (
            <div key={i} className="cys-skel-row">
              <Skeleton shape="circle" />
              <span className="cys-skel-text">
                <Skeleton width="medium" />
                <Skeleton width="short" />
              </span>
              <Skeleton shape="block" className="cys-skel-switch" />
            </div>
          ))}
        </div>
      ))}
    </SkeletonGroup>
  );
}

// ── «سیکل تو» ───────────────────────────────────────────────────────────────

function LengthsCard({ lengths, save }: { lengths: CycleLengths; save: Save }) {
  const t = useTranslations('me.cycleSettings');
  const loc = useLocale() as Locale;
  const [draft, setDraft] = useState({ cycle: lengths.cycleLength, period: lengths.periodLength });
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Follow the server while the user is not editing.
  useEffect(() => {
    if (!timer.current) setDraft({ cycle: lengths.cycleLength, period: lengths.periodLength });
  }, [lengths.cycleLength, lengths.periodLength]);
  useEffect(() => () => void (timer.current && clearTimeout(timer.current)), []);

  const change = (field: 'cycle' | 'period', value: number) => {
    const next = { ...draft, [field]: value };
    setDraft(next);
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(() => {
      timer.current = null;
      save({ cycle_length: next.cycle, period_length: next.period });
    }, LENGTH_SAVE_DELAY);
  };

  const caption = lengths.auto ? t('autoCaption') : t('manualCaption');
  const based = lengths.calculated.basedOnCycles;
  const autoDesc = !lengths.auto
    ? t('auto.manual')
    : based
      ? t('auto.basedOn', { count: formatNumber(based, loc) })
      : t('auto.noData');
  const tile = (kind: 'cycle' | 'period', value: number) => (
    <div className={clsx('cys-tile', `is-${kind}`)}>
      <span className="cys-tile-label">{t(kind === 'cycle' ? 'cycleLength' : 'periodLength')}</span>
      <span className="cys-tile-value">
        <span className="cys-tile-num">{formatNumber(value, loc)}</span>
        <span className="cys-tile-unit">{t('days')}</span>
      </span>
      <span className="cys-tile-cap">{caption}</span>
    </div>
  );

  return (
    <ListGroup title={t('yourCycle')} className="cys-card">
      <div className="cys-tiles">
        {tile('cycle', lengths.auto ? lengths.cycleLength : draft.cycle)}
        {tile('period', lengths.auto ? lengths.periodLength : draft.period)}
      </div>
      <ListRow
        id="cys-auto"
        icon="sun"
        iconTone="data"
        iconOutlined
        title={t('auto.title')}
        description={autoDesc}
        trailing={
          <Switch
            checked={lengths.auto}
            labelledBy="cys-auto-title"
            pilot={pilot('cycle-settings.auto')}
            onCheckedChange={(next) =>
              save(
                next
                  ? { lengths_auto: true }
                  : // Going manual pins today's numbers as the manual values.
                    { lengths_auto: false, cycle_length: draft.cycle, period_length: draft.period },
              )
            }
          />
        }
      />
      {lengths.auto ? null : (
        <div className="cys-manual">
          <NumberStepper
            label={t('cycleLength')}
            unit={t('days')}
            value={clamp(draft.cycle, CYCLE_RANGE)}
            min={CYCLE_RANGE.min}
            max={CYCLE_RANGE.max}
            onChange={(v) => change('cycle', v)}
            pilot={pilot('cycle-settings.cycle-length', 'stepper')}
            decrementLabel={t('decrease', { what: t('cycleLength') })}
            incrementLabel={t('increase', { what: t('cycleLength') })}
            locale={loc}
          />
          <NumberStepper
            label={t('periodLength')}
            unit={t('days')}
            value={clamp(draft.period, PERIOD_RANGE)}
            min={PERIOD_RANGE.min}
            max={PERIOD_RANGE.max}
            onChange={(v) => change('period', v)}
            pilot={pilot('cycle-settings.period-length', 'stepper')}
            decrementLabel={t('decrease', { what: t('periodLength') })}
            incrementLabel={t('increase', { what: t('periodLength') })}
            locale={loc}
          />
        </div>
      )}
    </ListGroup>
  );
}

// ── «اطلاعات سلامت» ─────────────────────────────────────────────────────────

type HealthField = Extract<QuickEditField, 'lastPeriod' | 'weight' | 'height'>;

/**
 * Last period start, weight and height — the rest of the «چرخه و سلامت» block
 * that used to sit on `/profile/account`; the cycle / period length rows there
 * are the lengths card above. Each row opens the shared quick-edit sheet
 * (POST /profile). Hidden until the profile has loaded.
 */
function HealthCard() {
  const t = useTranslations('me.cycleSettings');
  const th = useTranslations('profile');
  const loc = useLocale() as Locale;
  const health = useUserProfile().data?.health;
  const [editing, setEditing] = useState<HealthField | null>(null);
  if (!health) return null;

  const measure = (key: 'kg' | 'cm', v: number | null | undefined) =>
    v != null ? th(`health.${key}`, { value: formatNumber(v, loc) }) : th('health.empty');

  return (
    <>
      <ListGroup title={t('health.title')} className="cys-card">
        <ListRow
          icon="calendar"
          iconTone="period"
          iconOutlined
          title={th('health.lastPeriod')}
          value={health.lastPeriodStart ? formatLongDate(new Date(health.lastPeriodStart), loc) : th('health.empty')}
          onClick={() => setEditing('lastPeriod')}
          pilot={pilot('cycle-settings.edit:lastPeriod')}
        />
        <ListRow
          icon="scale"
          iconTone="data"
          iconOutlined
          title={th('health.weight')}
          value={measure('kg', health.weight)}
          onClick={() => setEditing('weight')}
          pilot={pilot('cycle-settings.edit:weight')}
        />
        <ListRow
          icon="walk"
          iconTone="data"
          iconOutlined
          title={th('health.height')}
          value={measure('cm', health.height)}
          onClick={() => setEditing('height')}
          pilot={pilot('cycle-settings.edit:height')}
        />
      </ListGroup>
      <QuickEditSheet
        field={editing}
        values={{ lastPeriodStart: health.lastPeriodStart, weight: health.weight, height: health.height }}
        onClose={() => setEditing(null)}
      />
    </>
  );
}

function SettingsBody({ settings, save }: { settings: CycleSettings; save: Save }) {
  return (
    <>
      <LengthsCard lengths={settings.lengths} save={save} />
      <HealthCard />
    </>
  );
}

/**
 * Cycle settings (B-N1-09, `nbl_Cycle_Settings` / `nbd_Cycle_Settings`) at
 * `/cycle/settings` — the one place for the cycle & health basics: cycle /
 * period length with «خودکار از داده‌ها», plus last period start, weight and
 * height (moved here from `/profile/account`). The cycle reminders live on
 * `/profile/notifications`, the life-stage mode on `/profile/mode`. Changes
 * save at once.
 */
export function CycleSettingsPage() {
  const t = useTranslations('me.cycleSettings');
  const router = useRouter();
  const query = useCycleSettings();
  const update = useUpdateCycleSettings();

  let body;
  if (query.isPending) {
    body = <LoadingState label={t('loading')} />;
  } else if (query.isError || !query.data) {
    body = (
      <EmptyState
        icon="calendar"
        title={t('error.title')}
        body={t('error.body')}
        action={
          <PrimaryButton icon="refresh" loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('error.retry')}
          </PrimaryButton>
        }
      />
    );
  } else {
    body = <SettingsBody settings={query.data} save={(patch) => update.mutate(patch)} />;
  }

  return (
    <div className="view cys-page">
      <SkyLayer />
      <div className="scroll cys-scroll">
        <ScreenHeader
          title={t('title')}
          onBack={() => router.back('/home')}
          backLabel={t('back')}
        />
        {update.isError ? (
          <p className="cys-error" role="alert">
            {t('saveError')}
          </p>
        ) : null}
        {body}
        <p className="cys-note">{t('disclaimer')}</p>
      </div>
    </div>
  );
}
