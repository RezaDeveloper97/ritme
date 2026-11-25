'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useId, useState } from 'react';

import { isFertilitySuppressed, useCycleToday } from '@/entities/cycle';
import { findEntitlement, usePlusStatus } from '@/entities/plus';
import { DeviceNotificationsRow } from '@/features/push-notifications';
import { pilot } from '@/shared/pilot';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import {
  EmptyState,
  Icon,
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
import { PlusBadge } from '@/shared/ui/plus-gate';

import {
  useCycleReminders,
  useNotificationSettings,
  useUpdateCycleReminders,
  useUpdateNotificationSettings,
} from '../api/settings';
import { clamp, DAYS_BEFORE_RANGE, type Reminder, type RemindersPatch, TIMED_REMINDERS } from '../model/reminders';
import { type CategoryCode, displayClock, isClock, type NotificationSettings, type SettingsPatch } from '../model/settings';
import '@/shared/ui/styles/ntf.css';

function LoadingState({ label }: { label: string }) {
  return (
    <SkeletonGroup label={label} className="ntf-skel">
      {[4, 4, 3, 1].map((rows, g) => (
        <div key={g} className="ntf-skel-sec">
          <Skeleton width="short" />
          <div className="nb-card ntf-skel-card">
            {Array.from({ length: rows }, (_, i) => (
              <div key={i} className="ntf-skel-row">
                <span className="ntf-skel-text">
                  <Skeleton width="medium" />
                  <Skeleton width="short" />
                </span>
                <Skeleton shape="block" className="ntf-skel-switch" />
              </div>
            ))}
          </div>
        </div>
      ))}
    </SkeletonGroup>
  );
}

type SaveReminders = (patch: RemindersPatch) => void;
type Translate = ReturnType<typeof useTranslations>;

const isTimed = (code: string): boolean => (TIMED_REMINDERS as readonly string[]).includes(code);

/** A cycle reminder's subtitle from its real schedule (the old `/cycle/settings` copy). */
function reminderSub(r: Reminder, t: Translate, loc: Locale): string {
  const time = formatNumber(displayClock(r.time), loc);
  switch (r.code) {
    case 'before_period':
      return t('items.before_period.subAt', { days: formatNumber(r.daysBefore ?? 2, loc), time });
    case 'pms':
      return r.cycleDay ? t('items.pms.subAt', { day: formatNumber(r.cycleDay, loc) }) : t('items.pms.subUnknown');
    case 'fertile_window':
      return t('items.fertile_window.sub');
    default:
      return t(`items.${r.code}.subAt`, { time });
  }
}

/** Time (and, before a period, how many days ahead) of one timed cycle reminder. */
function ReminderEditor({
  reminder,
  id,
  save,
  onDone,
}: {
  reminder: Reminder;
  id: string;
  save: SaveReminders;
  onDone: () => void;
}) {
  const t = useTranslations('me.notifSettings');
  const loc = useLocale() as Locale;
  const onTime = (value: string) => {
    if (isClock(value) && value !== reminder.time) save({ [reminder.code]: { time: value } });
  };
  return (
    <div id={id} className="ntf-rem-edit">
      {reminder.code === 'before_period' ? (
        <NumberStepper
          label={t('reminders.daysBefore')}
          unit={t('reminders.days')}
          value={clamp(reminder.daysBefore ?? 2, DAYS_BEFORE_RANGE)}
          min={DAYS_BEFORE_RANGE.min}
          max={DAYS_BEFORE_RANGE.max}
          onChange={(v) => save({ before_period: { days_before: v } })}
          pilot={pilot('notif.days-before', 'stepper')}
          decrementLabel={t('reminders.decrease')}
          incrementLabel={t('reminders.increase')}
          locale={loc}
        />
      ) : null}
      <div className="ntf-quiet-edit">
        <label className="ntf-time">
          <span className="ntf-time-label">{t('reminders.time')}</span>
          <input
            type="time"
            className="ntf-time-input"
            {...pilot(`notif.reminder-time:${reminder.code}`, 'text')}
            defaultValue={reminder.time}
            onBlur={(e) => onTime(e.target.value)}
            onChange={(e) => onTime(e.target.value)}
          />
        </label>
        <button type="button" className="ntf-time-done" onClick={onDone}>
          {t('quiet.done')}
        </button>
      </div>
    </div>
  );
}

/** The subtitle of a timed reminder: a button that opens its editor. */
function ReminderSubButton({
  sub,
  open,
  controls,
  code,
  onToggle,
}: {
  sub: string;
  open: boolean;
  controls: string;
  code: string;
  onToggle: () => void;
}) {
  const t = useTranslations('me.notifSettings');
  return (
    <button
      type="button"
      className="ntf-range"
      aria-expanded={open}
      aria-controls={controls}
      aria-label={`${t('reminders.edit')} — ${sub}`}
      onClick={onToggle}
      {...pilot(`notif.reminder-edit:${code}`)}
    >
      {sub}
      <Icon name="pencil" size={13} className="ntf-range-icon" />
    </button>
  );
}

interface BodyProps {
  settings: NotificationSettings;
  save: (patch: SettingsPatch) => void;
  reminders: Reminder[] | undefined;
  saveReminders: SaveReminders;
}

function SettingsBody({ settings, save, reminders, saveReminders }: BodyProps) {
  const t = useTranslations('me.notifSettings');
  const loc = useLocale() as Locale;
  const [editing, setEditing] = useState(false);
  const [editingReminder, setEditingReminder] = useState<string | null>(null);
  const reminderOf = (code: string) => reminders?.find((r) => r.code === code);
  const pill = reminderOf('pill');
  const subOf = (code: CategoryCode) => {
    const r = reminderOf(code);
    return r ? reminderSub(r, t, loc) : t(`items.${code}.sub`);
  };
  /** Subtitle of a category row: the editable schedule for a timed cycle reminder. */
  const description = (code: string, fallback: string) => {
    const r = reminderOf(code);
    if (!r || !isTimed(code)) return fallback;
    return (
      <ReminderSubButton
        sub={reminderSub(r, t, loc)}
        open={editingReminder === code}
        controls={`ntf-edit-${code}`}
        code={code}
        onToggle={() => setEditingReminder(editingReminder === code ? null : code)}
      />
    );
  };
  const editor = (code: string) => {
    const r = reminderOf(code);
    return r && editingReminder === code ? (
      <ReminderEditor reminder={r} id={`ntf-edit-${code}`} save={saveReminders} onDone={() => setEditingReminder(null)} />
    ) : null;
  };
  // §2.40: while fertility is suppressed (hormonal contraception, postpartum) there is no fertile window to remind about.
  // The server-side sender still needs to skip it too (follow-up); here the switch just isn't offered.
  const suppressed = isFertilitySuppressed(useCycleToday().data?.cycleView ?? null);
  // Plus upsell: shown once the entitlement is known (no row yet = status loading/unsupported).
  const sms = findEntitlement(usePlusStatus().data, 'plus.sms_reminders');
  const editorId = useId();
  const { quietHours: quiet } = settings;

  const clock = (value: string) => formatNumber(displayClock(value), loc);
  const onClock = (field: 'start' | 'end', value: string) => {
    if (isClock(value) && value !== quiet[field]) save({ quiet_hours: { [field]: value } });
  };

  return (
    <>
      <DeviceNotificationsRow />
      {settings.groups.map((group) => (
        <section key={group.code} className="ntf-sec" aria-labelledby={`ntf-g-${group.code}`}>
          <h2 id={`ntf-g-${group.code}`} className="ntf-label">
            {t(`groups.${group.code}`)}
          </h2>
          <ListGroup className="ntf-list">
            {group.items
              .filter((item) => !(suppressed && item.code === 'fertile_window'))
              .map((item) => (
                <div key={item.code} className="ntf-rem">
                  <ListRow
                    id={`ntf-${item.code}`}
                    title={t(`items.${item.code}.title`)}
                    description={description(item.code, subOf(item.code))}
                    trailing={
                      <Switch
                        checked={item.enabled}
                        labelledBy={`ntf-${item.code}-title`}
                        pilot={pilot(`notif.category:${item.code}`)}
                        onCheckedChange={(next) => save({ categories: { [item.code]: next } })}
                      />
                    }
                  />
                  {editor(item.code)}
                </div>
              ))}
            {/* «قرص پیشگیری» has no category row of its own; it is switched and timed through the cycle reminders. */}
            {group.code === 'cycle' && pill ? (
              <div className="ntf-rem">
                <ListRow
                  id="ntf-pill"
                  title={t('items.pill.title')}
                  description={description('pill', reminderSub(pill, t, loc))}
                  trailing={
                    <Switch
                      checked={pill.enabled}
                      labelledBy="ntf-pill-title"
                      pilot={pilot('notif.reminder:pill')}
                      onCheckedChange={(next) => saveReminders({ pill: { enabled: next } })}
                    />
                  }
                />
                {editor('pill')}
              </div>
            ) : null}
          </ListGroup>
        </section>
      ))}

      <section className="ntf-sec" aria-labelledby="ntf-g-quiet">
        <h2 id="ntf-g-quiet" className="ntf-label">
          {t('groups.quiet')}
        </h2>
        <ListGroup className="ntf-list">
          <ListRow
            id="ntf-quiet"
            title={t('quiet.title')}
            description={
              quiet.enabled ? (
                <button
                  type="button"
                  className="ntf-range"
                  aria-expanded={editing}
                  aria-controls={editorId}
                  aria-label={`${t('quiet.edit')} — ${t('quiet.range', { start: clock(quiet.start), end: clock(quiet.end) })}`}
                  onClick={() => setEditing((v) => !v)}
                  {...pilot('notif.quiet-edit')}
                >
                  {t('quiet.range', { start: clock(quiet.start), end: clock(quiet.end) })}
                  <Icon name="pencil" size={13} className="ntf-range-icon" />
                </button>
              ) : (
                t('quiet.off')
              )
            }
            trailing={
              <Switch
                checked={quiet.enabled}
                labelledBy="ntf-quiet-title"
                pilot={pilot('notif.quiet')}
                onCheckedChange={(next) => {
                  if (!next) setEditing(false);
                  save({ quiet_hours: { enabled: next } });
                }}
              />
            }
          />
          {quiet.enabled && editing ? (
            <div id={editorId} className="ntf-quiet-edit">
              <label className="ntf-time">
                <span className="ntf-time-label">{t('quiet.from')}</span>
                <input
                  type="time"
                  className="ntf-time-input"
                  {...pilot('notif.quiet-start', 'text')}
                  defaultValue={quiet.start}
                  onBlur={(e) => onClock('start', e.target.value)}
                  onChange={(e) => onClock('start', e.target.value)}
                />
              </label>
              <label className="ntf-time">
                <span className="ntf-time-label">{t('quiet.to')}</span>
                <input
                  type="time"
                  className="ntf-time-input"
                  {...pilot('notif.quiet-end', 'text')}
                  defaultValue={quiet.end}
                  onBlur={(e) => onClock('end', e.target.value)}
                  onChange={(e) => onClock('end', e.target.value)}
                />
              </label>
              <button type="button" className="ntf-time-done" onClick={() => setEditing(false)} {...pilot('notif.quiet-done')}>
                {t('quiet.done')}
              </button>
            </div>
          ) : null}
        </ListGroup>
      </section>

      {sms ? (
        sms.allowed ? (
          <aside className="nb-card ntf-neutral" aria-labelledby="ntf-sms-title">
            <Icon name="chat" size={18} className="ntf-neutral-icon" />
            <div className="ntf-neutral-text">
              <span id="ntf-sms-title" className="ntf-neutral-title">
                {t('sms.title')}
              </span>
              <p id="ntf-sms-desc" className="ntf-neutral-body">
                {t('sms.body')}
              </p>
            </div>
            <Switch
              checked={settings.smsReminders}
              labelledBy="ntf-sms-title"
              pilot={pilot('notif.sms')}
              describedBy="ntf-sms-desc"
              onCheckedChange={(next) => save({ sms_reminders: next })}
            />
          </aside>
        ) : (
          <Link href="/plus" className="nb-card ntf-neutral ntf-sms-locked">
            <Icon name="chat" size={18} className="ntf-neutral-icon" />
            <span className="ntf-neutral-text">
              <span className="ntf-neutral-title">{t('sms.title')}</span>
              <span className="ntf-neutral-body">{t('sms.body')}</span>
            </span>
            <PlusBadge label={t('sms.plus')} />
          </Link>
        )
      ) : null}

      <aside className="nb-card ntf-neutral" aria-labelledby="ntf-neutral-title">
        <Icon name="lock" size={18} className="ntf-neutral-icon" />
        <div className="ntf-neutral-text">
          <span id="ntf-neutral-title" className="ntf-neutral-title">
            {t('neutral.title')}
          </span>
          <p id="ntf-neutral-desc" className="ntf-neutral-body">
            {settings.neutralCopy ? t('neutral.on') : t('neutral.off')}
          </p>
        </div>
        <Switch
          checked={settings.neutralCopy}
          labelledBy="ntf-neutral-title"
          pilot={pilot('notif.neutral-copy')}
          describedBy="ntf-neutral-desc"
          onCheckedChange={(next) => save({ neutral_copy: next })}
        />
      </aside>
    </>
  );
}

/**
 * Notifications & reminders (B-N1-11, `nbl_Me_Notifications` /
 * `nbd_Me_Notifications`) at `/profile/notifications`: per-category switches
 * (cycle · health · other) — the cycle reminders with their time / days before
 * and the pill reminder, merged in from `/cycle/settings` — a quiet-hours window and «متن خنثی» — neutral
 * lock-screen copy. Every push the server sends goes through these settings
 * (backend-go internal/notifications). Switches save at once.
 */
export function NotificationSettingsPage() {
  const t = useTranslations('me.notifSettings');
  const router = useRouter();
  const query = useNotificationSettings();
  const update = useUpdateNotificationSettings();
  // Optional: the rows still work (switch only) if the cycle reminders fail to load.
  const reminders = useCycleReminders();
  const updateReminders = useUpdateCycleReminders();

  let body;
  if (query.isPending) {
    body = <LoadingState label={t('loading')} />;
  } else if (query.isError || !query.data) {
    body = (
      <EmptyState
        icon="bell"
        title={t('error.title')}
        body={t('error.body')}
        action={
          <PrimaryButton icon="refresh" loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('error.retry')}
          </PrimaryButton>
        }
      />
    );
  } else if (query.data.groups.length === 0) {
    body = <EmptyState icon="bell" title={t('empty.title')} body={t('empty.body')} />;
  } else {
    body = (
      <SettingsBody
        settings={query.data}
        save={(patch) => update.mutate(patch)}
        reminders={reminders.data}
        saveReminders={(patch) => updateReminders.mutate(patch)}
      />
    );
  }

  return (
    <div className="view ntf-page">
      <SkyLayer />
      <div className="scroll ntf-scroll">
        <ScreenHeader title={t('title')} onBack={() => router.push('/profile')} backLabel={t('back')} />
        {update.isError || updateReminders.isError ? (
          <p className="ntf-error" role="alert">
            {t('saveError')}
          </p>
        ) : null}
        {body}
      </div>
    </div>
  );
}
