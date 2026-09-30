'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useId, useState } from 'react';

import { type Locale, useRouter } from '@/shared/i18n';
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

import { useNotificationSettings, useUpdateNotificationSettings } from '../api/settings';
import { displayClock, isClock, type NotificationSettings, type SettingsPatch } from '../model/settings';

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
  const loc = useLocale() as Locale;
  const [editing, setEditing] = useState(false);
  const editorId = useId();
  const { quietHours: quiet } = settings;

  const clock = (value: string) => formatNumber(displayClock(value), loc);
  const onClock = (field: 'start' | 'end', value: string) => {
    if (isClock(value) && value !== quiet[field]) save({ quiet_hours: { [field]: value } });
  };

  return (
    <>
      {settings.groups.map((group) => (
        <section key={group.code} className="ntf-sec" aria-labelledby={`ntf-g-${group.code}`}>
          <h2 id={`ntf-g-${group.code}`} className="ntf-label">
            {t(`groups.${group.code}`)}
          </h2>
          <ListGroup className="ntf-list">
            {group.items.map((item) => (
              <ListRow
                key={item.code}
                id={`ntf-${item.code}`}
                title={t(`items.${item.code}.title`)}
                description={t(`items.${item.code}.sub`)}
                trailing={
                  <Switch
                    checked={item.enabled}
                    labelledBy={`ntf-${item.code}-title`}
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
                  defaultValue={quiet.end}
                  onBlur={(e) => onClock('end', e.target.value)}
                  onChange={(e) => onClock('end', e.target.value)}
                />
              </label>
              <button type="button" className="ntf-time-done" onClick={() => setEditing(false)}>
                {t('quiet.done')}
              </button>
            </div>
          ) : null}
        </ListGroup>
      </section>

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
        <ScreenHeader title={t('title')} onBack={() => router.push('/profile')} backLabel={t('back')} />
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
