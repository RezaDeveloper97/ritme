'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import { useUserMode } from '@/entities/message';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import {
  EmptyState,
  Icon,
  type IconName,
  ListGroup,
  ListRow,
  NumberStepper,
  PrimaryButton,
  ScreenHeader,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
  Switch,
  type Tone,
} from '@/shared/ui';

import { useCycleSettings, useUpdateCycleSettings } from '../api/settings';
import {
  clamp,
  CYCLE_RANGE,
  type CycleLengths,
  type CycleSettings,
  type CycleSettingsPatch,
  DAYS_BEFORE_RANGE,
  displayClock,
  isClock,
  PERIOD_RANGE,
  type Reminder,
  type ReminderCode,
} from '../model/settings';

type Save = (patch: CycleSettingsPatch) => void;

const REMINDER_LOOK: Record<ReminderCode, { icon: IconName; tone: Tone }> = {
  before_period: { icon: 'drop', tone: 'period' },
  pms: { icon: 'sparkle', tone: 'brand' },
  fertile_window: { icon: 'heart', tone: 'warm' },
  daily_log: { icon: 'pencil', tone: 'data' },
  pill: { icon: 'pill', tone: 'bloom' },
};

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
      {[5, 3].map((rows, g) => (
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
            decrementLabel={t('decrease', { what: t('periodLength') })}
            incrementLabel={t('increase', { what: t('periodLength') })}
            locale={loc}
          />
        </div>
      )}
    </ListGroup>
  );
}

// ── «یادآورها» ──────────────────────────────────────────────────────────────

function reminderSub(r: Reminder, t: ReturnType<typeof useTranslations>, loc: Locale): string {
  const time = formatNumber(displayClock(r.time), loc);
  switch (r.code) {
    case 'before_period':
      return t('reminders.items.before_period.sub', { days: formatNumber(r.daysBefore ?? 2, loc), time });
    case 'pms':
      return r.cycleDay
        ? t('reminders.items.pms.sub', { day: formatNumber(r.cycleDay, loc) })
        : t('reminders.items.pms.subUnknown');
    case 'fertile_window':
      return t('reminders.items.fertile_window.sub');
    default:
      return t(`reminders.items.${r.code}.sub`, { time });
  }
}

function ReminderEditor({ reminder, save, onDone }: { reminder: Reminder; save: Save; onDone: () => void }) {
  const t = useTranslations('me.cycleSettings');
  const loc = useLocale() as Locale;
  const onTime = (value: string) => {
    if (isClock(value) && value !== reminder.time) save({ reminders: { [reminder.code]: { time: value } } });
  };
  return (
    <div id={`cys-edit-${reminder.code}`} className="cys-edit">
      {reminder.code === 'before_period' ? (
        <NumberStepper
          className="cys-edit-days"
          label={t('reminders.daysBefore')}
          unit={t('days')}
          value={clamp(reminder.daysBefore ?? 2, DAYS_BEFORE_RANGE)}
          min={DAYS_BEFORE_RANGE.min}
          max={DAYS_BEFORE_RANGE.max}
          onChange={(v) => save({ reminders: { before_period: { days_before: v } } })}
          decrementLabel={t('decrease', { what: t('reminders.daysBefore') })}
          incrementLabel={t('increase', { what: t('reminders.daysBefore') })}
          locale={loc}
        />
      ) : null}
      <div className="cys-edit-row">
        <label className="cys-time">
          <span className="cys-time-label">{t('reminders.time')}</span>
          <input
            type="time"
            className="cys-time-input"
            defaultValue={reminder.time}
            onBlur={(e) => onTime(e.target.value)}
            onChange={(e) => onTime(e.target.value)}
          />
        </label>
        <button type="button" className="cys-done" onClick={onDone}>
          {t('reminders.done')}
        </button>
      </div>
    </div>
  );
}

function RemindersCard({ reminders, save }: { reminders: Reminder[]; save: Save }) {
  const t = useTranslations('me.cycleSettings');
  const loc = useLocale() as Locale;
  const [editing, setEditing] = useState<ReminderCode | null>(null);

  if (reminders.length === 0) {
    return (
      <ListGroup title={t('reminders.title')} className="cys-card">
        <p className="cys-empty">{t('reminders.empty')}</p>
      </ListGroup>
    );
  }

  return (
    <ListGroup title={t('reminders.title')} className="cys-card">
      {reminders.map((r) => {
        const look = REMINDER_LOOK[r.code];
        const open = editing === r.code;
        const sub = reminderSub(r, t, loc);
        return (
          <div key={r.code} className="cys-rem">
            <ListRow
              id={`cys-${r.code}`}
              icon={look.icon}
              iconTone={look.tone}
              iconOutlined
              title={t(`reminders.items.${r.code}.title`)}
              description={
                <button
                  type="button"
                  className="cys-sub"
                  aria-expanded={open}
                  aria-controls={`cys-edit-${r.code}`}
                  aria-label={`${t('reminders.edit')} — ${sub}`}
                  onClick={() => setEditing(open ? null : r.code)}
                >
                  {sub}
                  <Icon name="pencil" size={12} className="cys-sub-icon" />
                </button>
              }
              trailing={
                <Switch
                  checked={r.enabled}
                  labelledBy={`cys-${r.code}-title`}
                  onCheckedChange={(next) => save({ reminders: { [r.code]: { enabled: next } } })}
                />
              }
            />
            {open ? <ReminderEditor reminder={r} save={save} onDone={() => setEditing(null)} /> : null}
          </div>
        );
      })}
    </ListGroup>
  );
}

// ── «حالت ریتمی» ────────────────────────────────────────────────────────────

type ModeKey = 'cycle' | 'ttc' | 'pregnancy';

const MODES: readonly { key: ModeKey; icon: IconName; tone: Tone }[] = [
  { key: 'cycle', icon: 'drop', tone: 'period' },
  { key: 'ttc', icon: 'target', tone: 'warm' },
  { key: 'pregnancy', icon: 'user', tone: 'bloom' },
];

/**
 * The life-stage modes. The full switcher is B-N2-03 (/profile/mode); until
 * then the pregnancy row reuses the existing pregnancy setup and the other
 * targets are shown as «به‌زودی».
 */
function ModesCard() {
  const t = useTranslations('me.cycleSettings');
  const router = useRouter();
  const { data } = useUserMode();
  const current: ModeKey = data?.mode === 'pregnancy' ? 'pregnancy' : data?.isTtc ? 'ttc' : 'cycle';
  return (
    <ListGroup title={t('modes.title')} className="cys-card">
      {MODES.map((m) => {
        const active = m.key === current;
        const title = t(`modes.${m.key}`);
        if (active) {
          return (
            <ListRow
              key={m.key}
              icon={m.icon}
              iconTone={m.tone}
              iconOutlined
              title={title}
              description={t('modes.active')}
              trailing={
                <span className="cys-check" role="img" aria-label={t('modes.active')}>
                  <Icon name="check" size={14} strokeWidth={2.6} />
                </span>
              }
            />
          );
        }
        if (m.key === 'pregnancy' && current !== 'pregnancy') {
          return (
            <ListRow
              key={m.key}
              icon={m.icon}
              iconTone={m.tone}
              iconOutlined
              title={title}
              description={t('modes.switch')}
              onClick={() => router.push('/pregnancy/setup')}
            />
          );
        }
        return (
          <ListRow
            key={m.key}
            icon={m.icon}
            iconTone={m.tone}
            iconOutlined
            title={title}
            description={t('modes.switch')}
            trailing={<StatusPill tone="neutral">{t('modes.soon')}</StatusPill>}
          />
        );
      })}
    </ListGroup>
  );
}

function SettingsBody({ settings, save }: { settings: CycleSettings; save: Save }) {
  return (
    <>
      <LengthsCard lengths={settings.lengths} save={save} />
      <RemindersCard reminders={settings.reminders} save={save} />
      <ModesCard />
    </>
  );
}

/**
 * Cycle settings (B-N1-09, `nbl_Cycle_Settings` / `nbd_Cycle_Settings`) at
 * `/cycle/settings`: cycle / period length with «خودکار از داده‌ها», the cycle
 * reminders (switch + time; stored as notification preferences, which every
 * push sender honours) and the life-stage mode. Changes save at once.
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
          subtitle={t('subtitle')}
          onBack={() => router.push('/home')}
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
