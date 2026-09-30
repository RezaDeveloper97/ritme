'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState, type ReactNode } from 'react';

import { useUserMode } from '@/entities/message';
import { useUserProfile } from '@/entities/user';
import { useLogout } from '@/features/auth';
import { QuickEditSheet, type QuickEditField } from '@/features/edit-profile';
import { DeleteAccountConfirm } from '@/features/manage-account';
import { localizeHref, useRouter, type Locale } from '@/shared/i18n';
import { formatLongDate, formatNumber } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import {
  Icon,
  ListGroup,
  ListRow,
  ScreenHeader,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
} from '@/shared/ui';

import { MeErrorCard } from './MeErrorCard';
import { firstLetter, groupMobile } from './me-format';

/** One labelled value with an action at the end (`nbl_Me_Profile` field card). */
function FieldRow({
  label,
  value,
  action,
  valueDir,
}: {
  label: string;
  value: ReactNode;
  action: ReactNode;
  valueDir?: 'ltr';
}) {
  return (
    <div className="acct-field">
      <div className="acct-field-body">
        <span className="acct-field-label">{label}</span>
        <bdi dir={valueDir} className="acct-field-value">
          {value}
        </bdi>
      </div>
      {action}
    </div>
  );
}

/**
 * Account screen (B-N1-10, `nbl_Me_Profile` / `nbd_Me_Profile`) at
 * `/profile/account`. Name and birth date are editable today (POST /profile);
 * family name, email, phone change, photo and the device list have no API yet
 * and show «به‌زودی» (bloom/QUESTIONS.md). The cycle & health numbers that the
 * old profile tab edited inline live here until `/cycle/settings` (B-N1-09).
 */
export function AccountPage() {
  const t = useTranslations('me');
  const th = useTranslations('profile');
  const router = useRouter();
  const loc = useLocale() as Locale;
  const mounted = useMounted();
  const profileQuery = useUserProfile();
  const { data: userMode } = useUserMode();
  const logout = useLogout();
  const [editing, setEditing] = useState<QuickEditField | null>(null);
  const [deleteOpen, setDeleteOpen] = useState(false);

  const profile = profileQuery.data;
  const health = profile?.health;
  const loading = !mounted || profileQuery.isLoading;
  const soon = <StatusPill tone="neutral">{t('soon')}</StatusPill>;

  const edit = (field: QuickEditField, label: string) => (
    <button
      type="button"
      className="acct-action"
      aria-label={t('account.editLabel', { field: label })}
      onClick={() => setEditing(field)}
    >
      {t('account.edit')}
    </button>
  );

  const dateOrEmpty = (iso: string | null | undefined) =>
    iso ? formatLongDate(new Date(iso), loc) : th('health.empty');
  const daysOrEmpty = (v: number | null | undefined) =>
    v != null ? th('health.days', { days: formatNumber(v, loc) }) : th('health.empty');
  const measureOrEmpty = (key: 'kg' | 'cm', v: number | null | undefined) =>
    v != null ? th(`health.${key}`, { value: formatNumber(v, loc) }) : th('health.empty');

  const handleLogout = () => {
    if (logout.isPending) return;
    logout.mutate(undefined, {
      onSettled: () => window.location.replace(localizeHref('/signup', loc)),
    });
  };

  return (
    <div className="view acct-page">
      <SkyLayer />
      <div className="scroll acct-scroll">
        <ScreenHeader title={t('account.title')} onBack={() => router.push('/profile')} backLabel={t('account.back')} />

        {loading ? (
          <SkeletonGroup label={t('loading')} className="acct-skel">
            <Skeleton shape="circle" className="acct-skel-avatar" />
            <Skeleton shape="card" />
            <Skeleton shape="card" />
          </SkeletonGroup>
        ) : profileQuery.isError ? (
          <MeErrorCard onRetry={() => profileQuery.refetch()} />
        ) : (
          <>
            <div className="acct-photo">
              <span className="acct-avatar" aria-hidden>
                {profile?.name ? firstLetter(profile.name) : <Icon name="user" size={40} />}
                <span className="acct-avatar-cam">
                  <Icon name="camera" size={16} />
                </span>
              </span>
              <span className="acct-photo-hint">{t('account.photoHint')}</span>
            </div>

            <section className="nb-card acct-fields">
              <FieldRow
                label={t('account.name')}
                value={profile?.name || t('account.empty')}
                action={edit('name', t('account.name'))}
              />
              <FieldRow label={t('account.familyName')} value={t('account.empty')} action={soon} />
              <FieldRow
                label={t('account.mobile')}
                value={profile?.mobile ? groupMobile(profile.mobile, loc) : t('account.empty')}
                valueDir="ltr"
                action={soon}
              />
              <FieldRow
                label={t('account.birthday')}
                value={health?.birthday ? formatLongDate(new Date(health.birthday), loc) : t('account.empty')}
                action={edit('birthday', t('account.birthday'))}
              />
              <FieldRow label={t('account.email')} value={t('account.empty')} action={soon} />
            </section>

            <ListGroup>
              <ListRow
                icon="crown"
                iconTone="warm"
                title={t('account.subscription')}
                description={userMode?.isPremium ? t('plus.active') : t('plus.soonSub')}
              />
              <ListRow
                icon="smartphone"
                title={t('account.sessions')}
                description={t('account.sessionsSub')}
                trailing={soon}
              />
            </ListGroup>

            {health ? (
              <ListGroup title={t('account.health')}>
                <ListRow
                  icon="refresh"
                  title={th('health.cycleDuration')}
                  value={daysOrEmpty(health.cycleDuration)}
                  onClick={() => setEditing('cycleDuration')}
                />
                <ListRow
                  icon="drop"
                  iconTone="period"
                  title={th('health.periodDuration')}
                  value={daysOrEmpty(health.periodDuration)}
                  onClick={() => setEditing('periodDuration')}
                />
                <ListRow
                  icon="calendar"
                  title={th('health.lastPeriod')}
                  value={dateOrEmpty(health.lastPeriodStart)}
                  onClick={() => setEditing('lastPeriod')}
                />
                <ListRow
                  icon="scale"
                  iconTone="data"
                  title={th('health.weight')}
                  value={measureOrEmpty('kg', health.weight)}
                  onClick={() => setEditing('weight')}
                />
                <ListRow
                  icon="walk"
                  iconTone="data"
                  title={th('health.height')}
                  value={measureOrEmpty('cm', health.height)}
                  onClick={() => setEditing('height')}
                />
              </ListGroup>
            ) : null}
          </>
        )}

        <button type="button" className="acct-logout" onClick={handleLogout} disabled={logout.isPending}>
          <Icon name="logout" size={18} />
          {logout.isPending ? t('loggingOut') : t('logout')}
        </button>
        <button type="button" className="acct-delete" onClick={() => setDeleteOpen(true)}>
          {t('account.delete')}
        </button>
      </div>

      <QuickEditSheet
        field={editing}
        values={{
          name: profile?.name,
          cycleDuration: health?.cycleDuration,
          periodDuration: health?.periodDuration,
          birthday: health?.birthday,
          lastPeriodStart: health?.lastPeriodStart,
          weight: health?.weight,
          height: health?.height,
        }}
        onClose={() => setEditing(null)}
      />
      {/* Deleting ends the session and lands on /signup (§11). */}
      <DeleteAccountConfirm open={deleteOpen} onClose={() => setDeleteOpen(false)} />
    </div>
  );
}
