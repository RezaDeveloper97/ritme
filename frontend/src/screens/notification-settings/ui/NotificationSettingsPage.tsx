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
  PrimaryButton,
  ScreenHeader,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  Switch,
} from '@/shared/ui';
import { PlusBadge } from '@/shared/ui/plus-gate';

import { useNotificationSettings, usePmsReminderDay, useUpdateNotificationSettings } from '../api/settings';
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

interface BodyProps {
  settings: NotificationSettings;
  save: (patch: SettingsPatch) => void;
}

function SettingsBody({ settings, save }: BodyProps) {
  const t = useTranslations('me.notifSettings');
  const tc = useTranslations('me.cycleSettings');
  const loc = useLocale() as Locale;
  const [editing, setEditing] = useState(false);
  // PMS row: the real reminder day from cycle settings, as `/cycle/settings` shows it (n1-stage B-1).
  const pmsDay = usePmsReminderDay().data ?? null;
  const subOf = (code: CategoryCode) =>
    code === 'pms'
      ? pmsDay
        ? tc('reminders.items.pms.sub', { day: formatNumber(pmsDay, loc) })
        : tc('reminders.items.pms.subUnknown')
      : t(`items.${code}.sub`);
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
              <ListRow
                key={item.code}
                id={`ntf-${item.code}`}
                title={t(`items.${item.code}.title`)}
                description={subOf(item.code)}
                trailing={
                  <Switch
                    checked={item.enabled}
                    labelledBy={`ntf-${item.code}-title`}
                    pilot={pilot(`notif.category:${item.code}`)}
                    onCheckedChange={(next) => save({ categories: { [item.code]: next } })}
                  />
                }
              />
            ))}
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
 * (cycle · health · other), a quiet-hours window and «متن خنثی» — neutral
 * lock-screen copy. Every push the server sends goes through these settings
 * (backend-go internal/notifications). Switches save at once.
 */
export function NotificationSettingsPage() {
  const t = useTranslations('me.notifSettings');
  const router = useRouter();
  const query = useNotificationSettings();
  const update = useUpdateNotificationSettings();

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
    body = <SettingsBody settings={query.data} save={(patch) => update.mutate(patch)} />;
  }

  return (
    <div className="view ntf-page">
      <SkyLayer />
      <div className="scroll ntf-scroll">
        <ScreenHeader title={t('title')} onBack={() => router.back('/profile')} backLabel={t('back')} />
        {update.isError ? (
          <p className="ntf-error" role="alert">
            {t('saveError')}
          </p>
        ) : null}
        {body}
      </div>
    </div>
  );
}
